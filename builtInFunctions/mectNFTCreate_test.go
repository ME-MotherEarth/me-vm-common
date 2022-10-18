package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/data/vm"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNftCreateWithStubArguments() *mectNFTCreate {
	nftCreate, _ := NewMECTNFTCreateFunc(
		1,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)

	return nftCreate
}

func TestNewMECTNFTCreateFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		nil,
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilMarshalizer, err)

	nftCreate, err = NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		nil,
		&mock.MECTRoleHandlerStub{},
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilGlobalSettingsHandler, err)

	nftCreate, err = NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		nil,
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilRolesHandler, err)

	nftCreate, err = NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		nil,
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilMECTNFTStorageHandler, err)

	nftCreate, err = NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		nil,
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilEpochHandler, err)
}

func TestNewMECTNFTCreateFunc(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		createNewMECTDataStorageHandler(),
		&mock.AccountsStub{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.False(t, check.IfNil(nftCreate))
	assert.Nil(t, err)
}

func TestMectNFTCreate_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	nftCreate.SetNewGasConfig(nil)
	assert.Equal(t, uint64(1), nftCreate.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftCreate.gasConfig)

	gasCost := createMockGasCost()
	nftCreate.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.MECTNFTCreate, nftCreate.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftCreate.gasConfig)
}

func TestMectNFTCreate_ProcessBuiltinFunctionInvalidArguments(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: []byte("recipient"),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilUserAccount, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNotEnoughGas, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender.AddressBytes(),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), []byte("arg2")},
			GasProvided: 1,
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
}

func TestMectNFTCreate_ProcessBuiltinFunctionNotAllowedToExecute(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	mectDataStorage := createNewMECTDataStorageHandler()
	nftCreate, _ := NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				return expectedErr
			},
		},
		mectDataStorage,
		mectDataStorage.accounts,
		0,
		&mock.EpochNotifierStub{},
	)
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  make([][]byte, 7),
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, expectedErr, err)
}

func TestMectNFTCreate_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	mectDataStorage := createNewMECTDataStorageHandler()
	firstCheck := true
	mectRoleHandler := &mock.MECTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if firstCheck {
				assert.Equal(t, core.MECTRoleNFTCreate, string(action))
				firstCheck = false
			} else {
				assert.Equal(t, core.MECTRoleNFTAddQuantity, string(action))
			}
			return nil
		},
	}
	nftCreate, _ := NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		mectRoleHandler,
		mectDataStorage,
		mectDataStorage.accounts,
		0,
		&mock.EpochNotifierStub{},
	)
	address := bytes.Repeat([]byte{1}, 32)
	sender := mock.NewUserAccount(address)
	//add some data in the trie, otherwise the creation will fail (it won't happen in real case usage as the create NFT
	//will be called after the creation permission was set in the account's data)
	_ = sender.AccountDataHandler().SaveKeyValue([]byte("key"), []byte("value"))

	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
			},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	createdMect, latestNonce := readNFTData(t, sender, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedMect := &mect.MECToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedMect, createdMect)

	tokenMetaData := &mect.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    address,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseMECTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	mectData, _, _ := mectDataStorage.getMECTDigitalTokenDataFromSystemAccount(tokenKey)
	assert.Equal(t, tokenMetaData, mectData.TokenMetaData)
	assert.Equal(t, mectData.Value, quantity)

	mectDataBytes := vmOutput.Logs[0].Topics[3]
	var mectDataFromLog mect.MECToken
	_ = nftCreate.marshaller.Unmarshal(&mectDataFromLog, mectDataBytes)
	require.Equal(t, mectData.TokenMetaData, mectDataFromLog.TokenMetaData)
}

func TestMectNFTCreate_ProcessBuiltinFunctionWithExecByCaller(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	mectDataStorage := createNewMECTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts)
	_ = mectDataStorage.flagSaveToSystemAccount.SetReturningPrevious()
	nftCreate, _ := NewMECTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.MECTRoleHandlerStub{},
		mectDataStorage,
		mectDataStorage.accounts,
		0,
		&mock.EpochNotifierStub{},
	)
	address := bytes.Repeat([]byte{1}, 32)
	userAddress := bytes.Repeat([]byte{2}, 32)
	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: userAddress,
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
				address,
			},
			CallType: vm.ExecOnDestByCaller,
		},
		RecipientAddr: userAddress,
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	roleAcc, _ := nftCreate.getAccount(address)

	createdMect, latestNonce := readNFTData(t, roleAcc, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedMect := &mect.MECToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedMect, createdMect)

	tokenMetaData := &mect.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    userAddress,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseMECTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	metaData, _ := mectDataStorage.getMECTMetaDataFromSystemAccount(tokenKey)
	assert.Equal(t, tokenMetaData, metaData)
}

func readNFTData(t *testing.T, account vmcommon.UserAccountHandler, marshaller vmcommon.Marshalizer, tokenID []byte, nonce uint64, _ []byte) (*mect.MECToken, uint64) {
	nonceKey := getNonceKey(tokenID)
	latestNonceBytes, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()

	createdTokenID := []byte(baseMECTKeyPrefix)
	createdTokenID = append(createdTokenID, tokenID...)
	tokenKey := computeMECTNFTTokenKey(createdTokenID, nonce)
	data, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(tokenKey)
	require.Nil(t, err)

	mectData := &mect.MECToken{}
	err = marshaller.Unmarshal(mectData, data)
	require.Nil(t, err)

	return mectData, latestNonce
}
