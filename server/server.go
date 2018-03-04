package main

/*

geth --dev console --ws --networkid 1337

*/

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/big"
	_ "net/http"
	"time"
	"log"
	"encoding/hex"
	"encoding/json"
	
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/crypto/sha3"

	_ "github.com/gin-gonic/gin"
	"fmt"
)

func assert(err error) {
	if err != nil {
		panic("Failed: " + err.Error())
	}
}

func sha3of(s string) string {
	sha := sha3.NewKeccak256()
	sha.Write([]byte(s))
	hash := sha.Sum(nil)
	return "0x"+hex.EncodeToString(hash)
}

type Web3Client struct {
	Client *ethclient.Client
	Account accounts.Account
	Ks *keystore.KeyStore
	Abi abi.ABI
	ByteCode []byte
	ContractAddress *common.Address
	ReceiptTimeout time.Duration
}

func NewWeb3Client( rpcUrl string, ks *keystore.KeyStore,  account accounts.Account) (*Web3Client,error) {

	var err error

	client, err := ethclient.Dial(rpcUrl);
	if err != nil {
		return nil, err
	}

	return &Web3Client{
		Client : client,
		Ks: ks,
		Account : account,
		ReceiptTimeout: 120 * time.Second,
	}, nil
}

func (b *Web3Client) SetContract(jsonFile string) (error) {

	content, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(content, &fields) ; err != nil {
		return err;
	}

	abivalue := fields["abi"]
	bytecodehex := fields["bytecode"].(string)
	if b.ByteCode,err = hex.DecodeString(bytecodehex[2:]) ; err!= nil {
		return err
	}

	abijson, err := json.Marshal(&abivalue)
	if err != nil {
		return err
	}

	b.Abi, err = abi.JSON(bytes.NewReader(abijson))
	if err != nil {
		return err
	}

	return nil
}

func (b *Web3Client) SetContractAddress(contractAddress common.Address) (error) {
	
	b.ContractAddress = &contractAddress
	return nil
}	

func (b *Web3Client) DeployContract() (error) {

	_,receipt,err := b.SendTransaction(nil,big.NewInt(0),b.ByteCode)
	if err != nil {
		return err
	}

	b.ContractAddress = &receipt.ContractAddress

	return nil
}

func (b *Web3Client) AccountInfo() (string, error) {

	address := b.Account.Address.Hex()
	ctx := context.TODO()
	balance, err := b.Client.BalanceAt(ctx,b.Account.Address,nil)
	if err!=nil {
		return "",nil
	}
	return address+"="+balance.String()+" wei",nil
}	
	
func (b *Web3Client) SendTransaction(to *common.Address, value *big.Int, calldata []byte) (*types.Transaction , *types.Receipt, error) {

	var err error
	var tx *types.Transaction
	var receipt *types.Receipt

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	network,err := b.Client.NetworkID(ctx)
	if err != nil {
		return nil,nil,err
	}
	
	gasPrice, err := b.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil,nil,err
	}
	
	nonce, err := b.Client.NonceAt(ctx, b.Account.Address, nil)
	if err != nil {
		return nil,nil,err
	}
	
	gasLimit, err := b.Client.EstimateGas(ctx,ethereum.CallMsg{
		From  : b.Account.Address,
		To    : to,
		Value : value,
		Data  : calldata,
	})

	if err != nil {
		return nil,nil,err
	}

	if to == nil {
		tx = types.NewContractCreation(
			nonce,             // nonce int64
			value,             // amount *big.Int
			gasLimit,          // gasLimit *big.Int
			gasPrice,          // gasPrice *big.Int
			calldata,          // data []byte
		)	
	} else {
		tx = types.NewTransaction(
			nonce,             // nonce int64
			*to, 			   // to common.Address
			value,             // amount *big.Int
			gasLimit,          // gasLimit *big.Int
			gasPrice,          // gasPrice *big.Int
			calldata,          // data []byte
		)	
	}

	if tx, err = b.Ks.SignTx(b.Account, tx, network) ; err != nil {
		return nil,nil,err
	}

	if err = b.Client.SendTransaction(ctx, tx) ; err != nil {
		return nil,nil,err
	}

	log.Println("Sent transaction ", tx.Hash().Hex())

	start := time.Now()
	for receipt == nil && time.Now().Sub(start) < b.ReceiptTimeout {
		receipt, err = b.Client.TransactionReceipt(ctx, tx.Hash())
		if receipt == nil { 
			time.Sleep(1000 * time.Millisecond)
		}
	}

	return tx, receipt, err		
}


