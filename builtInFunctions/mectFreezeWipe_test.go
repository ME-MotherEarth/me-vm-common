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

func TestMECTFreezeWipe_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewMECTFreezeWipeFunc(marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotMECTSystemSC)

	input.CallerAddr = core.MECTSCAddress
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	input.RecipientAddr = []byte("dst")
	acnt := mock.NewUserAccount(input.RecipientAddr)
	vmOutput, err := freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	frozenAmount := big.NewInt(42)
	mectToken := &mect.MECToken{
		Value: frozenAmount,
	}
	mectKey := append(freeze.keyPrefix, key...)
	marshaledData, _ := acnt.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)

	mectUserData := MECTUserMetadataFromBytes(mectToken.Properties)
	assert.True(t, mectUserData.Frozen)
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, frozenAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}

func TestMECTFreezeWipe_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewMECTFreezeWipeFunc(marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	key := []byte("key")

	input.Arguments = [][]byte{key}
	input.CallerAddr = core.MECTSCAddress
	input.RecipientAddr = []byte("dst")
	mectKey := append(freeze.keyPrefix, key...)
	mectToken := &mect.MECToken{Value: big.NewInt(10)}
	marshaledData, _ := freeze.marshaller.Marshal(mectToken)
	acnt := mock.NewUserAccount(input.RecipientAddr)
	_ = acnt.AccountDataHandler().SaveKeyValue(mectKey, marshaledData)

	_, err = freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	mectToken = &mect.MECToken{}
	marshaledData, _ = acnt.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)

	mectUserData := MECTUserMetadataFromBytes(mectToken.Properties)
	assert.True(t, mectUserData.Frozen)

	unFreeze, _ := NewMECTFreezeWipeFunc(marshaller, false, false)
	_, err = unFreeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	marshaledData, _ = acnt.AccountDataHandler().RetrieveValue(mectKey)
	_ = marshaller.Unmarshal(mectToken, marshaledData)

	mectUserData = MECTUserMetadataFromBytes(mectToken.Properties)
	assert.False(t, mectUserData.Frozen)

	// cannot wipe if account is not frozen
	wipe, _ := NewMECTFreezeWipeFunc(marshaller, false, true)
	_, err = wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.Equal(t, ErrCannotWipeAccountNotFrozen, err)

	marshaledData, _ = acnt.AccountDataHandler().RetrieveValue(mectKey)
	assert.NotEqual(t, 0, len(marshaledData))

	// can wipe as account is frozen
	metaData := MECTUserMetadata{Frozen: true}
	wipedAmount := big.NewInt(42)
	mectToken = &mect.MECToken{
		Value:      wipedAmount,
		Properties: metaData.ToBytes(),
	}
	mectTokenBytes, _ := marshaller.Marshal(mectToken)
	err = acnt.AccountDataHandler().SaveKeyValue(mectKey, mectTokenBytes)
	assert.NoError(t, err)

	wipe, _ = NewMECTFreezeWipeFunc(marshaller, false, true)
	vmOutput, err := wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _ = acnt.AccountDataHandler().RetrieveValue(mectKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, wipedAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}
