package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/parsers"
)

const existsOnShard = byte(1)

type mectDataStorage struct {
	accounts              vmcommon.AccountsAdapter
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	shardCoordinator      vmcommon.Coordinator
	txDataParser          vmcommon.CallArgsParser

	flagSaveToSystemAccount          atomic.Flag
	saveToSystemEnableEpoch          uint32
	flagCheckFrozenCollection        atomic.Flag
	checkFrozenCollectionEnableEpoch uint32
	flagSendAlwaysEnableEpoch        atomic.Flag
	sendAlwaysEnableEpoch            uint32
	flagFixOldTokenLiquidity         atomic.Flag
	fixOldTokenLiquidityEnableEpoch  uint32
}

// ArgsNewMECTDataStorage defines the argument list for new mect data storage handler
type ArgsNewMECTDataStorage struct {
	Accounts                        vmcommon.AccountsAdapter
	GlobalSettingsHandler           vmcommon.MECTGlobalSettingsHandler
	Marshalizer                     vmcommon.Marshalizer
	SaveToSystemEnableEpoch         uint32
	SendAlwaysEnableEpoch           uint32
	FixOldTokenLiquidityEnableEpoch uint32
	EpochNotifier                   vmcommon.EpochNotifier
	ShardCoordinator                vmcommon.Coordinator
}

// NewMECTDataStorage creates a new mect data storage handler
func NewMECTDataStorage(args ArgsNewMECTDataStorage) (*mectDataStorage, error) {
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &mectDataStorage{
		accounts:              args.Accounts,
		globalSettingsHandler: args.GlobalSettingsHandler,
		marshaller:            args.Marshalizer,
		keyPrefix:             []byte(baseMECTKeyPrefix),
		shardCoordinator:      args.ShardCoordinator,
		txDataParser:          parsers.NewCallArgsParser(),

		flagSaveToSystemAccount:          atomic.Flag{},
		saveToSystemEnableEpoch:          args.SaveToSystemEnableEpoch,
		flagCheckFrozenCollection:        atomic.Flag{},
		checkFrozenCollectionEnableEpoch: args.SaveToSystemEnableEpoch,
		flagSendAlwaysEnableEpoch:        atomic.Flag{},
		sendAlwaysEnableEpoch:            args.SendAlwaysEnableEpoch,
		flagFixOldTokenLiquidity:         atomic.Flag{},
		fixOldTokenLiquidityEnableEpoch:  args.FixOldTokenLiquidityEnableEpoch,
	}

	args.EpochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// GetMECTNFTTokenOnSender gets the nft token on sender account
func (e *mectDataStorage) GetMECTNFTTokenOnSender(
	accnt vmcommon.UserAccountHandler,
	mectTokenKey []byte,
	nonce uint64,
) (*mect.MECToken, error) {
	mectData, isNew, err := e.GetMECTNFTTokenOnDestination(accnt, mectTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if isNew {
		return nil, ErrNewNFTDataOnSenderAddress
	}

	return mectData, nil
}

// GetMECTNFTTokenOnDestination gets the nft token on destination account
func (e *mectDataStorage) GetMECTNFTTokenOnDestination(
	accnt vmcommon.UserAccountHandler,
	mectTokenKey []byte,
	nonce uint64,
) (*mect.MECToken, bool, error) {
	mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
	mectData := &mect.MECToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, err := accnt.AccountDataHandler().RetrieveValue(mectNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return mectData, true, nil
	}

	err = e.marshaller.Unmarshal(mectData, marshaledData)
	if err != nil {
		return nil, false, err
	}

	if !e.flagSaveToSystemAccount.IsSet() || nonce == 0 {
		return mectData, false, nil
	}

	mectMetaData, err := e.getMECTMetaDataFromSystemAccount(mectNFTTokenKey)
	if err != nil {
		return nil, false, err
	}
	if mectMetaData != nil {
		mectData.TokenMetaData = mectMetaData
	}

	return mectData, false, nil
}

func (e *mectDataStorage) getMECTDigitalTokenDataFromSystemAccount(
	tokenKey []byte,
) (*mect.MECToken, vmcommon.UserAccountHandler, error) {
	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, nil, err
	}

	marshaledData, err := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, systemAcc, nil
	}

	mectData := &mect.MECToken{}
	err = e.marshaller.Unmarshal(mectData, marshaledData)
	if err != nil {
		return nil, nil, err
	}

	return mectData, systemAcc, nil
}

