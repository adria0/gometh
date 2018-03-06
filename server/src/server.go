package main

/*

geth --dev console --ws --networkid 1337

*/

import (
	"log"
	"math/big"
	"time"

	"gometh"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func assert(err error) {
	if err != nil {
		panic("Failed: " + err.Error())
	}
}

var (
	parentClient   *gometh.Web3Client
	parentContract *gometh.Contract
	childClient    *gometh.Web3Client
	childContract  *gometh.Contract
)

func callLock(value *big.Int) error {
	_, _, err := parentContract.SendTransactionSync(parentClient, value, "lock")
	return err
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

	mintmsg, err := childContract.Abi.Pack("_mint", event.From, event.Value)
	assert(err)

	var txhash [32]byte
	copy(txhash[:], eventlog.TxHash.Bytes())

	_, _, err = childContract.SendTransactionSync(
		childClient, big.NewInt(0),
		"partialExecuteOn", event.Epoch, txhash, mintmsg,
	)
	assert(err)
}

func handleLogEvent(eventlog *types.Log) {
	var event string
	err := parentContract.Abi.Unpack(&event, "Log", eventlog.Data)
	assert(err)

	log.Printf("Unpacked LogEvent %#v\n", event)
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

	// -- create clients

	parentClient, err = gometh.NewWeb3Client(
		"ws://127.0.0.1:8546",
		ks,
		account,
	)
	assert(err)

	childClient, err = gometh.NewWeb3Client(
		"ws://127.0.0.1:8546",
		ks,
		account,
	)
	assert(err)

	parentAccountInfo, err := parentClient.AccountInfo()
	log.Println("ACCOUNT INFO PARENT CHAIN", parentAccountInfo)

	childAccountInfo, err := childClient.AccountInfo()
	log.Println("ACCOUNT INFO CHiLD CHAIN", childAccountInfo)

	// -- contracts

	parentContract, err = gometh.NewContract("../../build/contracts/GometParent.json")
	assert(err)

	childContract, err = gometh.NewContract("../../build/contracts/GometChild.json")
	assert(err)

	// -- deploy contracts
	initialSigners := []common.Address{parentClient.Account.Address}

	log.Println("--- deploying parent contract ---")
	_, _, err = parentContract.Deploy(parentClient, initialSigners)
	assert(err)

	log.Println("--- deploying child contract  ---")
	_, _, err = childContract.Deploy(childClient, initialSigners)
	assert(err)

	// -- start processing

	parentClient.RegisterEventHandler(
		*parentContract.Address,
		"LogLock(uint256,address,uint256)",
		handleLockEvent,
	)
	parentClient.RegisterEventHandler(
		*parentContract.Address,
		"Log(string)",
		handleLogEvent,
	)
	childClient.RegisterEventHandler(
		*childContract.Address,
		"Log(string)",
		handleLogEvent,
	)

	childClient.HandleEvents()
	parentClient.HandleEvents()

	<-time.After(time.Second)

	callLock(big.NewInt(1))

	<-time.After(time.Second * 3600)

}
