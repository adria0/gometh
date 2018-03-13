package gometh

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	eth "gometh/eth"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func assert(err error) {
	if err != nil {
		panic("Failed: " + err.Error())
	}
}

var (
	parentClient   *eth.Web3Client
	childClient    *eth.Web3Client
	parentContract *eth.Contract
	childContract  *eth.Contract
	wethContract   *eth.Contract
)

func callLock(value *big.Int) error {
	_, _, err := parentContract.SendTransactionSync(value, "lock")
	return err
}

func callBurn(value *big.Int) error {
	_, _, err := childContract.SendTransactionSync(big.NewInt(0), "burn", value)
	return err
}

func sign(client *eth.Web3Client, data ...[]byte) ([3][32]byte, error) {
	web3SignaturePrefix := []byte("\x19Ethereum Signed Message:\n32")

	hash := crypto.Keccak256(data...)
	prefixedHash := crypto.Keccak256(web3SignaturePrefix, hash)

	var ret [3][32]byte

	// The produced signature is in the [R || S || V] format where V is 0 or 1.
	sig, err := client.Ks.SignHash(client.Account, prefixedHash)
	if err != nil {
		return ret, err
	}

	// We need to convert it to the format []uint256 = {v,r,s} format
	ret[0][31] = sig[64] + 27
	copy(ret[1][:], sig[0:32])
	copy(ret[2][:], sig[32:64])
	return ret, nil
}

func handleLockEvent(eventlog *types.Log) {

	type LogLockEvent struct {
		Epoch *big.Int
		From  common.Address
		Value *big.Int
	}

	var event LogLockEvent
	err := parentContract.Abi.Unpack(&event, "LogLock", eventlog.Data)
	assert(err)

	log.Printf("LockEvent %v %v wei", event.From.Hex(), event.Value)

	mintmsg, err := childContract.Abi.Pack("_mintmultisigned", event.From, event.Value)
	assert(err)

	var txhash [32]byte
	copy(txhash[:], eventlog.TxHash.Bytes())

	log.Printf("partialExecuteOff _mint")

	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOn", event.Epoch, txhash, mintmsg,
	)

	assert(err)

}

func handleLogEvent(eventlog *types.Log) {

	var event string
	err := parentContract.Abi.Unpack(&event, "Log", eventlog.Data)
	assert(err)

	log.Printf("contractlog %#v\n", event)
}

func handleBurnEvent(eventlog *types.Log) {

	type BurnEvent struct {
		Epoch *big.Int
		From  common.Address
		Value *big.Int
	}

	var event BurnEvent
	err := childContract.Abi.Unpack(&event, "LogBurn", eventlog.Data)
	assert(err)

	log.Printf("LogBurn")

	burnmsg, err := childContract.Abi.Pack("_burnmultisigned", event.From, event.Value)
	assert(err)

	var txhash [32]byte
	copy(txhash[:], eventlog.TxHash.Bytes())

	log.Printf("partialExecuteOff _burnmultisigned")

	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOn", event.Epoch, txhash, burnmsg,
	)

	assert(err)

}

func handleBurnMultisignedEvent(eventlog *types.Log) {

	log.Printf("LogBurnMultisigned")

}

func handleStateChange(eventlog *types.Log) {

	type StateChangeEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	epoch := big.NewInt(0)
	txid := common.BytesToHash(eventlog.TxHash.Bytes())

	var event StateChangeEvent
	err := wethContract.Abi.Unpack(&event, "StateChange", eventlog.Data)
	assert(err)

	msg, err := childContract.Abi.Pack("_statechangemultisigned", event.BlockNo, event.RootState)
	assert(err)
	sig, err := sign(childClient, abi.U256(epoch), txid[:], msg)

	assert(err)

	log.Printf("partialExecuteOff _statechangemultisigned")
	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOff", epoch, txid, msg, sig,
	)

	assert(err)
}