func (e *mectDataStorage) getMECTMetaDataFromSystemAccount(
	tokenKey []byte,
) (*mect.MetaData, error) {
	mectData, _, err := e.getMECTDigitalTokenDataFromSystemAccount(tokenKey)
	if err != nil {
		return nil, err
	}
	if mectData == nil {
		return nil, nil
	}

	return mectData.TokenMetaData, nil
}

// CheckCollectionIsFrozenForAccount returns
func (e *mectDataStorage) checkCollectionIsFrozenForAccount(
	accnt vmcommon.UserAccountHandler,
	mectTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	if !e.flagCheckFrozenCollection.IsSet() {
		return nil
	}
	if nonce == 0 || isReturnWithError {
		return nil
	}

	mectData := &mect.MECToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, err := accnt.AccountDataHandler().RetrieveValue(mectTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil
	}

	err = e.marshaller.Unmarshal(mectData, marshaledData)
	if err != nil {
		return err
	}

	mectUserMetaData := MECTUserMetadataFromBytes(mectData.Properties)
	if mectUserMetaData.Frozen {
		return ErrMECTIsFrozenForAccount
	}

	return nil
}

func (e *mectDataStorage) checkFrozenPauseProperties(
	acnt vmcommon.UserAccountHandler,
	mectTokenKey []byte,
	nonce uint64,
	mectData *mect.MECToken,
	isReturnWithError bool,
) error {
	err := checkFrozeAndPause(acnt.AddressBytes(), mectTokenKey, mectData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
	err = checkFrozeAndPause(acnt.AddressBytes(), mectNFTTokenKey, mectData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	err = e.checkCollectionIsFrozenForAccount(acnt, mectTokenKey, nonce, isReturnWithError)
	if err != nil {
		return err
	}

	return nil
}

// AddToLiquiditySystemAcc will increase/decrease the liquidity for MECT Tokens on the metadata
func (e *mectDataStorage) AddToLiquiditySystemAcc(
	mectTokenKey []byte,
	nonce uint64,
	transferValue *big.Int,
) error {
	if !e.flagSaveToSystemAccount.IsSet() || !e.flagSendAlwaysEnableEpoch.IsSet() || nonce == 0 {
		return nil
	}

	mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
	mectData, systemAcc, err := e.getMECTDigitalTokenDataFromSystemAccount(mectNFTTokenKey)
	if err != nil {
		return err
	}

	if mectData == nil {
		return ErrNilMECTData
	}

	// old style metaData - nothing to do
	if len(mectData.Reserved) == 0 {
		return nil
	}

	if e.flagFixOldTokenLiquidity.IsSet() {
		// old tokens which were transferred intra shard before the activation of this flag
		if mectData.Value.Cmp(zero) == 0 && transferValue.Cmp(zero) < 0 {
			mectData.Reserved = nil
			return e.marshalAndSaveData(systemAcc, mectData, mectNFTTokenKey)
		}
	}

	mectData.Value.Add(mectData.Value, transferValue)
	if mectData.Value.Cmp(zero) < 0 {
		return ErrInvalidLiquidityForMECT
	}

	if mectData.Value.Cmp(zero) == 0 {
		err = systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, nil)
		if err != nil {
			return err
		}

		return e.accounts.SaveAccount(systemAcc)
	}

	err = e.marshalAndSaveData(systemAcc, mectData, mectNFTTokenKey)
	if err != nil {
		return err
	}

	return nil
}

// SaveMECTNFTToken saves the nft token to the account and system account
func (e *mectDataStorage) SaveMECTNFTToken(
	senderAddress []byte,
	acnt vmcommon.UserAccountHandler,
	mectTokenKey []byte,
	nonce uint64,
	mectData *mect.MECToken,
	mustUpdate bool,
	isReturnWithError bool,
) ([]byte, error) {
	err := e.checkFrozenPauseProperties(acnt, mectTokenKey, nonce, mectData, isReturnWithError)
	if err != nil {
		return nil, err
	}

	mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
	senderShardID := e.shardCoordinator.ComputeId(senderAddress)
	if e.flagSaveToSystemAccount.IsSet() {
		err = e.saveMECTMetaDataToSystemAccount(acnt, senderShardID, mectNFTTokenKey, nonce, mectData, mustUpdate)
		if err != nil {
			return nil, err
		}
	}

	if mectData.Value.Cmp(zero) <= 0 {
		return nil, acnt.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, nil)
	}

	if !e.flagSaveToSystemAccount.IsSet() {
		marshaledData, err := e.marshaller.Marshal(mectData)
		if err != nil {
			return nil, err
		}

		return marshaledData, acnt.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshaledData)
	}

	mectDataOnAccount := &mect.MECToken{
		Type:       mectData.Type,
		Value:      mectData.Value,
		Properties: mectData.Properties,
	}
	marshaledData, err := e.marshaller.Marshal(mectDataOnAccount)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshaledData)
}

