package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/data/smartContractResult"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

func createNewMECTDataStorageHandler() *mectDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewMECTDataStorage{
		Accounts:                accounts,
		GlobalSettingsHandler:   &mock.GlobalSettingsHandlerStub{},
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 0,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewMECTDataStorage(args)
	return dataStore
}

func createMockArgsForNewMECTDataStorage() ArgsNewMECTDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewMECTDataStorage{
		Accounts:                accounts,
		GlobalSettingsHandler:   &mock.GlobalSettingsHandlerStub{},
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 0,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
	}
	return args
}

func createNewMECTDataStorageHandlerWithArgs(
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
) *mectDataStorage {
	args := ArgsNewMECTDataStorage{
		Accounts:                accounts,
		GlobalSettingsHandler:   globalSettingsHandler,
		Marshalizer:             &mock.MarshalizerMock{},
		SaveToSystemEnableEpoch: 10,
		EpochNotifier:           &mock.EpochNotifierStub{},
		ShardCoordinator:        &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewMECTDataStorage(args)
	return dataStore
}

func TestNewMECTDataStorage(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	args.Marshalizer = nil
	e, err := NewMECTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArgsForNewMECTDataStorage()
	args.Accounts = nil
	e, err = NewMECTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArgsForNewMECTDataStorage()
	args.ShardCoordinator = nil
	e, err = NewMECTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilShardCoordinator)

	args = createMockArgsForNewMECTDataStorage()
	args.GlobalSettingsHandler = nil
	e, err = NewMECTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilGlobalSettingsHandler)

	args = createMockArgsForNewMECTDataStorage()
	args.EpochNotifier = nil
	e, err = NewMECTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilEpochHandler)

	args = createMockArgsForNewMECTDataStorage()
	e, err = NewMECTDataStorage(args)
	assert.Nil(t, err)
	assert.False(t, e.IsInterfaceNil())
}

func TestMectDataStorage_GetMECTNFTTokenOnDestinationNoDataInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	mectDataGet, _, err := e.GetMECTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, mectData, mectDataGet)
}

func TestMectDataStorage_GetMECTNFTTokenOnDestinationGetDataFromSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	systemAcc, _ := e.getSystemAccount()
	metaData := &mect.MetaData{
		Name: []byte("test"),
	}
	mectDataOnSystemAcc := &mect.MECToken{TokenMetaData: metaData}
	mectMetaDataBytes, _ := args.Marshalizer.Marshal(mectDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectMetaDataBytes)

	mectDataGet, _, err := e.GetMECTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	mectData.TokenMetaData = metaData
	assert.Equal(t, mectData, mectDataGet)
}

func TestMectDataStorage_GetMECTNFTTokenOnDestinationMarshalERR(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	mectDataBytes = append(mectDataBytes, mectDataBytes...)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	_, _, err := e.GetMECTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)

	_, err = e.GetMECTNFTTokenOnSender(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestMectDataStorage_MarshalErrorOnSystemACC(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	systemAcc, _ := e.getSystemAccount()
	metaData := &mect.MetaData{
		Name: []byte("test"),
	}
	mectDataOnSystemAcc := &mect.MECToken{TokenMetaData: metaData}
	mectMetaDataBytes, _ := args.Marshalizer.Marshal(mectDataOnSystemAcc)
	mectMetaDataBytes = append(mectMetaDataBytes, mectMetaDataBytes...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectMetaDataBytes)

	_, _, err := e.GetMECTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestMECTDataStorage_saveDataToSystemAccNotNFTOrMetaData(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	err := e.saveMECTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 0, nil, true)
	assert.Nil(t, err)

	err = e.saveMECTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 1, &mect.MECToken{}, true)
	assert.Nil(t, err)
}

func TestMectDataStorage_SaveMECTNFTTokenNoChangeInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	systemAcc, _ := e.getSystemAccount()
	metaData := &mect.MetaData{
		Name: []byte("test"),
	}
	mectDataOnSystemAcc := &mect.MECToken{TokenMetaData: metaData}
	mectMetaDataBytes, _ := args.Marshalizer.Marshal(mectDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectMetaDataBytes)

	newMetaData := &mect.MetaData{Name: []byte("newName")}
	transferMECTData := &mect.MECToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
	_, err := e.SaveMECTNFTToken([]byte("address"), userAcc, []byte(key), nonce, transferMECTData, false, false)
	assert.Nil(t, err)

	mectDataGet, _, err := e.GetMECTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	mectData.TokenMetaData = metaData
	mectData.Value = big.NewInt(100)
	assert.Equal(t, mectData, mectDataGet)
}

func TestMectDataStorage_SaveMECTNFTTokenWhenQuantityZero(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	nonce := uint64(10)
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
		TokenMetaData: &mect.MetaData{
			Name:  []byte("test"),
			Nonce: nonce,
		},
	}

	tokenIdentifier := "testTkn"
	key := baseMECTKeyPrefix + tokenIdentifier
	mectDataBytes, _ := args.Marshalizer.Marshal(mectData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectDataBytes)

	mectData.Value = big.NewInt(0)
	_, err := e.SaveMECTNFTToken([]byte("address"), userAcc, []byte(key), nonce, mectData, false, false)
	assert.Nil(t, err)

	val, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	assert.Nil(t, val)
	assert.Nil(t, err)

	mectMetaData, err := e.getMECTMetaDataFromSystemAccount(tokenKey)
	assert.Nil(t, err)
	assert.Equal(t, mectData.TokenMetaData, mectMetaData)
}

