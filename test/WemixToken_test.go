package test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wemade-tree/contract-test/backend"
)

type (
	//Structure to store block partner information
	typePartner struct {
		Serial                 *big.Int
		Partner                common.Address
		Payer                  common.Address
		BlockStaking           *big.Int
		BlockWaitingWithdrawal *big.Int
		BalanceStaking         *big.Int
	}
	typePartnerSlice []*typePartner
)

//Print all block partner information in log.
func (p *typePartner) log(serial *big.Int, t *testing.T) {
	t.Logf("Partner:%s serial:%v", p.Partner.Hex(), serial)
	t.Logf(" -Payer:%v", p.Payer.Hex())
	t.Logf(" -BalanceStaking:%v", p.BalanceStaking)
	t.Logf(" -BlockStaking:%v", p.BlockStaking)
	t.Logf(" -BlockWaitingWithdrawal:%v", p.BlockWaitingWithdrawal)
}

//Retrieve and store all block partner information from the blockchain,
func (p *typePartnerSlice) loadAllStake(t *testing.T, contract *backend.Contract) {
	partnersNumber := (*big.Int)(nil)
	if err := contract.Call(&partnersNumber, "partnersNumber"); err != nil {
		t.Fatal(err)
	}

	for i := int64(0); i < partnersNumber.Int64(); i++ {
		s := typePartner{}

		if err := contract.Call(&s, "partnerByIndex", new(big.Int).SetInt64(i)); err != nil {
			t.Fatal(err)
		}
		*p = append(*p, &s)
	}
	t.Logf("ok > loadAllStake, partners number: %d", partnersNumber)
}

//After compiling and distributing the contract, return the Contract pointer object.
func depolyWemix(t *testing.T) *backend.Contract {
	contract, err := backend.NewContract("../contracts/WemixToken.sol", "WemixToken")
	if err != nil {
		t.Fatal(err)
	}

	ecoFundKey, _ := crypto.GenerateKey()
	wemixKey, _ := crypto.GenerateKey()

	//deploy contract
	args := []interface{}{
		crypto.PubkeyToAddress(ecoFundKey.PublicKey), //ecoFund address
		crypto.PubkeyToAddress(wemixKey.PublicKey),   //wemix address
	}
	if err := contract.Deploy(args...); err != nil {
		t.Fatal(err)
	}
	return contract
}

//Test to compile and deploy the contract
func TestWemixDeploy(t *testing.T) {
	contract := depolyWemix(t)

	t.Log("contract source file:", contract.File)
	t.Log("contract name:", contract.Name)
	t.Log("contract Language:", contract.Info.Language)
	t.Log("contract LanguageVersion", contract.Info.LanguageVersion)
	t.Log("contract CompilerVersion", contract.Info.CompilerVersion)
	t.Log("contract bytecode size:", len(contract.Code))

	t.Log("ok > contract address deployed", contract.Address.Hex())
}

//Test to verify the variables of the deployed contract.
//Fatal if the expected value and the actual contract value differ.
func TestWemixVariable(t *testing.T) {
	contract := depolyWemix(t)

	checkVariable(t, contract, "name", "WEMIX TOKEN")
	checkVariable(t, contract, "symbol", "WEMIX")
	checkVariable(t, contract, "decimals", uint8(18))
	checkVariable(t, contract, "totalSupply", toBig(t, "1000000000000000000000000000"))
	checkVariable(t, contract, "unitStaking", toBig(t, "5000000000000000000000000"))
	checkVariable(t, contract, "minBlockWaitingWithdrawal", new(big.Int).SetUint64(7776000))
	checkVariable(t, contract, "maxTimesMintingOnce", new(big.Int).SetUint64(50))
	checkVariable(t, contract, "ecoFund", contract.ConstructorInputs[0].(common.Address))
	checkVariable(t, contract, "wemix", contract.ConstructorInputs[1].(common.Address))
	checkVariable(t, contract, "nextPartnerToMint", new(big.Int))
	checkVariable(t, contract, "mintToPartner", new(big.Int).SetUint64(500000000000000000))
	checkVariable(t, contract, "mintToEcoFund", new(big.Int).SetUint64(250000000000000000))
	checkVariable(t, contract, "mintToWemix", new(big.Int).SetUint64(250000000000000000))
	checkVariable(t, contract, "blockToMint", contract.BlockDeployed)
}

