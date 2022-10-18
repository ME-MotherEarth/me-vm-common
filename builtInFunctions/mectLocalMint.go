package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectLocalMint struct {
	baseAlwaysActive
	keyPrefix             []byte
	marshaller            vmcommon.Marshalizer
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler
	rolesHandler          vmcommon.MECTRoleHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewMECTLocalMintFunc returns the mect local mint built-in function component
func NewMECTLocalMintFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	rolesHandler vmcommon.MECTRoleHandler,
) (*mectLocalMint, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &mectLocalMint{
		keyPrefix:             []byte(baseMECTKeyPrefix),
		marshaller:            marshaller,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectLocalMint) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTLocalMint
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT local mint function call
func (e *mectLocalMint) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.MECTRoleLocalMint))
	if err != nil {
		return nil, err
	}

	if len(vmInput.Arguments[1]) > core.MaxLenForMECTIssueMint {
		return nil, fmt.Errorf("%w max length for mect issue is %d", ErrInvalidArguments, core.MaxLenForMECTIssueMint)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	mectTokenKey := append(e.keyPrefix, tokenID...)
	err = addToMECTBalance(acntSnd, mectTokenKey, big.NewInt(0).Set(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost}

	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTLocalMint), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectLocalMint) IsInterfaceNil() bool {
	return e == nil
}
