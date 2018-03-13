package gometh

import (
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"fmt"
)

type EventHandler struct {
	Address        common.Address
	EventSignature string
	Topic          string
	Handler        func(*types.Log)
}

type Web3Client struct {
	ID             string
	ClientMutex    *sync.Mutex
	Client         *ethclient.Client
	Account        accounts.Account
	Ks             *keystore.KeyStore
	ReceiptTimeout time.Duration
	EventHandlers  []EventHandler
}

func NewWeb3Client(rpcUrl string, ks *keystore.KeyStore, account accounts.Account) (*Web3Client, error) {

	var err error

	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}

	return &Web3Client{
		Client:         client,
		Ks:             ks,
		Account:        account,
		ReceiptTimeout: 120 * time.Second,
		EventHandlers:  []EventHandler{},
	}, nil
}

func (b *Web3Client) AccountInfo() (string, error) {

	address := b.Account.Address.Hex()
	ctx := context.TODO()
	balance, err := b.Client.BalanceAt(ctx, b.Account.Address, nil)
	if err != nil {

		return "", err
	}
	return address + "=" + balance.String() + " wei", nil
}

func (b *Web3Client) SendTransactionSync(to *common.Address, value *big.Int, calldata []byte) (*types.Transaction, *types.Receipt, error) {

	var err error
	var tx *types.Transaction
	var receipt *types.Receipt

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	network, err := b.Client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err
	}

	gasPrice, err := b.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, err
	}

	gasLimit, err := b.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  b.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	})
	if err != nil {
		return nil, nil, err
	}

	b.ClientMutex.Lock()

	nonce, err := b.Client.NonceAt(ctx, b.Account.Address, nil)
	if err != nil {
		b.ClientMutex.Unlock()
		return nil, nil, err
	}

	if to == nil {
		tx = types.NewContractCreation(
			nonce,    // nonce int64
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	} else {
		tx = types.NewTransaction(
			nonce,    // nonce int64
			*to,      // to common.Address
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	}

	if tx, err = b.Ks.SignTx(b.Account, tx, network); err != nil {
		b.ClientMutex.Unlock()
		return nil, nil, err
	}

	if err = b.Client.SendTransaction(ctx, tx); err != nil {
		b.ClientMutex.Unlock()
		return nil, nil, err
	}
	b.ClientMutex.Unlock()

	start := time.Now()
	for receipt == nil && time.Now().Sub(start) < b.ReceiptTimeout {
		receipt, err = b.Client.TransactionReceipt(ctx, tx.Hash())
		if receipt == nil {
			time.Sleep(1000 * time.Millisecond)
		}
	}

	if receipt != nil && receipt.Status == types.ReceiptStatusFailed {
		return tx, receipt, fmt.Errorf("ReceiptStatusFailed")
	}

	if receipt == nil {
		return tx, receipt, fmt.Errorf("ReceiptStatusFailed")
	}

	return tx, receipt, err
}

func (b *Web3Client) Call(to *common.Address, value *big.Int, calldata []byte) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	msg := ethereum.CallMsg{
		From:  b.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	}

	return b.Client.CallContract(ctx, msg, nil)
}

func (b *Web3Client) RegisterEventHandler(contract *Contract, event string, handler func(*types.Log)) error {

	abievent, ok := contract.Abi.Events[event]
	if !ok {
		return fmt.Errorf("Event %v not found", event)
	}
	topicID := abievent.Id()

	eventHandler := EventHandler{
		Address:        *contract.Address,
		EventSignature: abievent.String(),
		Topic:          "0x" + hex.EncodeToString(topicID[:]),
		Handler:        handler,
	}

	b.EventHandlers = append(b.EventHandlers, eventHandler)
	return nil
}

func dumpLogEvent(eventlog *types.Log) {
	fmt.Println("Log from address", eventlog.Address.Hex())
	for c, t := range eventlog.Topics {
		fmt.Printf("  Topic[%v]: %v", c, t.Hex())
	}
	fmt.Println("  Data:", hex.EncodeToString(eventlog.Data))
}

func (b *Web3Client) HandleEvents() error {

	ctx := context.Background()
	ch := make(chan types.Log)

	addrs := []common.Address{}

	for _, v := range b.EventHandlers {
		found := false
		for _, addr := range addrs {
			if addr == v.Address {
				found = true
				break
			}
		}
		if !found {
			addrs = append(addrs, v.Address)
		}
	}

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   big.NewInt(10000000),
		Addresses: addrs,
		Topics:    [][]common.Hash{{}},
	}
	_, err := b.Client.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		return err
	}
	go func() {
		for true {
			logevent := <-ch
			if logevent.Removed {
				continue
			}
			for _, v := range b.EventHandlers {
				if logevent.Address == v.Address && logevent.Topics[0].Hex() == v.Topic {
					//					log.Println("[Event] ", v.EventSignature)
					if v.Handler != nil {
						go v.Handler(&logevent)
					}
					break
				}
			}
			//dumpLogEvent(&logevent)
		}
	}()

	return nil
}