func handleMintMultisigned(eventlog *types.Log) {

	type MintMultisignedEvent struct {
		To    common.Address
		Value *big.Int
	}

	var event MintMultisignedEvent
	err := childContract.Abi.Unpack(&event, "LogMintMultisigned", eventlog.Data)
	assert(err)

	log.Printf("MintMultisigned %v %v wei\n", event.To.Hex(), event.Value)
}

func handleTransferEvent(eventlog *types.Log) {

	type TransferEvent struct {
		Value *big.Int
	}

	var event TransferEvent
	err := wethContract.Abi.Unpack(&event, "Transfer", eventlog.Data)
	assert(err)

	from := common.BytesToAddress(eventlog.Topics[1][:])
	to := common.BytesToAddress(eventlog.Topics[2][:])

	log.Printf("WTransfer %v %v->%v\n", event.Value, from.Hex(), to.Hex())
}

func handleStateChangeMultisigned(eventlog *types.Log) {

	type StateChangeMultisignedEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	log.Printf("StateChangeMultisigned")

}

func dotest() {
	callLock(big.NewInt(1000))
	callBurn(big.NewInt(10))
}

func startServer() {

	// -- open keystore

	var err error
	var account accounts.Account

	ks := keystore.NewKeyStore(C.KeystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) != 1 {
		panic(fmt.Sprintf("Not exact one account in keystore, was %v", len(ks.Accounts())))
	}
	account = ks.Accounts()[0]
	assert(ks.Unlock(account, C.KeystorePasswd))

	// -- create clients

	parentClient, err = eth.NewWeb3Client(
		C.ParentWSUrl,
		ks,
		account,
	)
	assert(err)

	childClient, err = eth.NewWeb3Client(
		C.ChildrenWSUrl,
		ks,
		account,
	)
	assert(err)

	parentClient.ClientMutex = &sync.Mutex{}
	childClient.ClientMutex = parentClient.ClientMutex

	parentAccountInfo, err := parentClient.AccountInfo()
	assert(err)
	log.Println("ACCOUNT INFO PARENT CHAIN", parentAccountInfo)

	childAccountInfo, err := childClient.AccountInfo()
	assert(err)
	log.Println("ACCOUNT INFO CHiLD CHAIN", childAccountInfo)

	// -- load contracts
	parentContract, err = eth.NewContract(parentClient, C.ContractsPath+"/GometParent.json")
	assert(err)

	childContract, err = eth.NewContract(childClient, C.ContractsPath+"/GometChild.json")
	assert(err)

	wethContract, err = eth.NewContract(childClient, C.ContractsPath+"/WETH.json")
	assert(err)

	// -- deploy contracts
	initialSigners := []common.Address{parentClient.Account.Address}
	_, _, err = parentContract.Deploy(initialSigners)
	assert(err)
	_, _, err = childContract.Deploy(initialSigners)
	assert(err)
	_, _, err = wethContract.Deploy(childContract.Address)
	assert(err)

	// -- get weth address
	_, _, err = childContract.SendTransactionSync(big.NewInt(0), "init", wethContract.Address)
	assert(err)

	// -- register event handlers & start processing

	assert(parentClient.RegisterEventHandler(parentContract, "LogLock", handleLockEvent))
	assert(parentClient.RegisterEventHandler(parentContract, "Log", handleLogEvent))

	assert(childClient.RegisterEventHandler(childContract, "Log", handleLogEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogBurn", handleBurnEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogBurnMultisigned", handleBurnMultisignedEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogStateChangeMultisigned", handleStateChangeMultisigned))
	assert(childClient.RegisterEventHandler(childContract, "LogMintMultisigned", handleMintMultisigned))

	assert(childClient.RegisterEventHandler(wethContract, "StateChange", handleStateChange))
	assert(childClient.RegisterEventHandler(wethContract, "Log", handleLogEvent))

	childClient.HandleEvents()
	parentClient.HandleEvents()

	<-time.After(time.Second * 3600)

}
