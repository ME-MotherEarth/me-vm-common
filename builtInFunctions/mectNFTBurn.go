package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectNFTBurn struct {
	baseAlwaysActive
	keyPrefix             []byte
	mectStorageHandler    vmcommon.MECTNFTStorageHandler
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler
	rolesHandler          vmcommon.MECTRoleHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewMECTNFTBurnFunc returns the mect NFT burn built-in function component
func NewMECTNFTBurnFunc(
	funcGasCost uint64,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
	globalSettingsHandler vmcommon.ExtendedMECTGlobalSettingsHandler,
	rolesHandler vmcommon.MECTRoleHandler,
) (*mectNFTBurn, error) {
	if check.IfNil(mectStorageHandler) {
		return nil, ErrNilMECTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &mectNFTBurn{
		keyPrefix:             []byte(baseMECTKeyPrefix),
		mectStorageHandler:    mectStorageHandler,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT burn function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to burn
func (e *mectNFTBurn) ProcessBuiltinFunction(
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

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	err = e.isAllowedToBurn(acntSnd, vmInput.Arguments[0])
	if err != nil {
		return nil, err
	}

	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	mectData, err := e.mectStorageHandler.GetMECTNFTTokenOnSender(acntSnd, mectTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	quantityToBurn := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if mectData.Value.Cmp(quantityToBurn) < 0 {
		return nil, ErrInvalidNFTQuantity
	}

	mectData.Value.Sub(mectData.Value, quantityToBurn)

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, nonce, mectData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	err = e.mectStorageHandler.AddToLiquiditySystemAcc(mectTokenKey, nonce, big.NewInt(0).Neg(quantityToBurn))
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTBurn), vmInput.Arguments[0], nonce, quantityToBurn, vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *mectNFTBurn) isAllowedToBurn(acntSnd vmcommon.UserAccountHandler, tokenID []byte) error {
	mectTokenKey := append(e.keyPrefix, tokenID...)
	isBurnForAll := e.globalSettingsHandler.IsBurnForAll(mectTokenKey)
	if isBurnForAll {
		return nil
	}

	return e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.MECTRoleNFTBurn))
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTBurn) IsInterfaceNil() bool {
	return e == nil
}