func (e *mectDataStorage) saveMECTMetaDataToSystemAccount(
	userAcc vmcommon.UserAccountHandler,
	senderShardID uint32,
	mectNFTTokenKey []byte,
	nonce uint64,
	mectData *mect.MECToken,
	mustUpdate bool,
) error {
	if nonce == 0 {
		return nil
	}
	if mectData.TokenMetaData == nil {
		return nil
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	currentSaveData, err := systemAcc.AccountDataHandler().RetrieveValue(mectNFTTokenKey)
	if !mustUpdate && len(currentSaveData) > 0 {
		return nil
	}

	mectDataOnSystemAcc := &mect.MECToken{
		Type:          mectData.Type,
		Value:         big.NewInt(0),
		TokenMetaData: mectData.TokenMetaData,
		Properties:    make([]byte, e.shardCoordinator.NumberOfShards()),
	}
	if len(currentSaveData) == 0 && e.flagSendAlwaysEnableEpoch.IsSet() {
		mectDataOnSystemAcc.Properties = nil
		mectDataOnSystemAcc.Reserved = []byte{1}

		err = e.setReservedToNilForOldToken(mectDataOnSystemAcc, userAcc, mectNFTTokenKey)
		if err != nil {
			return err
		}
	}

	if !e.flagSendAlwaysEnableEpoch.IsSet() {
		selfID := e.shardCoordinator.SelfId()
		if selfID != core.MetachainShardId {
			mectDataOnSystemAcc.Properties[selfID] = existsOnShard
		}
		if senderShardID != core.MetachainShardId {
			mectDataOnSystemAcc.Properties[senderShardID] = existsOnShard
		}
	}

	return e.marshalAndSaveData(systemAcc, mectDataOnSystemAcc, mectNFTTokenKey)
}

func (e *mectDataStorage) setReservedToNilForOldToken(
	mectDataOnSystemAcc *mect.MECToken,
	userAcc vmcommon.UserAccountHandler,
	mectNFTTokenKey []byte,
) error {
	if !e.flagFixOldTokenLiquidity.IsSet() {
		return nil
	}

	if check.IfNil(userAcc) {
		return ErrNilUserAccount
	}
	dataOnUserAcc, errNotCritical := userAcc.AccountDataHandler().RetrieveValue(mectNFTTokenKey)
	shouldIgnoreToken := errNotCritical != nil || len(dataOnUserAcc) == 0
	if shouldIgnoreToken {
		return nil
	}

	mectDataOnUserAcc := &mect.MECToken{}
	err := e.marshaller.Unmarshal(mectDataOnUserAcc, dataOnUserAcc)
	if err != nil {
		return err
	}

	// tokens which were last moved before flagOptimizeNFTStore keep the mect metaData on the user account
	// these are not compatible with the new liquidity model,so we set the reserved field to nil
	if mectDataOnUserAcc.TokenMetaData != nil {
		mectDataOnSystemAcc.Reserved = nil
	}

	return nil
}

func (e *mectDataStorage) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	mectData *mect.MECToken,
	mectNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(mectData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemAcc)
}

func (e *mectDataStorage) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

//TODO: merge properties in case of shard merge

// WasAlreadySentToDestinationShardAndUpdateState checks whether NFT metadata was sent to destination shard or not
// and saves the destination shard as sent
func (e *mectDataStorage) WasAlreadySentToDestinationShardAndUpdateState(
	tickerID []byte,
	nonce uint64,
	dstAddress []byte,
) (bool, error) {
	if !e.flagSaveToSystemAccount.IsSet() {
		return false, nil
	}

	if nonce == 0 {
		return true, nil
	}
	dstShardID := e.shardCoordinator.ComputeId(dstAddress)
	if dstShardID == e.shardCoordinator.SelfId() {
		return true, nil
	}

	if e.flagSendAlwaysEnableEpoch.IsSet() {
		return false, nil
	}

	if dstShardID == core.MetachainShardId {
		return true, nil
	}
	mectTokenKey := append(e.keyPrefix, tickerID...)
	mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)

	mectData, systemAcc, err := e.getMECTDigitalTokenDataFromSystemAccount(mectNFTTokenKey)
	if err != nil {
		return false, err
	}
	if mectData == nil {
		return false, nil
	}

	if uint32(len(mectData.Properties)) < e.shardCoordinator.NumberOfShards() {
		newSlice := make([]byte, e.shardCoordinator.NumberOfShards())
		for i, val := range mectData.Properties {
			newSlice[i] = val
		}
		mectData.Properties = newSlice
	}

	if mectData.Properties[dstShardID] > 0 {
		return true, nil
	}

	mectData.Properties[dstShardID] = existsOnShard
	return false, e.marshalAndSaveData(systemAcc, mectData, mectNFTTokenKey)
}

