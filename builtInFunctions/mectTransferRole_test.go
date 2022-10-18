package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewMECTTransferRoleAddressFunc(t *testing.T) {
	_, err := NewMECTTransferRoleAddressFunc(nil, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	_, err = NewMECTTransferRoleAddressFunc(&mock.AccountsStub{}, nil, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Equal(t, err, ErrNilMarshalizer)

	_, err = NewMECTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, nil, 10, true)
	assert.Equal(t, err, ErrNilEpochHandler)

	e, err := NewMECTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 0, true)
	assert.Equal(t, err, ErrInvalidMaxNumAddresses)
	assert.True(t, check.IfNil(e))

	e, err = NewMECTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionMECTTransferRoleAddAddress)

	e, err = NewMECTTransferRoleAddressFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, 0, &mock.EpochNotifierStub{}, 10, false)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionMECTTransferRoleDeleteAddress)

	e.SetNewGasConfig(nil)
	assert.False(t, e.IsInterfaceNil())
}

func TestMECTTransferRoleProcessBuiltInFunction_Errors(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewMECTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionMECTTransferRoleAddAddress)

	_, err = e.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     nil,
		Function:          "",
		AllowInitFunction: false,
	}

	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrAddressIsNotMECTSystemSC)

	vmInput.CallerAddr = core.MECTSCAddress
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	errNotImplemented := errors.New("not implemented")
	vmInput.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errNotImplemented)

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}
	accounts.SaveAccountCalled = func(account vmcommon.AccountHandler) error {
		return errNotImplemented
	}
	e.maxNumAddresses = 1
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, ErrTooManyTransferAddresses)

	e.maxNumAddresses = 10
	marshaller.Fail = true
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errors.New("MarshalizerMock generic error"))

	systemAcc.Storage[string(append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))] = []byte{1, 1, 1}
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errors.New("MarshalizerMock generic error"))

	marshaller.Fail = false
	systemAcc.Storage[string(append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))] = nil
	_, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Equal(t, err, errNotImplemented)
}

func TestMECTTransferRoleProcessBuiltInFunction_AddNewAddresses(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewMECTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionMECTTransferRoleAddAddress)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.MECTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     vmcommon.SystemAccountAddress,
		Function:          "",
		AllowInitFunction: false,
	}

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ := getMECTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ = getMECTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	e.set = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
	addresses, _, _ = getMECTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 0)
}

func TestMECTTransferRoleIsSenderOrDestinationWithTransferRole(t *testing.T) {
	accounts := &mock.AccountsStub{}
	marshaller := &mock.MarshalizerMock{}
	e, err := NewMECTTransferRoleAddressFunc(accounts, marshaller, 0, &mock.EpochNotifierStub{}, 10, true)
	assert.Nil(t, err)
	assert.Equal(t, e.function, vmcommon.BuiltInFunctionMECTTransferRoleAddAddress)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: core.MECTSCAddress,
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("token"), {1}, {2}, {3}},
		},
		RecipientAddr:     vmcommon.SystemAccountAddress,
		Function:          "",
		AllowInitFunction: false,
	}

	systemAcc := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		return systemAcc, nil
	}

	vmOutput, err := e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)

	addresses, _, _ := getMECTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 3)

	globalSettings, _ := NewMECTGlobalSettingsFunc(accounts, marshaller, true, vmcommon.BuiltInFunctionMECTSetBurnRoleForAll, 0, &mock.EpochNotifierStub{})
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(nil, nil, nil))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], []byte("random")))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), vmInput.Arguments[0]))
	assert.True(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), []byte("random"), vmInput.Arguments[0]))

	e.set = false
	vmOutput, err = e.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	assert.Equal(t, vmOutput.ReturnCode, vmcommon.Ok)
	addresses, _, _ = getMECTRolesForAcnt(e.marshaller, systemAcc, append(transferAddressesKeyPrefix, vmInput.Arguments[0]...))
	assert.Equal(t, len(addresses.Roles), 0)
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(nil, nil, nil))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], []byte("random")))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole(vmInput.Arguments[1], []byte("random"), vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), vmInput.Arguments[2], vmInput.Arguments[0]))
	assert.False(t, globalSettings.IsSenderOrDestinationWithTransferRole([]byte("random"), []byte("random"), vmInput.Arguments[0]))
}
