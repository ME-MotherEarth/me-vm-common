package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-core/marshal"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

const transfer = "transfer"

var transferAddressesKeyPrefix = []byte(core.MotherEarthProtectedKeyPrefix + transfer + core.MECTKeyIdentifier)

type mectTransferAddress struct {
	*baseEnabled
	set             bool
	marshaller      vmcommon.Marshalizer
	accounts        vmcommon.AccountsAdapter
	maxNumAddresses uint32
}

// NewMECTTransferRoleAddressFunc returns the mect transfer role address handler built-in function component
func NewMECTTransferRoleAddressFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
	maxNumAddresses uint32,
	set bool,
) (*mectTransferAddress, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if maxNumAddresses < 1 {
		return nil, ErrInvalidMaxNumAddresses
	}

	e := &mectTransferAddress{
		accounts:        accounts,
		marshaller:      marshaller,
		maxNumAddresses: maxNumAddresses,
		set:             set,
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionMECTTransferRoleAddAddress,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}
	if !set {
		e.function = vmcommon.BuiltInFunctionMECTTransferRoleDeleteAddress
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectTransferAddress) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT change roles function call
func (e *mectTransferAddress) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicMECTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.MECTSCAddress) {
		return nil, ErrAddressIsNotMECTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	mectTokenTransferRoleKey := append(transferAddressesKeyPrefix, vmInput.Arguments[0]...)
	addresses, _, err := getMECTRolesForAcnt(e.marshaller, systemAcc, mectTokenTransferRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		err = e.addNewAddresses(vmInput, addresses)
		if err != nil {
			return nil, err
		}
	} else {
		deleteRoles(addresses, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(systemAcc, mectTokenTransferRoleKey, addresses, e.marshaller)
	if err != nil {
		return nil, err
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{systemAcc.AddressBytes()}, vmInput.Arguments[1:]...)
	addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

func (e *mectTransferAddress) addNewAddresses(vmInput *vmcommon.ContractCallInput, addresses *mect.MECTRoles) error {
	for _, newAddress := range vmInput.Arguments[1:] {
		isNew := true
		for _, address := range addresses.Roles {
			if bytes.Equal(newAddress, address) {
				isNew = false
				break
			}
		}
		if isNew {
			addresses.Roles = append(addresses.Roles, newAddress)
		}
	}

	if uint32(len(addresses.Roles)) > e.maxNumAddresses {
		return ErrTooManyTransferAddresses
	}

	return nil
}

func (e *mectTransferAddress) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// IsInterfaceNil returns true if underlying object in nil
func (e *mectTransferAddress) IsInterfaceNil() bool {
	return e == nil
}
