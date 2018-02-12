package main

/*

geth --dev console --ws --networkid 1337

*/

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
	"log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/crypto/sha3"

	"github.com/gin-gonic/gin"
	"fmt"
)

func assert(err error) {
	if err != nil {
		panic("Failed: " + err.Error())
	}
}

func sha3of(s string) common.Hash {
	hasher := sha3.NewKeccak256()
	hash := hasher.Sum([]byte(s))
	var r common.Hash
	copy(r[:], hash[:32])
	return r
}


type BridgeClient struct {
	client *ethclient.Client
	account accounts.Account
	ks *keystore.KeyStore
	encoder abi.ABI
	contractAddress common.Address
}

func NewBridgeClient( rpcUrl string, abiFile string,  contractAddress common.Address,
	keystoreFolder string ,  passwd string ) (*BridgeClient,error) {

	var err error

	client, err := ethclient.Dial(rpcUrl);
	if err != nil {
		return nil, err
	}
	
	def, err := ioutil.ReadFile(abiFile)
	if err != nil {
		return nil, err
	}

	encoder, err := abi.JSON(bytes.NewReader(def))
	if err != nil {
		return nil, err
	}

	ks := keystore.NewKeyStore(keystoreFolder, keystore.StandardScryptN, keystore.StandardScryptP)

	if len(ks.Accounts()) != 1 {
		return nil, fmt.Errorf("only one account in keystore is expected")
	}

	err = ks.Unlock(ks.Accounts()[0], passwd)
	if err != nil {
		return nil, err
	}

	return &BridgeClient{
		client : client,
		account : ks.Accounts()[0],
		ks: ks,
		encoder : encoder,
		contractAddress : contractAddress,
	}, nil
}

func (b *BridgeClient) sendTransaction(calldata []byte, amount *big.Int ) (string , error) {

	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	network,err := b.client.NetworkID(ctx)
	if err != nil {
		return "",err
	}

	gasPrice, err := b.client.SuggestGasPrice(ctx)
	if err != nil {
		return "",err
	}

	nonce, err := b.client.NonceAt(ctx, b.account.Address, nil)
	if err != nil {
		return "",err
	}

	gasLimit := big.NewInt(200000)

	tx := types.NewTransaction(
		nonce,             // nonce int64
		b.contractAddress, // to common.Address
		amount,            // amount *big.Int
		gasLimit,          // gasLimit *big.Int
		gasPrice,          // gasPrice *big.Int
		calldata,          // data []byte
	)

	tx, err = b.ks.SignTx(b.account, tx, network)
	if err != nil {
		return "",err
	}

	err = b.client.SendTransaction(ctx, tx)
	if err != nil {
		return "",err
	}

	return tx.Hash().Hex(), nil
}

func (b *BridgeClient) FullExecute(epoch uint, txid [32]byte, data []byte, sigs [][32]byte) (string , error) {

	msgdata, err := b.encoder.Pack("fullExecute", epoch,txid,data,sigs)
	if err != nil {
		return "",err
	}

	return b.sendTransaction(msgdata,big.NewInt(0))
}

func (b *BridgeClient) PartialExecute(epoch uint, txid []byte, data []byte, sigs [][32]byte) (string, error) {

	// function partialExecute(uint256 _epoch, bytes32 _txid, bytes _data, uint8 _v, bytes32 _r, bytes32 _s) public {
	calldata, err := b.encoder.Pack("partialExecute", epoch,txid,data,sigs)
	if err != nil {
		return "",err
	}

	return b.sendTransaction(calldata,big.NewInt(0))
} 

func (b *BridgeClient) StartServer() error {

	// logSignersChanged := sha3of("LogSignersChanged(uint,address[])")

	ctx := context.Background()
	ch := make(chan types.Log)

	query := ethereum.FilterQuery {
		FromBlock: big.NewInt(0),
		ToBlock: big.NewInt(10000000),
		Addresses: []common.Address{b.contractAddress},
		Topics: [][]common.Hash{{}},
	}
	_, err := b.client.SubscribeFilterLogs(ctx,query,ch)
	if err != nil {
		return err
	}
	log.Println("Server started...")

	go func() {
		log.Println("Started listening logs")
		for true {
	 	   log := <-ch
	       fmt.Printf("Matching log encountered %v\n",log)
	    }
	}()

	return nil
}

func createKeyStoreIfNotExists(keystoreFolder,passwd string) (bool) {
	ks := keystore.NewKeyStore(keystoreFolder, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) == 0 {
		log.Println("Keystore not found. Creating new one.")
		_, err := ks.NewAccount(passwd)
		if err != nil {
			panic(err)
		}
		log.Println("Caller address", ks.Accounts()[0].Address.Hex())
		return true
	}
	return false
}


func main() {

	passwd := "111111"

	if createKeyStoreIfNotExists("./keystore",passwd) {
		return
	}

	c,err := NewBridgeClient(
		"ws://127.0.0.1:8546",
		"./parent.abi",
		common.HexToAddress("0x8626F17170Db46FF28C5fBE37182AFC175cE4642"),
		"keystore",
		passwd,
	)
	assert(err)

	err = c.StartServer()
	assert(err)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"success": true} )
	})
	r.Run()
}
