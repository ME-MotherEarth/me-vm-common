package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func TestMECTBurn_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewMECTBurnFunc(10, &mock.MarshalizerMock{}, globalSettingsHandler, 1000, &mock.EpochNotifierStub{})
	_, err := burnFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
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
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotMECTSystemSC)

	input.RecipientAddr = core.MECTSCAddress
	input.GasProvided = burnFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("dst"))
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	input.GasProvided = burnFunc.funcGasCost
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrMECTTokenIsPaused)
}

func TestMECTBurn_ProcessBuiltInFunctionSenderBurns(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewMECTBurnFunc(10, marshaller, globalSettingsHandler, 1000, &mock.EpochNotifierStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: core.MECTSCAddress,
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	mectFrozen := MECTUserMetadata{Frozen: true}
	mectNotFrozen := MECTUserMetadata{Frozen: false}

	mectKey := append(burnFunc.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(100), Properties: mectFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err := burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrMECTIsFrozenForAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	mectToken = &mect.MECToken{Value: big.NewInt(100), Properties: mectNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(mectToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrMECTTokenIsPaused)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return false
	}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)
	assert.True(t, mectToken.Value.Cmp(big.NewInt(90)) == 0)

	value = big.NewInt(100).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	value = big.NewInt(90).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _ = accSnd.AccountDataHandler().RetrieveValue(mectKey)
	assert.Equal(t, len(marshaledData), 0)
}
