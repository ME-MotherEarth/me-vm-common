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

func TestNewMECTNFTBurnFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	ebf, err := NewMECTNFTBurnFunc(10, nil, nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilMECTNFTStorageHandler, err)

	// nil pause handler
	ebf, err = NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	ebf, err = NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilRolesHandler, err)

	// should work
	ebf, err = NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})
	require.False(t, check.IfNil(ebf))
	require.NoError(t, err)
}

func TestMECTNFTBurn_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	ebf, _ := NewMECTNFTBurnFunc(defaultGasCost, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})

	ebf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, ebf.funcGasCost)
}

func TestMectNFTBurnFunc_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	ebf, _ := NewMECTNFTBurnFunc(defaultGasCost, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})

	ebf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				MECTNFTBurn: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, ebf.funcGasCost)
}

func TestMectNFTBurnFunc_ProcessBuiltinFunctionErrorOnCheckMECTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})

	// nil vm input
	output, err := ebf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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
	output, err = ebf.ProcessBuiltinFunction(
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

func TestMectNFTBurnFunc_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
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

func TestMectNFTBurnFunc_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler)
	output, err := ebf.ProcessBuiltinFunction(
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

func TestMectNFTBurnFunc_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
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

func TestMectNFTBurnFunc_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"), mectDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
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

func TestMectNFTBurnFunc_ProcessBuiltinFunctionInvalidBurnQuantity(t *testing.T) {
	t.Parallel()

	initialQuantity := big.NewInt(55)
	quantityToBurn := big.NewInt(75)

	marshaller := &mock.MarshalizerMock{}

	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"+"arg1"), mectDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrInvalidNFTQuantity, err)
}

func TestMectNFTBurnFunc_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	ebf, _ := NewMECTNFTBurnFunc(10, createNewMECTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}), globalSettingsHandler, &mock.MECTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"+"arg1"), mectDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), big.NewInt(5).Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrMECTTokenIsPaused, err)
}

func TestMectNFTBurnFunc_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	mectRoleHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.MECTRoleNFTBurn, string(action))
			return nil
		},
	}
	storageHandler := createNewMECTDataStorageHandler()
	ebf, _ := NewMECTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{}, mectRoleHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	nftTokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(nftTokenKey, mectDataBytes)

	_ = storageHandler.saveMECTMetaDataToSystemAccount(userAcc, 0, nftTokenKey, nonce.Uint64(), mectData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), nonce.Uint64(), initialQuantity)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, err := userAcc.AccountDataHandler().RetrieveValue(nftTokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := mect.MECToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}

func TestMectNFTBurnFunc_ProcessBuiltinFunctionWithGlobalBurn(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	storageHandler := createNewMECTDataStorageHandler()
	ebf, _ := NewMECTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			return true
		},
	}, &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errors.New("no burn allowed")
		},
	})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)
	_ = storageHandler.saveMECTMetaDataToSystemAccount(userAcc, 0, tokenKey, nonce.Uint64(), mectData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), nonce.Uint64(), initialQuantity)

	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
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
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}
