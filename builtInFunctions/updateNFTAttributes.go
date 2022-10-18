package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectNFTupdate struct {
	*baseEnabled
	keyPrefix             []byte
	mectStorageHandler    vmcommon.MECTNFTStorageHandler
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler
	rolesHandler          vmcommon.MECTRoleHandler
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewMECTNFTUpdateAttributesFunc returns the mect NFT update attribute built-in function component
func NewMECTNFTUpdateAttributesFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	rolesHandler vmcommon.MECTRoleHandler,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*mectNFTupdate, error) {
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

	e := &mectNFTupdate{
		keyPrefix:             []byte(baseMECTKeyPrefix),
		mectStorageHandler:    mectStorageHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		globalSettingsHandler: globalSettingsHandler,
		gasConfig:             gasConfig,
		rolesHandler:          rolesHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionMECTNFTUpdateAttributes,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTupdate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTUpdateAttributes
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT update attributes function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - new attributes
func (e *mectNFTupdate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkMECTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.MECTRoleNFTUpdateAttributes))
	if err != nil {
		return nil, err
	}

	gasCostForStore := uint64(len(vmInput.Arguments[2])) * e.gasConfig.StorePerByte
	if vmInput.GasProvided < e.funcGasCost+gasCostForStore {
		return nil, ErrNotEnoughGas
	}

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	mectData, err := e.mectStorageHandler.GetMECTNFTTokenOnSender(acntSnd, mectTokenKey, nonce)
	if err != nil {
		return nil, err
	}

	mectData.TokenMetaData.Attributes = vmInput.Arguments[2]

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, nonce, mectData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTUpdateAttributes), vmInput.Arguments[0], nonce, big.NewInt(0), vmInput.CallerAddr, vmInput.Arguments[2])

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTupdate) IsInterfaceNil() bool {
	return e == nil
}