func TestMectDataStorage_WasAlreadySentToDestinationShard(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewMECTDataStorage(args)

	tickerID := []byte("ticker")
	dstAddress := []byte("dstAddress")
	val, err := e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 0, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	e.flagSendAlwaysEnableEpoch.Reset()
	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return core.MetachainShardId
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	e.flagSendAlwaysEnableEpoch.SetValue(true)

	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return 1
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 5
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	systemAcc, _ := e.getSystemAccount()
	metaData := &mect.MetaData{
		Name: []byte("test"),
	}
	mectDataOnSystemAcc := &mect.MECToken{TokenMetaData: metaData}
	mectMetaDataBytes, _ := args.Marshalizer.Marshal(mectDataOnSystemAcc)
	key := baseMECTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, mectMetaDataBytes)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	e.flagSendAlwaysEnableEpoch.Reset()
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 10
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)
}

func TestMectDataStorage_SaveNFTMetaDataToSystemAccount(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewMECTDataStorage(args)

	e.flagSaveToSystemAccount.Reset()
	err := e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Nil(t, err)

	_ = e.flagSaveToSystemAccount.SetReturningPrevious()

	err = e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Nil(t, err)

	e.flagSendAlwaysEnableEpoch.Reset()
	err = e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Equal(t, err, ErrNilTransactionHandler)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte(core.BuiltInFunctionMECTNFTTransfer + "@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.NotNil(t, err)

	scr.Data = []byte(core.BuiltInFunctionMECTNFTTransfer + "@01@02@03@00")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	tickerID := []byte("TCK")
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
	}
	mectMarshalled, _ := args.Marshalizer.Marshal(mectData)
	scr.Data = []byte(core.BuiltInFunctionMECTNFTTransfer + "@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(mectMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := baseMECTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	mectGetData, _, _ := e.getMECTDigitalTokenDataFromSystemAccount(tokenKey)

	assert.Equal(t, mectData.TokenMetaData, mectGetData.TokenMetaData)
}

func TestMectDataStorage_SaveNFTMetaDataToSystemAccountWithMultiTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewMECTDataStorage(args)
	e.flagSendAlwaysEnableEpoch.Reset()

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	tickerID := []byte("TCK")
	mectData := &mect.MECToken{
		Value: big.NewInt(10),
		TokenMetaData: &mect.MetaData{
			Name: []byte("test"),
		},
	}
	mectMarshalled, _ := args.Marshalizer.Marshal(mectData)
	scr.Data = []byte(core.BuiltInFunctionMultiMECTNFTTransfer + "@00@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(mectMarshalled))
	err := e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiMECTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(mectMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiMECTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@02@10@" +
		hex.EncodeToString(tickerID) + "@01@" + hex.EncodeToString(mectMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := baseMECTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	mectGetData, _, _ := e.getMECTDigitalTokenDataFromSystemAccount(tokenKey)

	assert.Equal(t, mectData.TokenMetaData, mectGetData.TokenMetaData)

	otherTokenKey := append([]byte(key), big.NewInt(2).Bytes()...)
	mectGetData, _, err = e.getMECTDigitalTokenDataFromSystemAccount(otherTokenKey)
	assert.Nil(t, mectGetData)
	assert.Nil(t, err)
}

func TestMectDataStorage_checkCollectionFrozen(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewMECTDataStorage(args)

	e.flagCheckFrozenCollection.SetValue(false)

	acnt, _ := e.accounts.LoadAccount([]byte("address1"))
	userAcc := acnt.(vmcommon.UserAccountHandler)

	tickerID := []byte("TOKEN-ABCDEF")
	mectTokenKey := append(e.keyPrefix, tickerID...)
	err := e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 1, false)
	assert.Nil(t, err)

	e.flagCheckFrozenCollection.SetValue(true)
	err = e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 0, false)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 1, true)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 1, false)
	assert.Nil(t, err)

	tokenData, _ := getMECTDataFromKey(userAcc, mectTokenKey, e.marshaller)

	mectUserMetadata := MECTUserMetadataFromBytes(tokenData.Properties)
	mectUserMetadata.Frozen = false
	tokenData.Properties = mectUserMetadata.ToBytes()
	_ = saveMECTData(userAcc, tokenData, mectTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 1, false)
	assert.Nil(t, err)

	mectUserMetadata.Frozen = true
	tokenData.Properties = mectUserMetadata.ToBytes()
	_ = saveMECTData(userAcc, tokenData, mectTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, mectTokenKey, 1, false)
	assert.Equal(t, err, ErrMECTIsFrozenForAccount)
}

func TestMectDataStorage_AddToLiquiditySystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewMECTDataStorage()
	e, _ := NewMECTDataStorage(args)

	tokenKey := append(e.keyPrefix, []byte("TOKEN-ababab")...)
	nonce := uint64(10)
	err := e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Equal(t, err, ErrNilMECTData)

	systemAcc, _ := e.getSystemAccount()
	mectData := &mect.MECToken{Value: big.NewInt(0)}
	marshalledData, _ := e.marshaller.Marshal(mectData)

	mectNFTTokenKey := computeMECTNFTTokenKey(tokenKey, nonce)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshalledData)

	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Nil(t, err)

	mectData = &mect.MECToken{Value: big.NewInt(10), Reserved: []byte{1}}
	marshalledData, _ = e.marshaller.Marshal(mectData)

	_ = systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshalledData)
	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Nil(t, err)

	mectData, _, _ = e.getMECTDigitalTokenDataFromSystemAccount(mectNFTTokenKey)
	assert.Equal(t, mectData.Value, big.NewInt(20))

	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(-20))
	assert.Nil(t, err)

	mectData, _, _ = e.getMECTDigitalTokenDataFromSystemAccount(mectNFTTokenKey)
	assert.Nil(t, mectData)
}
