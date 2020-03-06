package backend

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Contract struct {
	File              string
	Name              string
	Backend           *backends.SimulatedBackend
	OwnerKey          *ecdsa.PrivateKey
	Owner             common.Address
	Info              *compiler.ContractInfo
	ConstructorInputs []interface{}
	Abi               *abi.ABI
	Code              []byte
	Address           common.Address
	BlockDeployed     *big.Int
}

func NewContract(file, name string) (*Contract, error) {

	ownerKey, _ := crypto.GenerateKey()

	r := &Contract{
		File: file,
		Name: name,
		Backend: backends.NewSimulatedBackend(
			nil,
			10000000,
		),
		OwnerKey: ownerKey,
		Owner:    crypto.PubkeyToAddress(ownerKey.PublicKey),
	}
	if err := r.compile(); err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Contract) compile() error {
	contracts, err := compiler.CompileSolidity("", p.File)
	if err != nil {
		return err
	}

	contract, ok := contracts[fmt.Sprintf("%s:%s", p.File, p.Name)]
	if ok == false {
		fmt.Errorf("%s contract is not here", p.Name)
	}

	abiBytes, err := json.Marshal(contract.Info.AbiDefinition)
	if err != nil {
		return err
	}
	abi, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return err
	}
	p.Info = &contract.Info
	p.Abi = &abi
	p.Code = common.FromHex(contract.Code)
	return nil
}

func (p *Contract) Deploy(args ...interface{}) error {
	input, err := p.Abi.Pack("", args...)
	if err != nil {
		return err
	}

	p.ConstructorInputs = args

	tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(0), append(p.Code, input...))
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, p.OwnerKey)

	if err := p.Backend.SendTransaction(context.Background(), tx); err != nil {
		return err
	}
	p.Backend.Commit()

	//get contract address through receipt
	receipt, err := p.Backend.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return fmt.Errorf("status of deploy tx receipt: %v", receipt.Status)
	}
	p.Address = receipt.ContractAddress
	p.BlockDeployed = receipt.BlockNumber
	return nil
}

func (p *Contract) Call(result interface{}, method string, args ...interface{}) error {
	if input, err := p.Abi.Pack(method, args...); err != nil {
		return err
	} else {
		msg := ethereum.CallMsg{From: common.Address{}, To: &p.Address, Data: input}

		out := result
		if output, err := p.Backend.CallContract(context.TODO(), msg, nil); err != nil {
			return err
		} else if err := p.Abi.Unpack(out, method, output); err != nil {
			return err
		}
	}
	return nil
}

func (p *Contract) LowCall(method string, args ...interface{}) ([]interface{}, error) {
	if input, err := p.Abi.Pack(method, args...); err != nil {
		return nil, err
	} else {
		msg := ethereum.CallMsg{From: common.Address{}, To: &p.Address, Data: input}
		if out, err := p.Backend.CallContract(context.TODO(), msg, nil); err != nil {
			return []interface{}{}, err
		} else {
			if ret, err := p.Abi.Methods[method].Outputs.UnpackValues(out); err != nil {
				return []interface{}{}, err
			} else {
				return ret, nil
			}
		}
	}
}

func (p *Contract) Execute(key *ecdsa.PrivateKey, method string, args ...interface{}) (*types.Receipt, error) {
	if key == nil {
		key = p.OwnerKey
	}

	data, err := p.Abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	nonce, err := p.Backend.PendingNonceAt(context.Background(), crypto.PubkeyToAddress(key.PublicKey))
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, p.Address, new(big.Int), uint64(10000000), big.NewInt(0), data)
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, key)

	if err != nil {
		return nil, err
	}
	if err := p.Backend.SendTransaction(context.Background(), tx); err != nil {
		return nil, err
	}
	p.Backend.Commit()

	receipt, err := p.Backend.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return nil, err
	}

	return receipt, nil
}
