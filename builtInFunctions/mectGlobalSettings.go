package builtInFunctions

import (
	"bytes"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/marshal"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectGlobalSettings struct {
	*baseEnabled
	keyPrefix  []byte
	set        bool
	accounts   vmcommon.AccountsAdapter
	marshaller marshal.Marshalizer
}

// NewMECTGlobalSettingsFunc returns the mect pause/un-pause built-in function component
func NewMECTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	set bool,
	function string,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*mectGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if !isCorrectFunction(function) {
		return nil, ErrInvalidArguments
	}

	e := &mectGlobalSettings{
		keyPrefix:  []byte(baseMECTKeyPrefix),
		set:        set,
		accounts:   accounts,
		marshaller: marshaller,
	}

	e.baseEnabled = &baseEnabled{
		function:        function,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

func isCorrectFunction(function string) bool {
	switch function {
	case core.BuiltInFunctionMECTPause, core.BuiltInFunctionMECTUnPause, core.BuiltInFunctionMECTSetLimitedTransfer, core.BuiltInFunctionMECTUnSetLimitedTransfer:
		return true
	case vmcommon.BuiltInFunctionMECTSetBurnRoleForAll, vmcommon.BuiltInFunctionMECTUnSetBurnRoleForAll:
		return true
	default:
		return false
	}
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectGlobalSettings) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT pause function call
func (e *mectGlobalSettings) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.MECTSCAddress) {
		return nil, ErrAddressIsNotMECTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	err := e.toggleSetting(mectTokenKey)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *mectGlobalSettings) toggleSetting(mectTokenKey []byte) error {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	mectMetaData, err := e.getGlobalMetadata(mectTokenKey)
	if err != nil {
		return err
	}

	switch e.function {
	case core.BuiltInFunctionMECTSetLimitedTransfer, core.BuiltInFunctionMECTUnSetLimitedTransfer:
		mectMetaData.LimitedTransfer = e.set
		break
	case core.BuiltInFunctionMECTPause, core.BuiltInFunctionMECTUnPause:
		mectMetaData.Paused = e.set
		break
	case vmcommon.BuiltInFunctionMECTUnSetBurnRoleForAll, vmcommon.BuiltInFunctionMECTSetBurnRoleForAll:
		mectMetaData.BurnRoleForAll = e.set
		break
	}

	err = systemSCAccount.AccountDataHandler().SaveKeyValue(mectTokenKey, mectMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemSCAccount)
}

func (e *mectGlobalSettings) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// IsPaused returns true if the mectTokenKey (prefixed) is paused
func (e *mectGlobalSettings) IsPaused(mectTokenKey []byte) bool {
	mectMetadata, err := e.getGlobalMetadata(mectTokenKey)
	if err != nil {
		return false
	}

	return mectMetadata.Paused
}

// IsLimitedTransfer returns true if the mectTokenKey (prefixed) is with limited transfer
func (e *mectGlobalSettings) IsLimitedTransfer(mectTokenKey []byte) bool {
	mectMetadata, err := e.getGlobalMetadata(mectTokenKey)
	if err != nil {
		return false
	}

	return mectMetadata.LimitedTransfer
}

// IsBurnForAll returns true if the mectTokenKey (prefixed) is with burn for all
func (e *mectGlobalSettings) IsBurnForAll(mectTokenKey []byte) bool {
	mectMetadata, err := e.getGlobalMetadata(mectTokenKey)
	if err != nil {
		return false
	}

	return mectMetadata.BurnRoleForAll
}

// IsSenderOrDestinationWithTransferRole returns true if we have transfer role on the system account
func (e *mectGlobalSettings) IsSenderOrDestinationWithTransferRole(sender, destination, tokenID []byte) bool {
	if !e.baseEnabled.IsActive() {
		return false
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return false
	}

	mectTokenTransferRoleKey := append(transferAddressesKeyPrefix, tokenID...)
	addresses, _, err := getMECTRolesForAcnt(e.marshaller, systemAcc, mectTokenTransferRoleKey)
	if err != nil {
		return false
	}

	for _, address := range addresses.Roles {
		if bytes.Equal(address, sender) || bytes.Equal(address, destination) {
			return true
		}
	}

	return false
}

func (e *mectGlobalSettings) getGlobalMetadata(mectTokenKey []byte) (*MECTGlobalMetadata, error) {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _ := systemSCAccount.AccountDataHandler().RetrieveValue(mectTokenKey)
	mectMetaData := MECTGlobalMetadataFromBytes(val)
	return &mectMetaData, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
