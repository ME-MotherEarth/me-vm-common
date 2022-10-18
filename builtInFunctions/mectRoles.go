package builtInFunctions

import (
	"bytes"
	"math"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

var roleKeyPrefix = []byte(core.MotherEarthProtectedKeyPrefix + core.MECTRoleIdentifier + core.MECTKeyIdentifier)

type mectRoles struct {
	baseAlwaysActive
	set        bool
	marshaller vmcommon.Marshalizer
}

// NewMECTRolesFunc returns the mect change roles built-in function component
func NewMECTRolesFunc(
	marshaller vmcommon.Marshalizer,
	set bool,
) (*mectRoles, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	e := &mectRoles{
		set:        set,
		marshaller: marshaller,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectRoles) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT change roles function call
func (e *mectRoles) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicMECTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.MECTSCAddress) {
		return nil, ErrAddressIsNotMECTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	mectTokenRoleKey := append(roleKeyPrefix, vmInput.Arguments[0]...)

	roles, _, err := getMECTRolesForAcnt(e.marshaller, acntDst, mectTokenRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		roles.Roles = append(roles.Roles, vmInput.Arguments[1:]...)
	} else {
		deleteRoles(roles, vmInput.Arguments[1:])
	}

	for _, arg := range vmInput.Arguments[1:] {
		if !bytes.Equal(arg, []byte(core.MECTRoleNFTCreateMultiShard)) {
			continue
		}

		err = saveLatestNonce(acntDst, vmInput.Arguments[0], computeStartNonce(vmInput.RecipientAddr))
		if err != nil {
			return nil, err
		}

		break
	}

	err = saveRolesToAccount(acntDst, mectTokenRoleKey, roles, e.marshaller)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{acntDst.AddressBytes()}, vmInput.Arguments[1:]...)
	addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

// Nonces on multi shard NFT create are from (LastByte * MaxUint64 / 256), this is in order to differentiate them
// even like this, if one contract makes 1000 NFT create on each block, it would need 14 million years to occupy the whole space
// 2 ^ 64 / 256 / 1000 / 14400 / 365 ~= 14 million
func computeStartNonce(destAddress []byte) uint64 {
	lastByteOfAddress := uint64(destAddress[len(destAddress)-1])
	startNonce := (math.MaxUint64 / 256) * lastByteOfAddress
	return startNonce
}

func deleteRoles(roles *mect.MECTRoles, deleteRoles [][]byte) {
	for _, deleteRole := range deleteRoles {
		index, exist := doesRoleExist(roles, deleteRole)
		if !exist {
			continue
		}

		copy(roles.Roles[index:], roles.Roles[index+1:])
		roles.Roles[len(roles.Roles)-1] = nil
		roles.Roles = roles.Roles[:len(roles.Roles)-1]
	}
}

func doesRoleExist(roles *mect.MECTRoles, role []byte) (int, bool) {
	for i, currentRole := range roles.Roles {
		if bytes.Equal(currentRole, role) {
			return i, true
		}
	}
	return -1, false
}

func getMECTRolesForAcnt(
	marshaller vmcommon.Marshalizer,
	acnt vmcommon.UserAccountHandler,
	key []byte,
) (*mect.MECTRoles, bool, error) {
	roles := &mect.MECTRoles{
		Roles: make([][]byte, 0),
	}

	marshaledData, err := acnt.AccountDataHandler().RetrieveValue(key)
	if err != nil || len(marshaledData) == 0 {
		return roles, true, nil
	}

	err = marshaller.Unmarshal(roles, marshaledData)
	if err != nil {
		return nil, false, err
	}

	return roles, false, nil
}

// CheckAllowedToExecute returns error if the account is not allowed to execute the given action
func (e *mectRoles) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if check.IfNil(account) {
		return ErrNilUserAccount
	}

	mectTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, isNew, err := getMECTRolesForAcnt(e.marshaller, account, mectTokenRoleKey)
	if err != nil {
		return err
	}
	if isNew {
		return ErrActionNotAllowed
	}
	_, exist := doesRoleExist(roles, action)
	if !exist {
		return ErrActionNotAllowed
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectRoles) IsInterfaceNil() bool {
	return e == nil
}
