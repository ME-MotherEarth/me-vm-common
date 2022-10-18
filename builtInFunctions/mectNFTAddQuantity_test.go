package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMECTNFTAddQuantityFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	eqf, err := NewMECTNFTAddQuantityFunc(10, nil, nil, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilMECTNFTStorageHandler, err)

	// nil pause handler
	eqf, err = NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), nil, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	eqf, err = NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilRolesHandler, err)

	// nil epoch handler
	eqf, err = NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, nil)
	require.True(t, check.IfNil(eqf))
	require.Equal(t, ErrNilEpochHandler, err)

	// should work
	eqf, err = NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	require.False(t, check.IfNil(eqf))
	require.NoError(t, err)
}

func TestMectNFTAddQuantity_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	eqf, _ := NewMECTNFTAddQuantityFunc(defaultGasCost, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	eqf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, eqf.funcGasCost)
}

func TestMectNFTAddQuantity_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	eqf, _ := NewMECTNFTAddQuantityFunc(defaultGasCost, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	eqf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				MECTNFTAddQuantity: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, eqf.funcGasCost)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionErrorOnCheckMECTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	// nil vm input
	output, err := eqf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(37),
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	// vm input - invalid number of arguments
	output, err = eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("single arg")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid number of arguments
	output, err = eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("arg0")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid receiver
	output, err = eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 2"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidRcvAddr, err)

	// nil user account
	output, err = eqf.ProcessBuiltinFunction(
		nil,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNilUserAccount, err)

	// not enough gas
	output, err = eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 1,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNotEnoughGas, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, localErr, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := eqf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Error(t, err)
	require.Equal(t, ErrNewNFTDataOnSenderAddress, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"), mectDataBytes)
	output, err := eqf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), {0}, []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrNFTDoesNotHaveMetadata, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}), globalSettingsHandler, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"+"arg1"), mectDataBytes)

	output, err := eqf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrMECTTokenIsPaused, err)
}

func TestMectNFTAddQuantity_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	valueToAdd := big.NewInt(37)
	expectedValue := big.NewInt(0).Add(initialValue, valueToAdd)

	marshaller := &mock.MarshalizerMock{}
	mectRoleHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.MECTRoleNFTAddQuantity, string(action))
			return nil
		},
	}
	eqf, _ := NewMECTNFTAddQuantityFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, mectRoleHandler, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: initialValue,
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	output, err := eqf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), valueToAdd.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := mect.MECToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedValue.Bytes(), finalTokenData.Value.Bytes())
}
