package main

/*

geth --dev console --ws --networkid 1337

*/

import (
	"encoding/hex"
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
	childClient    *gometh.Web3Client
	parentContract *gometh.Contract
	childContract  *gometh.Contract
	wethContract   *gometh.Contract
)

func callLock(value *big.Int) error {
	_, _, err := parentContract.SendTransactionSync(value, "lock")
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

func handleCommitStateEvent(eventlog *types.Log) {

	type CommitStateEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	var event CommitStateEvent
	err := wethContract.Abi.Unpack(&event, "CommitState", eventlog.Data)
	assert(err)

	log.Printf("CommitStateEvent block=%v hash=%v\n", event.BlockNo, hex.EncodeToString(event.RootState[:]))
}

func handleTransferEvent(eventlog *types.Log) {

	type TransferEvent struct {
		_     common.Address
		_     common.Address
		Value *big.Int
	}

	var event TransferEvent
	err := wethContract.Abi.Unpack(&event, "Transfer", eventlog.Data)
	assert(err)

	from := common.BytesToAddress(eventlog.Topics[1][:])
	to := common.BytesToAddress(eventlog.Topics[2][:])

	log.Printf("WEthTransfer %v %v->%v\n", event.Value, from.Hex(), to.Hex())
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
	parentContract, err = gometh.NewContract(parentClient, "../../build/contracts/GometParent.json")
	assert(err)

	childContract, err = gometh.NewContract(childClient, "../../build/contracts/GometChild.json")
	assert(err)

	// -- deploy contracts
	initialSigners := []common.Address{parentClient.Account.Address}
	_, _, err = parentContract.Deploy(initialSigners)
	assert(err)
	_, _, err = childContract.Deploy(initialSigners)
	assert(err)

	// -- get weth address
	wethcallresult, err := childContract.Call(big.NewInt(0), "weth")
	assert(err)
	wethContract, err = gometh.NewContract(childClient, "../../build/contracts/WETH.json")
	assert(err)
	wethContract.SetAddress(common.BytesToAddress(wethcallresult[12:]))

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

	childClient.RegisterEventHandler(
		*wethContract.Address,
		"CommitState(uint256,bytes32)",
		handleCommitStateEvent,
	)

	childClient.RegisterEventHandler(
		*wethContract.Address,
		"Transfer(address,address,uint256)",
		handleTransferEvent,
	)

	childClient.RegisterEventHandler(
		*wethContract.Address,
		"Log(string)",
		handleLogEvent,
	)

	childClient.HandleEvents()
	parentClient.HandleEvents()

	<-time.After(time.Second)

	callLock(big.NewInt(1))

	<-time.After(time.Second * 3600)

}
