package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/data/vm"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

const baseMECTKeyPrefix = core.MotherEarthProtectedKeyPrefix + core.MECTKeyIdentifier

var oneValue = big.NewInt(1)
var zeroByteArray = []byte{0}

type mectNFTTransfer struct {
	baseAlwaysActive
	keyPrefix                      []byte
	marshaller                     vmcommon.Marshalizer
	globalSettingsHandler          vmcommon.ExtendedMECTGlobalSettingsHandler
	payableHandler                 vmcommon.PayableChecker
	funcGasCost                    uint64
	accounts                       vmcommon.AccountsAdapter
	shardCoordinator               vmcommon.Coordinator
	gasConfig                      vmcommon.BaseOperationCost
	mutExecution                   sync.RWMutex
	rolesHandler                   vmcommon.MECTRoleHandler
	mectStorageHandler             vmcommon.MECTNFTStorageHandler
	transferToMetaEnableEpoch      uint32
	flagTransferToMeta             atomic.Flag
	check0TransferEnableEpoch      uint32
	flagCheck0Transfer             atomic.Flag
	checkCorrectTokenIDEnableEpoch uint32
	flagCheckCorrectTokenID        atomic.Flag
}

// NewMECTNFTTransferFunc returns the mect NFT transfer built-in function component
func NewMECTNFTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	rolesHandler vmcommon.MECTRoleHandler,
	transferToMetaEnableEpoch uint32,
	checkZeroTransferEnableEpoch uint32,
	checkCorrectTokenIDEnableEpoch uint32,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
	epochNotifier vmcommon.EpochNotifier,
) (*mectNFTTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
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
	if check.IfNil(mectStorageHandler) {
		return nil, ErrNilMECTNFTStorageHandler
	}

	e := &mectNFTTransfer{
		keyPrefix:                      []byte(baseMECTKeyPrefix),
		marshaller:                     marshaller,
		globalSettingsHandler:          globalSettingsHandler,
		funcGasCost:                    funcGasCost,
		accounts:                       accounts,
		shardCoordinator:               shardCoordinator,
		gasConfig:                      gasConfig,
		mutExecution:                   sync.RWMutex{},
		payableHandler:                 &disabledPayableHandler{},
		rolesHandler:                   rolesHandler,
		transferToMetaEnableEpoch:      transferToMetaEnableEpoch,
		check0TransferEnableEpoch:      checkZeroTransferEnableEpoch,
		checkCorrectTokenIDEnableEpoch: checkCorrectTokenIDEnableEpoch,
		mectStorageHandler:             mectStorageHandler,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *mectNFTTransfer) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagTransferToMeta.SetValue(epoch >= e.transferToMetaEnableEpoch)
	log.Debug("MECT NFT transfer to metachain flag", "enabled", e.flagTransferToMeta.IsSet())
	e.flagCheck0Transfer.SetValue(epoch >= e.check0TransferEnableEpoch)
	log.Debug("MECT NFT transfer check zero transfer", "enabled", e.flagCheck0Transfer.IsSet())
	e.flagCheckCorrectTokenID.SetValue(epoch >= e.checkCorrectTokenIDEnableEpoch)
	log.Debug("MECT NFT transfer check correct tokenID for transfer role", "enabled", e.flagCheckCorrectTokenID.IsSet())
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *mectNFTTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT transfer roles function call
// Requires 4 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to transfer
// arg3 - destination address
// if cross-shard, the rest of arguments will be filled inside the SCR
func (e *mectNFTTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicMECTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 4 {
		return nil, ErrInvalidArguments
	}

	if bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return e.processNFTTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	tickerID := vmInput.Arguments[0]
	mectTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])

	mectTransferData := &mect.MECToken{}
	if !bytes.Equal(vmInput.Arguments[3], zeroByteArray) {
		marshaledNFTTransfer := vmInput.Arguments[3]
		err = e.marshaller.Unmarshal(mectTransferData, marshaledNFTTransfer)
		if err != nil {
			return nil, err
		}
	} else {
		mectTransferData.Value = big.NewInt(0).Set(value)
		mectTransferData.Type = uint32(core.NonFungible)
	}

	err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsMECTNFTTransfer)
	if err != nil {
		return nil, err
	}
	err = e.addNFTToDestination(vmInput.CallerAddr, vmInput.RecipientAddr, acntDst, mectTransferData, mectTokenKey, nonce, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	// no need to consume gas on destination - sender already paid for it
	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	if len(vmInput.Arguments) > core.MinLenArgumentsMECTNFTTransfer && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsMECTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsMECTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsMECTNFTTransfer]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTTransfer), vmInput.Arguments[0], nonce, value, vmInput.CallerAddr, acntDst.AddressBytes())

	return vmOutput, nil
}

