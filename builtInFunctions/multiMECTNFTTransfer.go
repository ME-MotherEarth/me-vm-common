package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectNFTMultiTransfer struct {
	*baseEnabled
	keyPrefix                      []byte
	marshaller                     vmcommon.Marshalizer
	globalSettingsHandler          vmcommon.ExtendedMECTGlobalSettingsHandler
	payableHandler                 vmcommon.PayableChecker
	funcGasCost                    uint64
	accounts                       vmcommon.AccountsAdapter
	shardCoordinator               vmcommon.Coordinator
	gasConfig                      vmcommon.BaseOperationCost
	mutExecution                   sync.RWMutex
	mectStorageHandler             vmcommon.MECTNFTStorageHandler
	rolesHandler                   vmcommon.MECTRoleHandler
	transferToMetaEnableEpoch      uint32
	flagTransferToMeta             atomic.Flag
	checkCorrectTokenIDEnableEpoch uint32
	flagCheckCorrectTokenID        atomic.Flag
}

const argumentsPerTransfer = uint64(3)

// NewMECTNFTMultiTransferFunc returns the mect NFT multi transfer built-in function component
func NewMECTNFTMultiTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
	roleHandler vmcommon.MECTRoleHandler,
	transferToMetaEnableEpoch uint32,
	checkCorrectTokenIDEnableEpoch uint32,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
) (*mectNFTMultiTransfer, error) {
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
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(roleHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(mectStorageHandler) {
		return nil, ErrNilMECTNFTStorageHandler
	}

	e := &mectNFTMultiTransfer{
		keyPrefix:                      []byte(baseMECTKeyPrefix),
		marshaller:                     marshaller,
		globalSettingsHandler:          globalSettingsHandler,
		funcGasCost:                    funcGasCost,
		accounts:                       accounts,
		shardCoordinator:               shardCoordinator,
		gasConfig:                      gasConfig,
		mutExecution:                   sync.RWMutex{},
		payableHandler:                 &disabledPayableHandler{},
		rolesHandler:                   roleHandler,
		transferToMetaEnableEpoch:      transferToMetaEnableEpoch,
		checkCorrectTokenIDEnableEpoch: checkCorrectTokenIDEnableEpoch,
		mectStorageHandler:             mectStorageHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionMultiMECTNFTTransfer,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *mectNFTMultiTransfer) EpochConfirmed(epoch uint32, nonce uint64) {
	e.baseEnabled.EpochConfirmed(epoch, nonce)
	e.flagTransferToMeta.SetValue(epoch >= e.transferToMetaEnableEpoch)
	log.Debug("MECT NFT transfer to metachain flag", "enabled", e.flagTransferToMeta.IsSet())
	e.flagCheckCorrectTokenID.SetValue(epoch >= e.checkCorrectTokenIDEnableEpoch)
	log.Debug("MECT multi transfer check correct tokenID for transfer role", "enabled", e.flagCheckCorrectTokenID.IsSet())
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *mectNFTMultiTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTMultiTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTMultiTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT transfer roles function call
// Requires the following arguments:
// arg0 - destination address
// arg1 - number of tokens to transfer
// list of (tokenID - nonce - quantity) - in case of MECT nonce == 0
// function and list of arguments for SC Call
// if cross-shard, the rest of arguments will be filled inside the SCR
// arg0 - number of tokens to transfer
// list of (tokenID - nonce - quantity/MECT NFT data)
// function and list of arguments for SC Call
func (e *mectNFTMultiTransfer) ProcessBuiltinFunction(
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
		return e.processMECTNFTMultiTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	vmOutput.Logs = make([]*vmcommon.LogEntry, 0, numOfTransfers)
	startIndex := uint64(1)

	err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, int(minNumOfArguments))
	if err != nil {
		return nil, err
	}

	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := vmInput.Arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64()

		mectTokenKey := append(e.keyPrefix, tokenID...)

		value := big.NewInt(0)
		if nonce > 0 {
			mectTransferData := &mect.MECToken{}
			if len(vmInput.Arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
				marshaledNFTTransfer := vmInput.Arguments[tokenStartIndex+2]
				err = e.marshaller.Unmarshal(mectTransferData, marshaledNFTTransfer)
				if err != nil {
					return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
				}
			} else {
				mectTransferData.Value = big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
				mectTransferData.Type = uint32(core.NonFungible)
			}

			value.Set(mectTransferData.Value)
			err = e.addNFTToDestination(
				vmInput.CallerAddr,
				vmInput.RecipientAddr,
				acntDst,
				mectTransferData,
				mectTokenKey,
				nonce,
				vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
			}
		} else {
			transferredValue := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
			value.Set(transferredValue)
			err = addToMECTBalance(acntDst, mectTokenKey, transferredValue, e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, fmt.Errorf("%w for token %s", err, string(tokenID))
			}
		}

		addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMultiMECTNFTTransfer), tokenID, nonce, value, vmInput.CallerAddr, acntDst.AddressBytes())
	}

	// no need to consume gas on destination - sender already paid for it
	if len(vmInput.Arguments) > int(minNumOfArguments) && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > int(minNumOfArguments)+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return vmOutput, nil
}

