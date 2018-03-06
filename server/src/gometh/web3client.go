package gometh

import (
	"context"
	"encoding/hex"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
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
		return "", nil
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

	nonce, err := b.Client.NonceAt(ctx, b.Account.Address, nil)
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
		return nil, nil, err
	}

	if err = b.Client.SendTransaction(ctx, tx); err != nil {
		return nil, nil, err
	}

	log.Println("SEND Tx ", tx.Hash().Hex(), "...")

	start := time.Now()
	for receipt == nil && time.Now().Sub(start) < b.ReceiptTimeout {
		receipt, err = b.Client.TransactionReceipt(ctx, tx.Hash())
		if receipt == nil {
			time.Sleep(1000 * time.Millisecond)
		}
	}

	if receipt != nil && receipt.Status == types.ReceiptStatusFailed {
		log.Println("FAIL Tx ", tx.Hash().Hex())
		return tx, receipt, fmt.Errorf("ReceiptStatusFailed")
	}

	if receipt == nil {
		log.Println("LOST Tx ", tx.Hash().Hex())
		return tx, receipt, fmt.Errorf("ReceiptStatusFailed")
	}

	log.Println("SUCC Tx ", tx.Hash().Hex(), " gas ", receipt.GasUsed)

	return tx, receipt, err
}

func (b *Web3Client) RegisterEventHandler(address common.Address, eventSignature string, handler func(*types.Log)) {

	sha := sha3.NewKeccak256()
	sha.Write([]byte(eventSignature))
	hash := sha.Sum(nil)
	topic := "0x" + hex.EncodeToString(hash)

	eventHandler := EventHandler{
		Address:        address,
		EventSignature: eventSignature,
		Topic:          topic,
		Handler:        handler,
	}

	b.EventHandlers = append(b.EventHandlers, eventHandler)
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
			// dumpLogEvent(&logevent)
			for _, v := range b.EventHandlers {
				if logevent.Address == v.Address && logevent.Topics[0].Hex() == v.Topic {
					log.Print("EVENT ", v.EventSignature)
					v.Handler(&logevent)
					break
				}
			}
		}
	}()

	return nil
}
