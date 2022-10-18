package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

const maxLenForAddNFTQuantity = 32

type mectNFTAddQuantity struct {
	baseAlwaysActive
	keyPrefix             []byte
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler
	rolesHandler          vmcommon.MECTRoleHandler
	mectStorageHandler    vmcommon.MECTNFTStorageHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex

	valueLengthCheckEnableEpoch uint32
	flagValueLengthCheck        atomic.Flag
}

// NewMECTNFTAddQuantityFunc returns the mect NFT add quantity built-in function component
func NewMECTNFTAddQuantityFunc(
	funcGasCost uint64,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	rolesHandler vmcommon.MECTRoleHandler,
	valueLengthCheckEnableEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*mectNFTAddQuantity, error) {
	if check.IfNil(mectStorageHandler) {
		return nil, ErrNilMECTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &mectNFTAddQuantity{
		keyPrefix:                   []byte(baseMECTKeyPrefix),
		globalSettingsHandler:       globalSettingsHandler,
		rolesHandler:                rolesHandler,
		funcGasCost:                 funcGasCost,
		mutExecution:                sync.RWMutex{},
		mectStorageHandler:          mectStorageHandler,
		valueLengthCheckEnableEpoch: valueLengthCheckEnableEpoch,
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// EpochConfirmed is called whenever a new epoch is confirmed
func (e *mectNFTAddQuantity) EpochConfirmed(epoch uint32, _ uint64) {
	e.flagValueLengthCheck.SetValue(epoch >= e.valueLengthCheckEnableEpoch)
	log.Debug("MECT Add Quantity value length check", "enabled", e.flagValueLengthCheck.IsSet())
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTAddQuantity) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTAddQuantity
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to add
func (e *mectNFTAddQuantity) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkMECTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.MECTRoleNFTAddQuantity))
	if err != nil {
		return nil, err
	}

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	mectData, err := e.mectStorageHandler.GetMECTNFTTokenOnSender(acntSnd, mectTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	if e.flagValueLengthCheck.IsSet() && len(vmInput.Arguments[2]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for add nft quantity is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	mectData.Value.Add(mectData.Value, value)

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, nonce, mectData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}
	err = e.mectStorageHandler.AddToLiquiditySystemAcc(mectTokenKey, nonce, value)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTAddQuantity), vmInput.Arguments[0], nonce, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTAddQuantity) IsInterfaceNil() bool {
	return e == nil
}