func (e *mectNFTTransfer) processNFTTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[3]
	if len(dstAddress) != len(vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, not a valid destination address", ErrInvalidArguments)
	}
	if bytes.Equal(dstAddress, vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, can not transfer to self", ErrInvalidArguments)
	}
	isInvalidTransferToMeta := e.shardCoordinator.ComputeId(dstAddress) == core.MetachainShardId && !e.flagTransferToMeta.IsSet()
	if isInvalidTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}
	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	tickerID := vmInput.Arguments[0]
	mectTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	mectData, err := e.mectStorageHandler.GetMECTNFTTokenOnSender(acntSnd, mectTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if mectData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}
	if e.flagCheck0Transfer.IsSet() && quantityToTransfer.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}
	mectData.Value.Sub(mectData.Value, quantityToTransfer)

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, nonce, mectData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	mectData.Value.Set(quantityToTransfer)

	var userAccount vmcommon.UserAccountHandler
	if e.shardCoordinator.SelfId() == e.shardCoordinator.ComputeId(dstAddress) {
		accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
		if errLoad != nil {
			return nil, errLoad
		}

		var ok bool
		userAccount, ok = accountHandler.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = e.payableHandler.CheckPayable(vmInput, dstAddress, core.MinLenArgumentsMECTNFTTransfer)
		if err != nil {
			return nil, err
		}
		err = e.addNFTToDestination(vmInput.CallerAddr, dstAddress, userAccount, mectData, mectTokenKey, nonce, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(userAccount)
		if err != nil {
			return nil, err
		}
	} else {
		err = e.mectStorageHandler.AddToLiquiditySystemAcc(mectTokenKey, nonce, big.NewInt(0).Neg(quantityToTransfer))
		if err != nil {
			return nil, err
		}
	}

	tokenID := mectTokenKey
	if e.flagCheckCorrectTokenID.IsSet() {
		tokenID = tickerID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(tokenID, mectTokenKey, acntSnd.AddressBytes(), dstAddress, e.globalSettingsHandler, e.rolesHandler, acntSnd, userAccount, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}
	err = e.createNFTOutputTransfers(vmInput, vmOutput, mectData, dstAddress, tickerID, nonce)
	if err != nil {
		return nil, err
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTTransfer), vmInput.Arguments[0], nonce, quantityToTransfer, vmInput.CallerAddr, dstAddress)

	return vmOutput, nil
}

func (e *mectNFTTransfer) createNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	mectTransferData *mect.MECToken,
	dstAddress []byte,
	tickerID []byte,
	nonce uint64,
) error {
	nftTransferCallArgs := make([][]byte, 0)
	nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[:3]...)

	wasAlreadySent, err := e.mectStorageHandler.WasAlreadySentToDestinationShardAndUpdateState(tickerID, nonce, dstAddress)
	if err != nil {
		return err
	}

	if !wasAlreadySent || mectTransferData.Value.Cmp(oneValue) == 0 {
		marshaledNFTTransfer, err := e.marshaller.Marshal(mectTransferData)
		if err != nil {
			return err
		}

		gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
		if gasForTransfer > vmOutput.GasRemaining {
			return ErrNotEnoughGas
		}
		vmOutput.GasRemaining -= gasForTransfer
		nftTransferCallArgs = append(nftTransferCallArgs, marshaledNFTTransfer)
	} else {
		nftTransferCallArgs = append(nftTransferCallArgs, zeroByteArray)
	}

	if len(vmInput.Arguments) > core.MinLenArgumentsMECTNFTTransfer {
		nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[4:]...)
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, dstAddress, core.MinLenArgumentsMECTNFTTransfer)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			vmInput.CallerAddr,
			dstAddress,
			core.BuiltInFunctionMECTNFTTransfer,
			nftTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsMECTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsMECTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsMECTNFTTransfer]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *mectNFTTransfer) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	mectDataToTransfer *mect.MECToken,
	mectTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	currentMECTData, _, err := e.mectStorageHandler.GetMECTNFTTokenOnDestination(userAccount, mectTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, mectTokenKey, currentMECTData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(mectDataToTransfer.Value)
	mectDataToTransfer.Value.Add(mectDataToTransfer.Value, currentMECTData.Value)
	_, err = e.mectStorageHandler.SaveMECTNFTToken(sndAddress, userAccount, mectTokenKey, nonce, mectDataToTransfer, false, isReturnWithError)
	if err != nil {
		return err
	}

	isSameShard := e.shardCoordinator.SameShard(sndAddress, dstAddress)
	if !isSameShard {
		err = e.mectStorageHandler.AddToLiquiditySystemAcc(mectTokenKey, nonce, transferValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func addNFTTransferToVMOutput(
	senderAddress []byte,
	recipient []byte,
	funcToCall string,
	arguments [][]byte,
	gasLocked uint64,
	gasLimit uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	nftTransferTxData := funcToCall
	for _, arg := range arguments {
		nftTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Value:         big.NewInt(0),
		GasLimit:      gasLimit,
		GasLocked:     gasLocked,
		Data:          []byte(nftTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTTransfer) IsInterfaceNil() bool {
	return e == nil
}