// SaveNFTMetaDataToSystemAccount this saves the NFT metadata to the system account even if there was an error in processing
func (e *mectDataStorage) SaveNFTMetaDataToSystemAccount(
	tx data.TransactionHandler,
) error {
	if !e.flagSaveToSystemAccount.IsSet() {
		return nil
	}
	if e.flagSendAlwaysEnableEpoch.IsSet() {
		return nil
	}
	if check.IfNil(tx) {
		return ErrNilTransactionHandler
	}

	sndShardID := e.shardCoordinator.ComputeId(tx.GetSndAddr())
	dstShardID := e.shardCoordinator.ComputeId(tx.GetRcvAddr())
	isCrossShardTxAtDest := sndShardID != dstShardID && e.shardCoordinator.SelfId() == dstShardID
	if !isCrossShardTxAtDest {
		return nil
	}

	function, arguments, err := e.txDataParser.ParseData(string(tx.GetData()))
	if err != nil {
		return nil
	}
	if len(arguments) < 4 {
		return nil
	}

	switch function {
	case core.BuiltInFunctionMECTNFTTransfer:
		return e.addMetaDataToSystemAccountFromNFTTransfer(sndShardID, arguments)
	case core.BuiltInFunctionMultiMECTNFTTransfer:
		return e.addMetaDataToSystemAccountFromMultiTransfer(sndShardID, arguments)
	default:
		return nil
	}
}

func (e *mectDataStorage) addMetaDataToSystemAccountFromNFTTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	if !bytes.Equal(arguments[3], zeroByteArray) {
		mectTransferData := &mect.MECToken{}
		err := e.marshaller.Unmarshal(mectTransferData, arguments[3])
		if err != nil {
			return err
		}
		mectTokenKey := append(e.keyPrefix, arguments[0]...)
		nonce := big.NewInt(0).SetBytes(arguments[1]).Uint64()
		mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)

		return e.saveMECTMetaDataToSystemAccount(nil, sndShardID, mectNFTTokenKey, nonce, mectTransferData, true)
	}
	return nil
}

func (e *mectDataStorage) addMetaDataToSystemAccountFromMultiTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	numOfTransfers := big.NewInt(0).SetBytes(arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(arguments)) < minNumOfArguments {
		return fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	startIndex := uint64(1)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(arguments[tokenStartIndex+1]).Uint64()

		if nonce > 0 && len(arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			mectTransferData := &mect.MECToken{}
			marshaledNFTTransfer := arguments[tokenStartIndex+2]
			err := e.marshaller.Unmarshal(mectTransferData, marshaledNFTTransfer)
			if err != nil {
				return fmt.Errorf("%w for token %s", err, string(tokenID))
			}

			mectTokenKey := append(e.keyPrefix, tokenID...)
			mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
			err = e.saveMECTMetaDataToSystemAccount(nil, sndShardID, mectNFTTokenKey, nonce, mectTransferData, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *mectDataStorage) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagSaveToSystemAccount.SetValue(epoch >= e.saveToSystemEnableEpoch)
	log.Debug("MECT NFT save to system account", "enabled", e.flagSaveToSystemAccount.IsSet())

	e.flagCheckFrozenCollection.SetValue(epoch >= e.checkFrozenCollectionEnableEpoch)
	log.Debug("MECT NFT check frozen collection", "enabled", e.flagCheckFrozenCollection.IsSet())

	e.flagSendAlwaysEnableEpoch.SetValue(epoch >= e.sendAlwaysEnableEpoch)
	log.Debug("MECT send metadata always", "enabled", e.flagSendAlwaysEnableEpoch.IsSet())

	e.flagFixOldTokenLiquidity.SetValue(epoch >= e.fixOldTokenLiquidityEnableEpoch)
	log.Debug("MECT fix old token liquidity", "enabled", e.flagFixOldTokenLiquidity.IsSet())
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectDataStorage) IsInterfaceNil() bool {
	return e == nil
}
