package builtInFunctions

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/data/vm"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestMECTTransfer_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	shardC := &mock.ShardCoordinatorStub{}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		shardC,
		&mock.MECTRoleHandlerStub{},
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})
	_, err := transferFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	input.GasProvided = transferFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("address"))
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	input.GasProvided = transferFunc.funcGasCost
	input.RecipientAddr = core.MECTSCAddress
	shardC.ComputeIdCalled = func(address []byte) uint32 {
		return core.MetachainShardId
	}
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInvalidRcvAddr)
}

func TestMECTTransfer_ProcessBuiltInFunctionSingleShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRoleHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.MECTRoleTransfer, string(action))
			return nil
		},
	}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		mectRoleHandler,
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(90)) == 0)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(10)) == 0)
}

func TestMECTTransfer_ProcessBuiltInFunctionSenderInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.MECTRoleHandlerStub{},
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)
	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestMECTTransfer_ProcessBuiltInFunctionDestInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.MECTRoleHandlerStub{},
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accDst := mock.NewUserAccount([]byte("dst"))

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)
	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{}
	marshaledData, _ := accDst.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(10)) == 0)
	assert.Equal(t, uint64(0), vmOutput.GasRemaining)
}

func TestMECTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	mectGlobalSettingsFunc, _ := NewMECTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionMECTPause, 0, &mock.EpochNotifierStub{})
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		mectGlobalSettingsFunc,
		&mock.ShardCoordinatorStub{},
		&mock.MECTRoleHandlerStub{},
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	mectFrozen := MECTUserMetadata{Frozen: true}
	mectNotFrozen := MECTUserMetadata{Frozen: false}

	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100), Properties: mectFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrMECTIsFrozenForAccount)

	mectToken = &mect.MECToken{Value: big.NewInt(100), Properties: mectNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	mectToken = &mect.MECToken{Value: big.NewInt(100), Properties: mectFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(mectToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrMECTIsFrozenForAccount)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(100)) == 0)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	mectToken = &mect.MECToken{Value: big.NewInt(100), Properties: mectNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(mectToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	mectGlobal := MECTGlobalMetadata{Paused: true}
	pauseKey := []byte(baseMECTKeyPrefix + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, mectGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	input.ReturnCallAfterError = false
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrMECTTokenIsPaused)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestMECTTransfer_SndDstWithLimitedTransfer(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	rolesHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if bytes.Equal(action, []byte(core.MECTRoleTransfer)) {
				return ErrActionNotAllowed
			}
			return nil
		},
	}
	mectGlobalSettingsFunc, _ := NewMECTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionMECTSetLimitedTransfer, 0, &mock.EpochNotifierStub{})
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		mectGlobalSettingsFunc,
		&mock.ShardCoordinatorStub{},
		rolesHandler,
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	mectToken = &mect.MECToken{Value: big.NewInt(100)}
	marshaledData, _ = marshaller.Marshal(mectToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	mectGlobal := MECTGlobalMetadata{LimitedTransfer: true}
	pauseKey := []byte(baseMECTKeyPrefix + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, mectGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	_, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	input.ReturnCallAfterError = false
	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accSnd.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accDst.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestMECTTransfer_ProcessBuiltInFunctionOnAsyncCallBack(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		marshaller,
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.MECTRoleHandlerStub{},
		1000,
		0,
		&mock.EpochNotifierStub{},
	)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			CallType:    vm.AsynchronousCallBack,
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount(core.MECTSCAddress)

	mectKey := append(transferFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	marshaledData, _ = accDst.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(10)) == 0)

	assert.Equal(t, vmOutput.GasRemaining, input.GasProvided)

	vmOutput, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	vmOutput.GasRemaining = input.GasProvided - transferFunc.funcGasCost

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestMECTTransfer_EpochChange(t *testing.T) {
	t.Parallel()

	var functionHandler vmcommon.EpochSubscriberHandler
	notifier := &mock.EpochNotifierStub{
		RegisterNotifyHandlerCalled: func(handler vmcommon.EpochSubscriberHandler) {
			functionHandler = handler
		},
	}
	transferFunc, _ := NewMECTTransferFunc(
		10,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.ShardCoordinatorStub{},
		&mock.MECTRoleHandlerStub{},
		1,
		2,
		notifier,
	)

	functionHandler.EpochConfirmed(0, 0)
	assert.False(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())

	functionHandler.EpochConfirmed(1, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.False(t, transferFunc.flagCheckCorrectTokenID.IsSet())

	functionHandler.EpochConfirmed(2, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())

	functionHandler.EpochConfirmed(3, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())

	functionHandler.EpochConfirmed(4, 0)
	assert.True(t, transferFunc.flagTransferToMeta.IsSet())
	assert.True(t, transferFunc.flagCheckCorrectTokenID.IsSet())
}