/*
func (b *BridgeClient) DeployContract(contractFile string) (error) {
	
	msgdata, err := b.encoder.Pack("fullExecute", epoch,txid,data,sigs)
	if err != nil {
		return "",err
	}

	return b.sendTransaction(b.contractAddress,  big.NewInt(0), msgdata)
}

func (b *BridgeClient) FullExecute(epoch uint, txid [32]byte, data []byte, sigs [][32]byte) (string , error) {

	msgdata, err := b.encoder.Pack("fullExecute", epoch,txid,data,sigs)
	if err != nil {
		return "",err
	}

	return b.sendTransaction(b.contractAddress,  big.NewInt(0), msgdata)
}

func (b *BridgeClient) PartialExecute(epoch uint, txid []byte, data []byte, sigs [][32]byte) (string, error) {

	// function partialExecute(uint256 _epoch, bytes32 _txid, bytes _data, uint8 _v, bytes32 _r, bytes32 _s) public {
	calldata, err := b.encoder.Pack("partialExecute", epoch,txid,data,sigs)
	if err != nil {
		return "",err
	}

	return b.sendTransaction(b.contractAddress,  big.NewInt(0), calldata)
} 
*/

type GometParentHandler struct {
	*Web3Client
}

func NewGometParentHandler( rpcUrl string, ks *keystore.KeyStore,  account accounts.Account) (*GometParentHandler,error) {
	web3client,err := NewWeb3Client(rpcUrl,ks,account)
	return &GometParentHandler{web3client}, err
}


func (b *GometParentHandler) CallLock(value *big.Int) error {
	msgdata, err := b.Abi.Pack("lock")
	if err != nil {
		return err
	}
	_,_,err = b.SendTransaction(b.ContractAddress,  value, msgdata)
	return err
}

func (b *GometParentHandler) handleLockEvent(data []byte)  {

	type LogLockEvent struct {
		From common.Address
		Value *big.Int
	}

	var logLockEvent LogLockEvent
	err := b.Abi.Unpack(&logLockEvent,"LogLock",data)
	if err != nil {
		fmt.Println("Error",err)
	} else {
		fmt.Printf("Unpacked %#v",logLockEvent)				
	}	
}
	
func (b *GometParentHandler) ListenEvents() error {

	LogLockEventSignature := sha3of("LogLock(address,uint256)")

	ctx := context.Background()
	ch := make(chan types.Log)

	query := ethereum.FilterQuery {
		FromBlock: big.NewInt(0),
		ToBlock: big.NewInt(10000000),
		Addresses: []common.Address{*b.ContractAddress},
		Topics: [][]common.Hash{{}},
	}
	_, err := b.Client.SubscribeFilterLogs(ctx,query,ch)
	if err != nil {
		return err
	}
	log.Println("Server started...")

	go func() {
		log.Println("Started listening logs")
		for true {
			log := <-ch
			if log.Removed {
				continue
			}
			fmt.Println("Log from address",log.Address.Hex())
			for c,t := range(log.Topics) {
				fmt.Printf("  Topic[%v]: %v",c,t.Hex())				
			}
			fmt.Printf("  Data: %v",hex.EncodeToString(log.Data))
			if log.Topics[0].Hex() == LogLockEventSignature {
				go b.handleLockEvent(log.Data)
			}
		}
	}()

	return nil
}

func main() {

	// -- open keystore

	keystoreFolder := "keyStore"
	keystorePasswd := "111111"

	var err error
	var account accounts.Account
	
	ks := keystore.NewKeyStore(keystoreFolder, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) == 0 {
		account, err = ks.NewAccount(keystorePasswd)
		assert(err)
	} else {
		account = ks.Accounts()[0]
	}
	assert(ks.Unlock(account, keystorePasswd))

	c,err := NewGometParentHandler(
		"ws://127.0.0.1:8546",
		ks,
		account,
	)
	assert(err)
	
	accountInfo, err := c.AccountInfo()
	assert(err)

	log.Println("ACCOUNT INFO ",accountInfo)

	err = c.SetContract("../build/contracts/GometParent.json")
	assert(err)

	err = c.DeployContract()
	assert(err)

	log.Println("Contract deployed at ",c.ContractAddress.Hex())
	
	err = c.ListenEvents() 
	assert(err)

	<-time.After(time.Second)

	c.CallLock(big.NewInt(1))

	<-time.After(time.Second * 3600)
/*
	err = c.StartServer()
	assert(err)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"success": true} )
	})
	r.Run()
*/

}