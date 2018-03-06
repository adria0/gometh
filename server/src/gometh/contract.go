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

type Contract struct {
	Abi      abi.ABI
	ByteCode []byte
	Address  *common.Address
}

func NewContract(jsonFile string) (*Contract, error) {

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

	return &contract, nil
}

func (b *Contract) SetAddress(address common.Address) error {

	b.Address = &address
	return nil
}

func (b *Contract) SendTransactionSync(client *Web3Client, value *big.Int, funcname string, params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	msg, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		return nil, nil, err
	}
	return client.SendTransactionSync(b.Address, value, msg)
}

func (b *Contract) Deploy(client *Web3Client, params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	init, err := b.Abi.Pack("", params...)
	if err != nil {
		return nil, nil, err
	}

	code := append([]byte(nil), b.ByteCode...)
	code = append(code, init...)

	tx, receipt, err := client.SendTransactionSync(nil, big.NewInt(0), code)

	if err == nil {
		b.Address = &receipt.ContractAddress
	}

	return tx, receipt, err
}
