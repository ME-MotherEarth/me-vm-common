package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/data/vm"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

var zero = big.NewInt(0)

type mectTransfer struct {
	baseAlwaysActive
	funcGasCost           uint64
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler
	payableHandler        vmcommon.PayableChecker
	shardCoordinator      vmcommon.Coordinator
	mutExecution          sync.RWMutex

	rolesHandler                   vmcommon.MECTRoleHandler
	transferToMetaEnableEpoch      uint32
	flagTransferToMeta             atomic.Flag
	checkCorrectTokenIDEnableEpoch uint32
	flagCheckCorrectTokenID        atomic.Flag
}

// NewMECTTransferFunc returns the mect transfer built-in function component
func NewMECTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler,
	shardCoordinator vmcommon.Coordinator,
	rolesHandler vmcommon.MECTRoleHandler,
	transferToMetaEnableEpoch uint32,
	checkCorrectTokenIDEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*mectTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &mectTransfer{
		funcGasCost:                    funcGasCost,
		marshaller:                     marshaller,
		keyPrefix:                      []byte(baseMECTKeyPrefix),
		globalSettingsHandler:          globalSettingsHandler,
		payableHandler:                 &disabledPayableHandler{},
		shardCoordinator:               shardCoordinator,
		rolesHandler:                   rolesHandler,
		checkCorrectTokenIDEnableEpoch: checkCorrectTokenIDEnableEpoch,
		transferToMetaEnableEpoch:      transferToMetaEnableEpoch,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *mectTransfer) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagTransferToMeta.SetValue(epoch >= e.transferToMetaEnableEpoch)
	log.Debug("MECT transfer to metachain flag", "enabled", e.flagTransferToMeta.IsSet())
	e.flagCheckCorrectTokenID.SetValue(epoch >= e.checkCorrectTokenIDEnableEpoch)
	log.Debug("MECT transfer check correct tokenID for transfer role", "enabled", e.flagCheckCorrectTokenID.IsSet())
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTTransfer
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT transfer function calls
func (e *mectTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicMECTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	isInvalidTransferToMeta := e.shardCoordinator.ComputeId(vmInput.RecipientAddr) == core.MetachainShardId && !e.flagTransferToMeta.IsSet()
	if isInvalidTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	tokenID := vmInput.Arguments[0]

	keyToCheck := mectTokenKey
	if e.flagCheckCorrectTokenID.IsSet() {
		keyToCheck = tokenID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(keyToCheck, mectTokenKey, vmInput.CallerAddr, vmInput.RecipientAddr, e.globalSettingsHandler, e.rolesHandler, acntSnd, acntDst, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntSnd) {
		// gas is paid only by sender
		if vmInput.GasProvided < e.funcGasCost {
			return nil, ErrNotEnoughGas
		}

		err = addToMECTBalance(acntSnd, mectTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsMECTTransfer)
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if !check.IfNil(acntDst) {
		err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsMECTTransfer)
		if err != nil {
			return nil, err
		}

		err = addToMECTBalance(acntDst, mectTokenKey, value, e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		if isSCCallAfter {
			vmOutput.GasRemaining, err = vmcommon.SafeSubUint64(vmInput.GasProvided, e.funcGasCost)
			var callArgs [][]byte
			if len(vmInput.Arguments) > core.MinLenArgumentsMECTTransfer+1 {
				callArgs = vmInput.Arguments[core.MinLenArgumentsMECTTransfer+1:]
			}

			addOutputTransferToVMOutput(
				vmInput.CallerAddr,
				string(vmInput.Arguments[core.MinLenArgumentsMECTTransfer]),
				callArgs,
				vmInput.RecipientAddr,
				vmInput.GasLocked,
				vmInput.CallType,
				vmOutput)

			addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTTransfer), tokenID, 0, value, vmInput.CallerAddr, acntDst.AddressBytes())
			return vmOutput, nil
		}

		if vmInput.CallType == vm.AsynchronousCallBack && check.IfNil(acntSnd) {
			// gas was already consumed on sender shard
			vmOutput.GasRemaining = vmInput.GasProvided
		}

		addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTTransfer), tokenID, 0, value, vmInput.CallerAddr, acntDst.AddressBytes())
		return vmOutput, nil
	}

	// cross-shard MECT transfer call through a smart contract
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			core.BuiltInFunctionMECTTransfer,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTTransfer), tokenID, 0, value, vmInput.CallerAddr, vmInput.RecipientAddr)
	return vmOutput, nil
}