func (e *mectNFTMultiTransfer) processMECTNFTMultiTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[0]
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
	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	multiTransferCost := numOfTransfers * e.funcGasCost
	if vmInput.GasProvided < multiTransferCost {
		return nil, ErrNotEnoughGas
	}

	acntDst, err := e.loadAccountIfInShard(dstAddress)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntDst) {
		err = e.payableHandler.CheckPayable(vmInput, dstAddress, int(minNumOfArguments))
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - multiTransferCost,
		Logs:         make([]*vmcommon.LogEntry, 0, numOfTransfers),
	}

	startIndex := uint64(2)
	listMectData := make([]*mect.MECToken, numOfTransfers)
	listTransferData := make([]*vmcommon.MECTTransfer, numOfTransfers)

	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		listTransferData[i] = &vmcommon.MECTTransfer{
			MECTValue:      big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2]),
			MECTTokenName:  vmInput.Arguments[tokenStartIndex],
			MECTTokenType:  0,
			MECTTokenNonce: big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64(),
		}
		if listTransferData[i].MECTTokenNonce > 0 {
			listTransferData[i].MECTTokenType = uint32(core.NonFungible)
		}

		listMectData[i], err = e.transferOneTokenOnSenderShard(
			acntSnd,
			acntDst,
			dstAddress,
			listTransferData[i],
			vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, fmt.Errorf("%w for token %s", err, string(listTransferData[i].MECTTokenName))
		}

		addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMultiMECTNFTTransfer), listTransferData[i].MECTTokenName, listTransferData[i].MECTTokenNonce, listTransferData[i].MECTValue, vmInput.CallerAddr, dstAddress)
	}

	if !check.IfNil(acntDst) {
		err = e.accounts.SaveAccount(acntDst)
		if err != nil {
			return nil, err
		}
	}

	err = e.createMECTNFTOutputTransfers(vmInput, vmOutput, listMectData, listTransferData, dstAddress)
	if err != nil {
		return nil, err
	}

	return vmOutput, nil
}

func (e *mectNFTMultiTransfer) transferOneTokenOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	acntDst vmcommon.UserAccountHandler,
	dstAddress []byte,
	transferData *vmcommon.MECTTransfer,
	isReturnCallWithError bool,
) (*mect.MECToken, error) {
	if transferData.MECTValue.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}

	mectTokenKey := append(e.keyPrefix, transferData.MECTTokenName...)
	mectData, err := e.mectStorageHandler.GetMECTNFTTokenOnSender(acntSnd, mectTokenKey, transferData.MECTTokenNonce)
	if err != nil {
		return nil, err
	}

	if mectData.Value.Cmp(transferData.MECTValue) < 0 {
		return nil, computeInsufficientQuantityMECTError(transferData.MECTTokenName, transferData.MECTTokenNonce)
	}
	mectData.Value.Sub(mectData.Value, transferData.MECTValue)

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, transferData.MECTTokenNonce, mectData, false, isReturnCallWithError)
	if err != nil {
		return nil, err
	}

	mectData.Value.Set(transferData.MECTValue)

	tokenID := mectTokenKey
	if e.flagCheckCorrectTokenID.IsSet() {
		tokenID = transferData.MECTTokenName
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(tokenID, mectTokenKey, acntSnd.AddressBytes(), dstAddress, e.globalSettingsHandler, e.rolesHandler, acntSnd, acntDst, isReturnCallWithError)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntDst) {
		err = e.addNFTToDestination(acntSnd.AddressBytes(), dstAddress, acntDst, mectData, mectTokenKey, transferData.MECTTokenNonce, isReturnCallWithError)
		if err != nil {
			return nil, err
		}
	} else {
		err = e.mectStorageHandler.AddToLiquiditySystemAcc(mectTokenKey, transferData.MECTTokenNonce, big.NewInt(0).Neg(transferData.MECTValue))
		if err != nil {
			return nil, err
		}
	}

	return mectData, nil
}

