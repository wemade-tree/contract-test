package test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/wemade-tree/contract-test/backend"
)

type typeKeyMap map[common.Address]*ecdsa.PrivateKey

//Converts the given data into a byte slice and returns it.
func toBytes(t *testing.T, data interface{}) []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

//Converts the given number string based decimal number into big.Int type and returns it.
func toBig(t *testing.T, value10 string) *big.Int {
	ret, b := new(big.Int).SetString(value10, 10)
	if b == false {
		t.Fatal("failed > set string to *big.Int")
	}
	return ret
}

//checkVariable compares value stored in the blockchain with a given expected value
func checkVariable(t *testing.T, contract *backend.Contract, method string, expected interface{}) {
	if ret, err := contract.LowCall(method); err != nil {
		t.Fatal(err)
	} else if bytes.Equal(toBytes(t, ret[0]), toBytes(t, expected)) == false {
		t.Fatalf("failed > dismatch %s : expected %v , got %v", method, expected, ret[0])
	} else {
		switch expected.(type) {
		case common.Address:
			t.Log(method, ret[0].(common.Address).Hex())
		default:
			t.Log(method, ret[0])
		}
	}
}

//executeChangeMethod executes the method with the "change_" prefix in the contract,
//and then compares whether the given arg argument is applied well.
//The argument of "change_" prefixed method is assumed to be one.
func executeChangeMethod(t *testing.T, contract *backend.Contract, methodExceptCall string, arg interface{}) {
	changeMethod := "change_" + methodExceptCall

	if r, err := contract.Execute(nil, changeMethod, arg); err != nil {
		t.Fatal(err)
	} else if r.Status != 1 {
		t.Fatalf("failed > execute %s. receipt.status : %d", changeMethod, r.Status)
	} else if changed, err := contract.LowCall(methodExceptCall); err != nil {
		t.Fatal(err)
	} else if bytes.Equal(toBytes(t, changed[0]), toBytes(t, arg)) == false {
		switch arg.(type) {
		case common.Address:
			t.Fatalf("failed > %s : expected %v , got %v", methodExceptCall, arg.(common.Address).Hex(), changed[0].(common.Address).Hex())
		default:
			t.Fatalf("failed > %s : expected %v , got %v", methodExceptCall, arg, changed[0])
		}
	}
}

//causes contract execution to fail.
func expecedFail(t *testing.T, contract *backend.Contract, key *ecdsa.PrivateKey, method string, arg ...interface{}) {
	if r, err := contract.Execute(key, method, arg...); err != nil {
		t.Fatal(err)
	} else if r.Status == 0 {
		t.Logf("ok > denied to execute %s. receipt.status : %d", method, r.Status)
	} else {
		t.Fatalf("failed > accepted to execute %s. receipt.status : %d", method, r.Status)
	}
}

//checks if the contract execution is successful..
func expecedSuccess(t *testing.T, contract *backend.Contract, key *ecdsa.PrivateKey, method string, arg ...interface{}) {
	if r, err := contract.Execute(key, method, arg...); err != nil {
		t.Fatal(err)
	} else if r.Status == 1 {
		t.Logf("ok > accepted to execute %s. receipt.status : %d", method, r.Status)
	} else {
		t.Fatalf("failed > denied execute %s. receipt.status : %d", method, r.Status)
	}
}
