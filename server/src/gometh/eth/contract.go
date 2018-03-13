package gometh

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Contract is a smartcontract with optional address
type Contract struct {
	Abi      abi.ABI
	Client   *Web3Client
	ByteCode []byte
	Address  *common.Address
}

// NewContract initiates a contract ABI & bytecode from json file associated to a web3 client
func NewContract(client *Web3Client, jsonFile string) (*Contract, error) {

	var contract Contract

	content, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, err
	}

	abivalue := fields["abi"]
	bytecodehex := fields["bytecode"].(string)
	if contract.ByteCode, err = hex.DecodeString(bytecodehex[2:]); err != nil {
		return nil, err
	}

	abijson, err := json.Marshal(&abivalue)
	if err != nil {
		return nil, err
	}

	contract.Abi, err = abi.JSON(bytes.NewReader(abijson))
	if err != nil {
		return nil, err
	}

	contract.Client = client

	return &contract, nil
}

// SetAddress sets the contract's address
func (b *Contract) SetAddress(address common.Address) error {

	b.Address = &address
	return nil
}

// SendTransactionSync executes a contract method and wait it finalizes
func (b *Contract) SendTransactionSync(value *big.Int, funcname string, params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	msg, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		return nil, nil, err
	}
	return b.Client.SendTransactionSync(b.Address, value, msg)
}

// Deploy the contract
func (b *Contract) Deploy(params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	init, err := b.Abi.Pack("", params...)
	if err != nil {
		return nil, nil, err
	}

	code := append([]byte(nil), b.ByteCode...)
	code = append(code, init...)

	tx, receipt, err := b.Client.SendTransactionSync(nil, big.NewInt(0), code)

	if err == nil {
		b.Address = &receipt.ContractAddress
	}

	return tx, receipt, err
}

// Call an constant method
func (b *Contract) Call(value *big.Int, funcname string, params ...interface{}) ([]byte, error) {

	msgdata, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		return nil, err
	}
	return b.Client.Call(b.Address, value, msgdata)
}