func computeInsufficientQuantityMECTError(tokenID []byte, nonce uint64) error {
	err := fmt.Errorf("%w for token: %s", ErrInsufficientQuantityMECT, string(tokenID))
	if nonce > 0 {
		err = fmt.Errorf("%w nonce %d", err, nonce)
	}

	return err
}

func (e *mectNFTMultiTransfer) loadAccountIfInShard(dstAddress []byte) (vmcommon.UserAccountHandler, error) {
	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		return nil, nil
	}

	accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
	if errLoad != nil {
		return nil, errLoad
	}
	userAccount, ok := accountHandler.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAccount, nil
}

func (e *mectNFTMultiTransfer) createMECTNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	listMECTData []*mect.MECToken,
	listMECTTransfers []*vmcommon.MECTTransfer,
	dstAddress []byte,
) error {
	multiTransferCallArgs := make([][]byte, 0, argumentsPerTransfer*uint64(len(listMECTTransfers))+1)
	numTokenTransfer := big.NewInt(int64(len(listMECTTransfers))).Bytes()
	multiTransferCallArgs = append(multiTransferCallArgs, numTokenTransfer)

	for i, mectTransfer := range listMECTTransfers {
		multiTransferCallArgs = append(multiTransferCallArgs, mectTransfer.MECTTokenName)
		nonceAsBytes := []byte{0}
		if mectTransfer.MECTTokenNonce > 0 {
			nonceAsBytes = big.NewInt(0).SetUint64(mectTransfer.MECTTokenNonce).Bytes()
		}
		multiTransferCallArgs = append(multiTransferCallArgs, nonceAsBytes)

		if mectTransfer.MECTTokenNonce > 0 {
			wasAlreadySent, err := e.mectStorageHandler.WasAlreadySentToDestinationShardAndUpdateState(mectTransfer.MECTTokenName, mectTransfer.MECTTokenNonce, dstAddress)
			if err != nil {
				return err
			}

			sendCrossShardAsMarshalledData := !wasAlreadySent || mectTransfer.MECTValue.Cmp(oneValue) == 0 ||
				len(mectTransfer.MECTValue.Bytes()) > vmcommon.MaxLengthForValueToOptTransfer
			if sendCrossShardAsMarshalledData {
				marshaledNFTTransfer, err := e.marshaller.Marshal(listMECTData[i])
				if err != nil {
					return err
				}

				gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
				if gasForTransfer > vmOutput.GasRemaining {
					return ErrNotEnoughGas
				}
				vmOutput.GasRemaining -= gasForTransfer

				multiTransferCallArgs = append(multiTransferCallArgs, marshaledNFTTransfer)
			} else {
				multiTransferCallArgs = append(multiTransferCallArgs, mectTransfer.MECTValue.Bytes())
			}

		} else {
			multiTransferCallArgs = append(multiTransferCallArgs, mectTransfer.MECTValue.Bytes())
		}
	}

	minNumOfArguments := uint64(len(listMECTTransfers))*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) > minNumOfArguments {
		multiTransferCallArgs = append(multiTransferCallArgs, vmInput.Arguments[minNumOfArguments:]...)
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, dstAddress, int(minNumOfArguments))

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			vmInput.CallerAddr,
			dstAddress,
			core.BuiltInFunctionMultiMECTNFTTransfer,
			multiTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if uint64(len(vmInput.Arguments)) > minNumOfArguments+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *mectNFTMultiTransfer) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	mectDataToTransfer *mect.MECToken,
	mectTokenKey []byte,
	nonce uint64,
	isReturnCallWithError bool,
) error {
	currentMECTData, _, err := e.mectStorageHandler.GetMECTNFTTokenOnDestination(userAccount, mectTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, mectTokenKey, currentMECTData, e.globalSettingsHandler, isReturnCallWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(mectDataToTransfer.Value)
	mectDataToTransfer.Value.Add(mectDataToTransfer.Value, currentMECTData.Value)
	_, err = e.mectStorageHandler.SaveMECTNFTToken(sndAddress, userAccount, mectTokenKey, nonce, mectDataToTransfer, false, isReturnCallWithError)
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

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTMultiTransfer) IsInterfaceNil() bool {
	return e == nil
}
