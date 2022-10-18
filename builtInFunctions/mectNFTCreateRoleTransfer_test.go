package builtInFunctions

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestMectNFTCreateRoleTransfer_Constructor(t *testing.T) {
	t.Parallel()

	e, err := NewMECTNFTCreateRoleTransfer(nil, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilMarshalizer)

	e, err = NewMECTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, nil, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	e, err = NewMECTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, nil)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilShardCoordinator)

	e, err = NewMECTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, err)
	assert.NotNil(t, e)
	assert.False(t, e.IsInterfaceNil())

	e.SetNewGasConfig(&vmcommon.GasCost{})
}

func TestMECTNFTCreateRoleTransfer_ProcessWithErrors(t *testing.T) {
	t.Parallel()

	e, err := NewMECTNFTCreateRoleTransfer(&mock.MarshalizerMock{}, &mock.AccountsStub{}, mock.NewMultiShardsCoordinatorMock(2))
	assert.Nil(t, err)
	assert.NotNil(t, e)

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{})
	assert.Equal(t, err, ErrNilValue)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(10)}})
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}})
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.Arguments = [][]byte{{1}, {2}}
	vmOutput, err = e.ProcessBuiltinFunction(&mock.UserAccountStub{}, nil, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrNilUserAccount)
	assert.Nil(t, vmOutput)

	vmInput.CallerAddr = core.MECTSCAddress
	vmInput.Arguments = [][]byte{{1}, {2}, {3}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)

	vmInput.Arguments = [][]byte{{1}, {2}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)
}

func createMECTNFTCreateRoleTransferComponent(t *testing.T) *mectNFTCreateRoleTransfer {
	marshaller := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(2)
	mapAccounts := make(map[string]vmcommon.UserAccountHandler)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
	}

	e, err := NewMECTNFTCreateRoleTransfer(marshaller, accounts, shardCoordinator)
	assert.Nil(t, err)
	assert.NotNil(t, e)
	return e
}

func TestMECTNFTCreateRoleTransfer_ProcessAtCurrentShard(t *testing.T) {
	t.Parallel()

	e := createMECTNFTCreateRoleTransferComponent(t)

	tokenID := []byte("NFT")
	currentOwner := bytes.Repeat([]byte{1}, 32)
	destinationAddr := bytes.Repeat([]byte{2}, 32)
	vmInput := &vmcommon.ContractCallInput{}
	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = core.MECTSCAddress
	vmInput.Arguments = [][]byte{tokenID, destinationAddr}

	destAcc, _ := e.accounts.LoadAccount(currentOwner)
	userAcc := destAcc.(vmcommon.UserAccountHandler)

	mectTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err := saveRolesToAccount(userAcc, mectTokenRoleKey, &mect.MECTRoles{Roles: [][]byte{[]byte(core.MECTRoleNFTCreate), []byte(core.MECTRoleNFTAddQuantity)}}, e.marshaller)
	assert.Nil(t, err)
	_ = saveLatestNonce(userAcc, tokenID, 100)
	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	destAcc, _ = e.accounts.LoadAccount(currentOwner)
	userAcc = destAcc.(vmcommon.UserAccountHandler)

	vmOutput, err := e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 1)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, currentOwner, tokenID, 0)
	checkNFTCreateRoleExists(t, e, currentOwner, tokenID, -1)

	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)
}

func TestMECTNFTCreateRoleTransfer_ProcessCrossShard(t *testing.T) {
	t.Parallel()

	e := createMECTNFTCreateRoleTransferComponent(t)

	tokenID := []byte("NFT")
	currentOwner := bytes.Repeat([]byte{1}, 32)
	destinationAddr := bytes.Repeat([]byte{2}, 32)
	vmInput := &vmcommon.ContractCallInput{}
	vmInput.CallValue = big.NewInt(0)
	vmInput.CallerAddr = currentOwner
	nonce := uint64(100)
	vmInput.Arguments = [][]byte{tokenID, big.NewInt(0).SetUint64(nonce).Bytes()}

	destAcc, _ := e.accounts.LoadAccount(destinationAddr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	vmOutput, err := e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 0)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)

	destAcc, _ = e.accounts.LoadAccount(destinationAddr)
	userAcc = destAcc.(vmcommon.UserAccountHandler)
	vmOutput, err = e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, len(vmOutput.OutputAccounts), 0)

	_ = e.accounts.SaveAccount(userAcc)
	_, _ = e.accounts.Commit()
	checkLatestNonce(t, e, destinationAddr, tokenID, 100)
	checkNFTCreateRoleExists(t, e, destinationAddr, tokenID, 0)

	vmInput.Arguments = append(vmInput.Arguments, []byte{100})
	vmOutput, err = e.ProcessBuiltinFunction(nil, userAcc, vmInput)
	assert.Equal(t, err, ErrInvalidArguments)
	assert.Nil(t, vmOutput)
}

func checkLatestNonce(t *testing.T, e *mectNFTCreateRoleTransfer, addr []byte, tokenID []byte, expectedNonce uint64) {
	destAcc, _ := e.accounts.LoadAccount(addr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	nonce, _ := getLatestNonce(userAcc, tokenID)
	assert.Equal(t, expectedNonce, nonce)
}

func checkNFTCreateRoleExists(t *testing.T, e *mectNFTCreateRoleTransfer, addr []byte, tokenID []byte, expectedIndex int) {
	destAcc, _ := e.accounts.LoadAccount(addr)
	userAcc := destAcc.(vmcommon.UserAccountHandler)
	mectTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, _, _ := getMECTRolesForAcnt(e.marshaller, userAcc, mectTokenRoleKey)
	assert.Equal(t, 1, len(roles.Roles))
	index, _ := doesRoleExist(roles, []byte(core.MECTRoleNFTCreate))
	assert.Equal(t, expectedIndex, index)
}