func addOutputTransferToVMOutput(
	senderAddress []byte,
	function string,
	arguments [][]byte,
	recipient []byte,
	gasLocked uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	mectTransferTxData := function
	for _, arg := range arguments {
		mectTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Value:         big.NewInt(0),
		GasLimit:      vmOutput.GasRemaining,
		GasLocked:     gasLocked,
		Data:          []byte(mectTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
	vmOutput.GasRemaining = 0
}

func addToMECTBalance(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	value *big.Int,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	mectData, err := getMECTDataFromKey(userAcnt, key, marshaller)
	if err != nil {
		return err
	}

	if mectData.Type != uint32(core.Fungible) {
		return ErrOnlyFungibleTokensHaveBalanceTransfer
	}

	err = checkFrozeAndPause(userAcnt.AddressBytes(), key, mectData, globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	mectData.Value.Add(mectData.Value, value)
	if mectData.Value.Cmp(zero) < 0 {
		return ErrInsufficientFunds
	}

	err = saveMECTData(userAcnt, mectData, key, marshaller)
	if err != nil {
		return err
	}

	return nil
}

func checkFrozeAndPause(
	senderAddr []byte,
	key []byte,
	mectData *mect.MECToken,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if bytes.Equal(senderAddr, core.MECTSCAddress) {
		return nil
	}

	mectUserMetaData := MECTUserMetadataFromBytes(mectData.Properties)
	if mectUserMetaData.Frozen {
		return ErrMECTIsFrozenForAccount
	}

	if globalSettingsHandler.IsPaused(key) {
		return ErrMECTTokenIsPaused
	}

	return nil
}

func arePropertiesEmpty(properties []byte) bool {
	for _, property := range properties {
		if property != 0 {
			return false
		}
	}
	return true
}

func saveMECTData(
	userAcnt vmcommon.UserAccountHandler,
	mectData *mect.MECToken,
	key []byte,
	marshaller vmcommon.Marshalizer,
) error {
	isValueZero := mectData.Value.Cmp(zero) == 0
	if isValueZero && arePropertiesEmpty(mectData.Properties) {
		return userAcnt.AccountDataHandler().SaveKeyValue(key, nil)
	}

	marshaledData, err := marshaller.Marshal(mectData)
	if err != nil {
		return err
	}

	return userAcnt.AccountDataHandler().SaveKeyValue(key, marshaledData)
}

func getMECTDataFromKey(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	marshaller vmcommon.Marshalizer,
) (*mect.MECToken, error) {
	mectData := &mect.MECToken{Value: big.NewInt(0), Type: uint32(core.Fungible)}
	marshaledData, err := userAcnt.AccountDataHandler().RetrieveValue(key)
	if err != nil || len(marshaledData) == 0 {
		return mectData, nil
	}

	err = marshaller.Unmarshal(mectData, marshaledData)
	if err != nil {
		return nil, err
	}

	return mectData, nil
}

// will return nil if transfer is not limited
// if we are at sender shard, the sender or the destination must have the transfer role
// we cannot transfer a limited mect to destination shard, as there we do not know if that token was transferred or not
// by an account with transfer account
func checkIfTransferCanHappenWithLimitedTransfer(
	tokenID []byte, mectTokenKey []byte,
	senderAddress, destinationAddress []byte,
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler,
	roleHandler vmcommon.MECTRoleHandler,
	acntSnd, acntDst vmcommon.UserAccountHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if check.IfNil(acntSnd) {
		return nil
	}
	if !globalSettingsHandler.IsLimitedTransfer(mectTokenKey) {
		return nil
	}

	if globalSettingsHandler.IsSenderOrDestinationWithTransferRole(senderAddress, destinationAddress, tokenID) {
		return nil
	}

	errSender := roleHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.MECTRoleTransfer))
	if errSender == nil {
		return nil
	}

	errDestination := roleHandler.CheckAllowedToExecute(acntDst, tokenID, []byte(core.MECTRoleTransfer))
	return errDestination
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *mectTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectTransfer) IsInterfaceNil() bool {
	return e == nil
}