//Test to execute onlyOwner modifier method.
func TestWemixExecute(t *testing.T) {
	contract := depolyWemix(t)

	executeChangeMethod(t, contract, "unitStaking", big.NewInt(1))
	executeChangeMethod(t, contract, "minBlockWaitingWithdrawal", big.NewInt(1))
	executeChangeMethod(t, contract, "maxTimesMintingOnce", big.NewInt(1))
	executeChangeMethod(t, contract, "ecoFund", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	executeChangeMethod(t, contract, "wemix", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	executeChangeMethod(t, contract, "mintToPartner", big.NewInt(1))
	executeChangeMethod(t, contract, "mintToEcoFund", big.NewInt(1))
	executeChangeMethod(t, contract, "mintToWemix", big.NewInt(1))
}

//test to run onlyOwner modifier method under non-owner account.
func TestWemixOwner(t *testing.T) {
	contract := depolyWemix(t)

	key, _ := crypto.GenerateKey()

	expecedFail(t, contract, key, "change_unitStaking", big.NewInt(1))
	expecedFail(t, contract, key, "change_minBlockWaitingWithdrawal", big.NewInt(1))
	expecedFail(t, contract, key, "change_maxTimesMintingOnce", big.NewInt(1))
	expecedFail(t, contract, key, "change_ecoFund", common.HexToAddress("0x0000000000000000000000000000000000000001"))
	expecedFail(t, contract, key, "change_wemix", common.HexToAddress("0x0000000000000000000000000000000000000002"))
	expecedFail(t, contract, key, "change_mintToPartner", big.NewInt(1))
	expecedFail(t, contract, key, "change_mintToWemix", big.NewInt(1))
	expecedFail(t, contract, key, "transferOwnership", func() common.Address {
		k, _ := crypto.GenerateKey()
		return crypto.PubkeyToAddress(k.PublicKey)
	}())

	newOwnerKey, _ := crypto.GenerateKey()
	expecedSuccess(t, contract, nil, "transferOwnership", crypto.PubkeyToAddress(newOwnerKey.PublicKey))
	expecedSuccess(t, contract, newOwnerKey, "transferOwnership", contract.Owner)
}

//Test to run addAllowedStaker method.
func TestWemixAllowedPartner(t *testing.T) {
	contract := depolyWemix(t)

	//make an error occur
	if r, err := contract.Execute(nil, "stake", new(big.Int)); err != nil {
		t.Fatal(err)
	} else if r.Status != 0 {
		t.Fatalf("failed > addAllowedStaker needed before stake but accepted. receipt.Status : %d", r.Status)
	}

	//make partner
	partnerKey, _ := crypto.GenerateKey()
	partner := crypto.PubkeyToAddress(partnerKey.PublicKey)

	//addAllowedPartner
	if r, err := contract.Execute(nil, "addAllowedPartner", partner); err != nil {
		t.Fatal(err)
	} else if r.Status != 1 {
		t.Fatalf("failed > addAllowedPartner : %s, receipt.Status : %d ", contract.Address.Hex(), r.Status)
	}

	topics := []common.Hash{}
	if r, err := contract.Execute(nil, "stakeDelegated", partner, new(big.Int)); err != nil {
		t.Fatal(err)
	} else if r.Status != 1 {
		t.Fatalf("failed > not accepted but execute addAllowedStake before")
	} else {
		for _, g := range r.Logs {
			if g.Topics[0] == contract.Abi.Events["Staked"].Id() {
				topics = append(topics, g.Topics...)
			}
		}
		if common.BytesToAddress(topics[1].Bytes()) != partner {
			t.Fatal("failed > mismatch partner after stake")
		}
		if common.BytesToAddress(topics[2].Bytes()) != contract.Owner {
			t.Fatal("failed > mismatch payer after stake")
		}
	}

	t.Log("ok > test addAllowedPartner")
}

//test staking
func TestWemixStake(t *testing.T) {
	contract := depolyWemix(t)

	testStake(t, contract, true)
}

func testStake(t *testing.T, contract *backend.Contract, showStakeInfo bool) typeKeyMap {
	unitStaking := func() *big.Int {
		ret := (*big.Int)(nil)
		if err := contract.Call(&ret, "unitStaking"); err != nil {
			t.Fatal(err)
		}
		return ret
	}()

	minBlockWaitingWithdrawal := (*big.Int)(nil)
	if err := contract.Call(&minBlockWaitingWithdrawal, "minBlockWaitingWithdrawal"); err != nil {
		t.Fatal(err)
	}

	countExecuteStake := int64(0)
	partnerKeyMap := typeKeyMap{}

	_stake := func(delegation bool, partner common.Address, payerKey *ecdsa.PrivateKey, waitBlock *big.Int) *typePartner {
		//addAllowedPartner
		if r, err := contract.Execute(nil, "addAllowedPartner", partner); err != nil {
			t.Fatal(err)
		} else if r.Status != 1 {
			t.Fatalf("failed > addAllowedPartner : %s, receipt.Status : %d", partner.Hex(), r.Status)
		}

		//stakeDelegated
		serial := new(big.Int)
		var r *types.Receipt
		var err error
		if delegation == true {
			r, err = contract.Execute(payerKey, "stakeDelegated", partner, waitBlock)
		} else {
			r, err = contract.Execute(payerKey, "stake", waitBlock)
		}

		if err != nil {
			t.Fatal(err)
		} else if r.Status != 1 {
			t.Fatalf("failed > stake : %s, receipt.Status : %d", partner.Hex(), r.Status)
		} else {
			for _, g := range r.Logs {
				if g.Topics[0] == contract.Abi.Events["Staked"].Id() {
					serial = g.Topics[3].Big()
				}
			}
			if serial == nil {
				t.Fatal("failed > find stake index")
			}
			countExecuteStake++
		}

		result := typePartner{}
		if err := contract.Call(&result, "partnerBySerial", serial); err != nil {
			t.Fatal(err)
		} else if showStakeInfo == true {
			result.log(serial, t)
		}
		return &result
	}

	makePartner := func() (common.Address, *ecdsa.PrivateKey) {
		partnerKey, _ := crypto.GenerateKey()
		partner := crypto.PubkeyToAddress(partnerKey.PublicKey)
		partnerKeyMap[partner] = partnerKey
		return partner, partnerKey
	}

	//stake
	for i := 0; i < 3; i++ {
		partner, partnerKey := makePartner()

		//wemix 지급
		amount := new(big.Int).Mul(unitStaking, new(big.Int).SetInt64(int64(i+1)))
		if r, err := contract.Execute(nil, "transfer", partner, amount); err != nil {
			t.Fatal(err)
		} else if r.Status != 1 {
			t.Fatalf("failed > error transfer wemix to %s", partner.Hex())
		}
		waitBlock := new(big.Int).Mul(minBlockWaitingWithdrawal, new(big.Int).SetInt64(int64(i+1)))

		for {
			result := _stake(false, partner, partnerKey, waitBlock)

			if result.Payer != result.Partner {
				t.Fatalf("failed > mismatch payer:%v and partner:%v", result.Payer.Hex(), result.Partner.Hex())
			}
			if partner != result.Payer {
				t.Fatalf("failed > mismatch partner:%v and payer:%v", partner.Hex(), result.Payer.Hex())
			}

			//받은 토큰을 모두 staking했으면 종료
			balance := (*big.Int)(nil)
			if err := contract.Call(&balance, "balanceOf", partner); err != nil {
				t.Fatal(err)
			} else if balance.Sign() == 0 {
				break
			}
		}
	}

	//delegated stake
	for i := 0; i < 5; i++ {
		partner, _ := makePartner()

		amount := new(big.Int).Mul(unitStaking, new(big.Int).SetInt64(int64(i+1)))

		waitBlock := new(big.Int).Mul(minBlockWaitingWithdrawal, new(big.Int).SetInt64(int64(i+1)))

		for {
			result := _stake(true, partner, contract.OwnerKey, waitBlock)

			if result.Payer == result.Partner {
				t.Fatalf("failed > equal payer:%v, partner:%v", result.Payer.Hex(), result.Partner.Hex())
			}
			if contract.Owner != result.Payer {
				t.Fatalf("failed > mismatch sender:%v, payer:%v", contract.Owner.Hex(), result.Payer.Hex())
			}

			amount = new(big.Int).Sub(amount, result.BalanceStaking)
			if amount.Sign() == 0 {
				break
			}
		}
	}

	//Compare the number of block partners registered with the number registered on the blockchain.
	partnersNumber := (*big.Int)(nil)
	if err := contract.Call(&partnersNumber, "partnersNumber"); err != nil {
		t.Fatal(err)
	}
	if partnersNumber.Cmp(new(big.Int).SetInt64(countExecuteStake)) != 0 {
		t.Fatalf("failed > mismatch partnersNumber, expected:%v, got:%v", countExecuteStake, partnersNumber)
	}

	return partnerKeyMap
}

//test to withdraw
func TestWemixWithdraw(t *testing.T) {
	contract := depolyWemix(t)

	//withdrawalWaitingMinBlockd을 짧게 바꿈.
	if r, err := contract.Execute(nil, "change_minBlockWaitingWithdrawal", new(big.Int).SetUint64(1000)); err != nil {
		t.Fatal(err)
	} else if r.Status != 1 {
		t.Fatalf("failed > execute change_minBlockWaitingWithdrawal, receipt.Status : %d", r.Status)
	}

	partnerKeyMap := testStake(t, contract, false)

	stakes := typePartnerSlice{}
	stakes.loadAllStake(t, contract)

	contractBalance := (*big.Int)(nil)
	if err := contract.Call(&contractBalance, "balanceOf", contract.Address); err != nil {
		t.Fatal(err)
	}

	totalStakeBalance := new(big.Int)
	for i := 0; i < len(stakes); i++ {
		totalStakeBalance.Add(totalStakeBalance, stakes[i].BalanceStaking)
	}

	if totalStakeBalance.Cmp(contractBalance) != 0 {
		t.Fatalf("failed > mismatch balance of token contract between total stake balance, contract:%v, total:%v", contractBalance, totalStakeBalance)
	} else {
		t.Logf("ok > contract's balance: %v, total stake balance:%v", contractBalance, totalStakeBalance)
	}

	for {
		for i, s := range stakes {
			key := (*ecdsa.PrivateKey)(nil)
			if s.Partner == s.Payer {
				key = partnerKeyMap[s.Payer]
			}

			if r, err := contract.Execute(key, "withdraw", s.Serial); err != nil {
				t.Fatal(err)
			} else {
				block := contract.Backend.Blockchain().CurrentBlock().Header().Number
				blockWithdrawable := new(big.Int).Add(s.BlockStaking, s.BlockWaitingWithdrawal)
				if r.Status == 1 {
					if block.Cmp(blockWithdrawable) < 0 {
						t.Fatalf("failed > withdrawal blockWithdrawable:%v, currentBlock:%v", blockWithdrawable, block)
					}
					t.Logf("ok > withdrawal : %v", s.Serial)
					stakes[i] = stakes[len(stakes)-1]
					stakes = stakes[:len(stakes)-1]
					break
				} else {
					if block.Cmp(blockWithdrawable) >= 0 {
						t.Fatal("failed > withdrawal", "index", i, "block", block, "blockWithdrawable", blockWithdrawable, "BlockStaking", s.BlockStaking, "BlockWaitingWithdrawal", s.BlockWaitingWithdrawal)
					}
				}
			}
		}

		if len(stakes) == 0 {
			break
		}

		//make block
		contract.Backend.Commit()
	}

	for staker, key := range partnerKeyMap {
		balance := (*big.Int)(nil)
		if err := contract.Call(&balance, "balanceOf", staker); err != nil {
			t.Fatal(err)
		}
		if balance.Sign() > 0 {
			if r, err := contract.Execute(key, "transfer", contract.Owner, balance); err != nil {
				t.Fatal(err)
			} else if r.Status != 1 {
				t.Fatal("failed > execute transfer to onwer")
			} else {
				t.Log("ok > return token to owner")
			}
		}
	}
}

func testMint(t *testing.T, contract *backend.Contract) {
	stakes := typePartnerSlice{}
	stakes.loadAllStake(t, contract)

	balancePartners := func() map[common.Address]*big.Int {
		m := make(map[common.Address]*big.Int)
		for _, s := range stakes {
			if _, ok := m[s.Partner]; ok == true {
				continue
			}
			b := (*big.Int)(nil)
			if err := contract.Call(&b, "balanceOf", s.Partner); err != nil {
				t.Fatal(err)
			}
			m[s.Partner] = b
		}
		return m
	}()

	wemix := common.Address{}
	if err := contract.Call(&wemix, "wemix"); err != nil {
		t.Fatal(err)
	}

	balanceWemix := func() *big.Int {
		b := (*big.Int)(nil)
		if err := contract.Call(&b, "balanceOf", wemix); err != nil {
			t.Fatal(err)
		}
		return b
	}()

	ecoFund := common.Address{}
	if err := contract.Call(&ecoFund, "ecoFund"); err != nil {
		t.Fatal(err)
	}

	balanceEcoFund := func() *big.Int {
		b := (*big.Int)(nil)
		if err := contract.Call(&b, "balanceOf", ecoFund); err != nil {
			t.Fatal(err)
		}
		return b
	}()

	mintToPartner := (*big.Int)(nil)
	if err := contract.Call(&mintToPartner, "mintToPartner"); err != nil {
		t.Fatal(err)
	}

	mintToWemix := (*big.Int)(nil)
	if err := contract.Call(&mintToWemix, "mintToWemix"); err != nil {
		t.Fatal(err)
	}

	mintToEcoFund := (*big.Int)(nil)
	if err := contract.Call(&mintToEcoFund, "mintToEcoFund"); err != nil {
		t.Fatal(err)
	}

	nextPartnerToMint := (*big.Int)(nil)
	if err := contract.Call(&nextPartnerToMint, "nextPartnerToMint"); err != nil {
		t.Fatal(err)
	}
	indexNext := int(nextPartnerToMint.Uint64())

	initialTotalSupply := (*big.Int)(nil)
	if err := contract.Call(&initialTotalSupply, "totalSupply"); err != nil {
		t.Fatal(err)
	}

	startBlock := (*big.Int)(nil)
	if err := contract.Call(&startBlock, "blockToMint"); err != nil {
		t.Fatal(err)
	}
	timesMinting := uint64(1000)

	for i := uint64(0); i < timesMinting; i++ {
		contract.Backend.Commit() //make block
		key, _ := crypto.GenerateKey()
		if r, err := contract.Execute(key, "mint"); err != nil {
			t.Fatal(err)
		} else if r.Status != 1 {
			t.Fatalf("failed > execute mint, receipt,Status : %d", r.Status)
		}
	}
	endBlock := (*big.Int)(nil)
	if err := contract.Call(&endBlock, "blockToMint"); err != nil {
		t.Fatal(err)
	}

	pendingBlock := (*big.Int)(nil)
	if err := contract.Call(&pendingBlock, "pendingBlock"); err != nil {
		t.Fatal(err)
	}
	if pendingBlock.Sign() != 0 {
		t.Fatal("failed > pending block is remained")
	}

	totalMinted := new(big.Int)

	//expected
	for i := startBlock.Uint64(); i < endBlock.Uint64(); i++ {
		if len(stakes) > 0 {
			if indexNext >= len(stakes) {
				indexNext = 0
			}
			s := stakes[indexNext]
			balancePartners[s.Partner].Add(balancePartners[s.Partner], mintToPartner)
			totalMinted.Add(totalMinted, mintToPartner)
			indexNext++
		}
		balanceWemix.Add(balanceWemix, mintToWemix)
		totalMinted.Add(totalMinted, mintToWemix)
		balanceEcoFund.Add(balanceEcoFund, mintToEcoFund)
		totalMinted.Add(totalMinted, mintToEcoFund)
	}

	checkBalance := func(tag string, addr common.Address, expected *big.Int) {
		got := (*big.Int)(nil)
		if err := contract.Call(&got, "balanceOf", addr); err != nil {
			t.Fatal(err)
		}
		if expected.Cmp(got) != 0 {
			t.Fatalf("failed > mismatch %s(%s) balance  expected:%v, got:%v", tag, addr.Hex(), expected, got)
		} else {
			t.Logf("ok > %s(%s) balance  expected:%v, got:%v", tag, addr.Hex(), expected, got)
		}
	}

	for a, expected := range balancePartners {
		checkBalance("partner", a, expected)
	}
	checkBalance("wemix", wemix, balanceWemix)
	checkBalance("ecoFund", ecoFund, balanceEcoFund)

	totalSupply := (*big.Int)(nil)
	if err := contract.Call(&totalSupply, "totalSupply"); err != nil {
		t.Fatal(err)
	} else {
		expected := new(big.Int).Add(initialTotalSupply, totalMinted)
		if totalSupply.Cmp(expected) != 0 {
			t.Fatalf("failed > mismatch totalSupply and expected totalSupply after mint, got :%d, expectd: %d", totalSupply, expected)
		} else {
			t.Logf("ok > match totalSupply and expected totalSupply after mint, got :%d, expectd: %d", totalSupply, expected)
		}
	}
}

//After registering block partners, do minting test and check the amount of minting.
func TestWemixMint(t *testing.T) {
	contract := depolyWemix(t)
	testStake(t, contract, false)

	testMint(t, contract)
}

//Test minting without block partner and check minting amount.
func TestWemixMintWithoutPartner(t *testing.T) {
	contract := depolyWemix(t)

	testMint(t, contract)
}
