package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectNFTAddUri struct {
	*baseEnabled
	keyPrefix             []byte
	mectStorageHandler    vmcommon.MECTNFTStorageHandler
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler
	rolesHandler          vmcommon.MECTRoleHandler
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewMECTNFTAddUriFunc returns the mect NFT add URI built-in function component
func NewMECTNFTAddUriFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	mectStorageHandler vmcommon.MECTNFTStorageHandler,
	globalSettingsHandler vmcommon.MECTGlobalSettingsHandler,
	rolesHandler vmcommon.MECTRoleHandler,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*mectNFTAddUri, error) {
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

	e := &mectNFTAddUri{
		keyPrefix:             []byte(baseMECTKeyPrefix),
		mectStorageHandler:    mectStorageHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		globalSettingsHandler: globalSettingsHandler,
		gasConfig:             gasConfig,
		rolesHandler:          rolesHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionMECTNFTAddURI,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTAddUri) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.MECTNFTAddURI
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves MECT NFT add uris function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg[2:] - uris to add
func (e *mectNFTAddUri) ProcessBuiltinFunction(
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

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.MECTRoleNFTAddURI))
	if err != nil {
		return nil, err
	}

	gasCostForStore := e.getGasCostForURIStore(vmInput)
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

	mectData.TokenMetaData.URIs = append(mectData.TokenMetaData.URIs, vmInput.Arguments[2:]...)

	_, err = e.mectStorageHandler.SaveMECTNFTToken(acntSnd.AddressBytes(), acntSnd, mectTokenKey, nonce, mectData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
	}

	extraTopics := append([][]byte{vmInput.CallerAddr}, vmInput.Arguments[2:]...)
	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTAddURI), vmInput.Arguments[0], nonce, big.NewInt(0), extraTopics...)

	return vmOutput, nil
}

func (e *mectNFTAddUri) getGasCostForURIStore(vmInput *vmcommon.ContractCallInput) uint64 {
	lenURIs := 0
	for _, uri := range vmInput.Arguments[2:] {
		lenURIs += len(uri)
	}
	return uint64(lenURIs) * e.gasConfig.StorePerByte
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTAddUri) IsInterfaceNil() bool {
	return e == nil
}
