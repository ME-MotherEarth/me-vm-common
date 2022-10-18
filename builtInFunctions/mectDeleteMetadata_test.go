package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func createMockArgsForNewMECTDelete() ArgsNewMECTDeleteMetadata {
	return ArgsNewMECTDeleteMetadata{
		FuncGasCost:     1,
		Marshalizer:     &mock.MarshalizerMock{},
		Accounts:        &mock.AccountsStub{},
		ActivationEpoch: 0,
		EpochNotifier:   &mock.EpochNotifierStub{},
		AllowedAddress:  bytes.Repeat([]byte{1}, 32),
		Delete:          true,
	}
}

func TestNewMECTDeleteMetadataFunc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDelete()
	args.Marshalizer = nil
	_, err := NewMECTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArgsForNewMECTDelete()
	args.Accounts = nil
	_, err = NewMECTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArgsForNewMECTDelete()
	args.EpochNotifier = nil
	_, err = NewMECTDeleteMetadataFunc(args)
	assert.Equal(t, err, ErrNilEpochHandler)

	args = createMockArgsForNewMECTDelete()
	e, err := NewMECTDeleteMetadataFunc(args)
	assert.Nil(t, err)
	assert.False(t, e.IsInterfaceNil())
	assert.True(t, e.IsActive())

	args = createMockArgsForNewMECTDelete()
	args.ActivationEpoch = 1
	e, _ = NewMECTDeleteMetadataFunc(args)
	assert.False(t, e.IsActive())

	e.SetNewGasConfig(&vmcommon.GasCost{})
}

func TestMectDeleteMetaData_ProcessBuiltinFunctionErrors(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDelete()
	e, _ := NewMECTDeleteMetadataFunc(args)

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrNilVmInput)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(10)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}})
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrAddressIsNotAllowed)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidRcvAddr)

	vmInput.RecipientAddr = e.allowedAddress
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	e.delete = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	e.delete = true
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments = [][]byte{{1}, {0}, {1}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	e.delete = false

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}

	e.accounts = accounts
	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNonce)

	vmInput.Arguments[0] = []byte("TOKEN-ABABAB")
	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidTokenID)

	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	mectMetadata := &mect.MetaData{Name: []byte("something"), Nonce: 1}
	marshalledData, _ := e.marshaller.Marshal(mectMetadata)
	vmInput.Arguments[2] = make([]byte, len(marshalledData))
	copy(vmInput.Arguments[2], marshalledData)

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	mectNftTokenKey := computeMECTNFTTokenKey(mectTokenKey, 1)
	err = acnt.SaveKeyValue(mectNftTokenKey, []byte("t"))
	assert.Nil(t, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, err)

	mectData := &mect.MECToken{Value: big.NewInt(0), TokenMetaData: &mect.MetaData{Name: []byte("data")}}
	marshalledData, _ = e.marshaller.Marshal(mectData)
	err = acnt.SaveKeyValue(mectNftTokenKey, marshalledData)
	assert.Nil(t, err)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.NotNil(t, ErrTokenHasValidMetadata)

	_ = acnt.SaveKeyValue(mectNftTokenKey, nil)
	testErr := errors.New("testError")
	accounts.SaveAccountCalled = func(account vmcommon.AccountHandler) error {
		return testErr
	}

	vmInput.Arguments[1] = []byte{2}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidMetadata)

	vmInput.Arguments[1] = []byte{1}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, testErr)
}

func TestMectDeleteMetaData_ProcessBuiltinFunctionAdd(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDelete()
	args.Delete = false
	e, _ := NewMECTDeleteMetadataFunc(args)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmInput.RecipientAddr = e.allowedAddress
	vmInput.Arguments = [][]byte{{1}, {0}, {1}}
	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmInput.Arguments[1] = []byte{1}
	mectMetadata := &mect.MetaData{Name: []byte("something"), Nonce: 1}
	marshalledData, _ := e.marshaller.Marshal(mectMetadata)
	vmInput.Arguments[2] = make([]byte, len(marshalledData))
	copy(vmInput.Arguments[2], marshalledData)

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}

	e.accounts = accounts

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.NotNil(t, vmOutput)
	assert.Nil(t, err)
}

func TestMectDeleteMetaData_ProcessBuiltinFunctionDelete(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDelete()
	e, _ := NewMECTDeleteMetadataFunc(args)

	vmInput := &vmcommon.ContractCallInput{VMInput: vmcommon.VMInput{CallValue: big.NewInt(0)}}
	vmInput.CallerAddr = e.allowedAddress
	vmInput.RecipientAddr = e.allowedAddress
	vmInput.Arguments = [][]byte{{1}, {2}, {1}, {1}}

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		}}

	e.accounts = accounts

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidTokenID)

	vmInput.Arguments[0] = []byte("TOKEN-ababab")
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments[2] = []byte{0}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNonce)

	vmInput.Arguments[2] = []byte{10}
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidArguments)

	vmInput.Arguments[1] = []byte{1}
	vmInput.Arguments[3] = []byte{11}

	vmInput.Arguments = append(vmInput.Arguments, []byte("TOKEN-ababab"), []byte{2})
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, err, ErrInvalidNumOfArgs)

	vmInput.Arguments = append(vmInput.Arguments, []byte{1}, []byte{2}, []byte{4}, []byte{10})
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.NotNil(t, vmOutput)
}
