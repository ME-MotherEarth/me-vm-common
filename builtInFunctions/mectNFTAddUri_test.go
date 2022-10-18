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

func TestNewMECTNFTAddUriFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	e, err := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, nil, nil, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilMECTNFTStorageHandler, err)

	// nil pause handler
	e, err = NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), nil, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	e, err = NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, 0, &mock.EpochNotifierStub{})
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilRolesHandler, err)

	// nil epoch notifier
	e, err = NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, nil)
	require.True(t, check.IfNil(e))
	require.Equal(t, ErrNilEpochHandler, err)

	// should work
	e, err = NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 1, &mock.EpochNotifierStub{})
	require.False(t, check.IfNil(e))
	require.NoError(t, err)
	require.False(t, e.IsActive())
}

func TestMECTNFTAddUri_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	e, _ := NewMECTNFTAddUriFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	e.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, e.funcGasCost)
}

func TestMECTNFTAddUri_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	e, _ := NewMECTNFTAddUriFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	e.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				MECTNFTAddURI: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, e.funcGasCost)
}

func TestMECTNFTAddUri_ProcessBuiltinFunctionErrorOnCheckInput(t *testing.T) {
	t.Parallel()

	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	// nil vm input
	output, err := e.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})
	output, err := e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"+"arg1"), mectDataBytes)
	output, err := e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewMECTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}), globalSettingsHandler, &mock.MECTRoleHandlerStub{}, 0, &mock.EpochNotifierStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	mectDataBytes, _ := marshaller.Marshal(mectData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.MotherEarthProtectedKeyPrefix+core.MECTKeyIdentifier+"arg0"+"arg1"), mectDataBytes)

	output, err := e.ProcessBuiltinFunction(
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

func TestMECTNFTAddUri_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	URIToAdd := []byte("NewURI")

	mectDataStorage := createNewMECTDataStorageHandler()
	marshaller := &mock.MarshalizerMock{}
	mectRoleHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.MECTRoleNFTAddURI, string(action))
			return nil
		},
	}
	e, _ := NewMECTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, mectDataStorage, &mock.GlobalSettingsHandlerStub{}, mectRoleHandler, 0, &mock.EpochNotifierStub{})

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

	output, err := e.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), URIToAdd},
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

	metaData, _ := mectDataStorage.getMECTMetaDataFromSystemAccount(tokenKey)
	require.Equal(t, metaData.URIs[0], URIToAdd)
}
