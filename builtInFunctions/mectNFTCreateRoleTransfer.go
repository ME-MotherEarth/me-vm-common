package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectNFTCreateRoleTransfer struct {
	baseAlwaysActive
	keyPrefix        []byte
	marshaller       vmcommon.Marshalizer
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
}

// NewMECTNFTCreateRoleTransfer returns the mect NFT create role transfer built-in function component
func NewMECTNFTCreateRoleTransfer(
	marshaller vmcommon.Marshalizer,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
) (*mectNFTCreateRoleTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &mectNFTCreateRoleTransfer{
		keyPrefix:        []byte(baseMECTKeyPrefix),
		marshaller:       marshaller,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectNFTCreateRoleTransfer) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT create role transfer function call
func (e *mectNFTCreateRoleTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {

	err := checkBasicMECTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidArguments
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	if bytes.Equal(vmInput.CallerAddr, core.MECTSCAddress) {
		outAcc, errExec := e.executeTransferNFTCreateChangeAtCurrentOwner(vmOutput, acntDst, vmInput)
		if errExec != nil {
			return nil, errExec
		}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		vmOutput.OutputAccounts[string(outAcc.Address)] = outAcc
	} else {
		err = e.executeTransferNFTCreateChangeAtNextOwner(vmOutput, acntDst, vmInput)
		if err != nil {
			return nil, err
		}
	}

	return vmOutput, nil
}

func (e *mectNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtCurrentOwner(
	vmOutput *vmcommon.VMOutput,
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.OutputAccount, error) {
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	if len(vmInput.Arguments[1]) != len(vmInput.CallerAddr) {
		return nil, ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce, err := getLatestNonce(acntDst, tokenID)
	if err != nil {
		return nil, err
	}

	err = saveLatestNonce(acntDst, tokenID, 0)
	if err != nil {
		return nil, err
	}

	mectTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.deleteCreateRoleFromAccount(acntDst, mectTokenRoleKey)
	if err != nil {
		return nil, err
	}

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(false)}
	addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

	destAddress := vmInput.Arguments[1]
	if e.shardCoordinator.ComputeId(destAddress) == e.shardCoordinator.SelfId() {
		newDestinationAcc, errLoad := e.accounts.LoadAccount(destAddress)
		if errLoad != nil {
			return nil, errLoad
		}
		newDestUserAcc, ok := newDestinationAcc.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = saveLatestNonce(newDestUserAcc, tokenID, nonce)
		if err != nil {
			return nil, err
		}

		err = e.addCreateRoleToAccount(newDestUserAcc, mectTokenRoleKey)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(newDestUserAcc)
		if err != nil {
			return nil, err
		}

		logData = [][]byte{destAddress, boolToSlice(true)}
		addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)
	}

	outAcc := &vmcommon.OutputAccount{
		Address:         destAddress,
		Balance:         big.NewInt(0),
		BalanceDelta:    big.NewInt(0),
		OutputTransfers: make([]vmcommon.OutputTransfer, 0),
	}
	outTransfer := vmcommon.OutputTransfer{
		Value: big.NewInt(0),
		Data: []byte(core.BuiltInFunctionMECTNFTCreateRoleTransfer + "@" +
			hex.EncodeToString(tokenID) + "@" + hex.EncodeToString(big.NewInt(0).SetUint64(nonce).Bytes())),
		SenderAddress: vmInput.CallerAddr,
	}
	outAcc.OutputTransfers = append(outAcc.OutputTransfers, outTransfer)

	return outAcc, nil
}

func (e *mectNFTCreateRoleTransfer) deleteCreateRoleFromAccount(
	acntDst vmcommon.UserAccountHandler,
	mectTokenRoleKey []byte,
) error {
	roles, _, err := getMECTRolesForAcnt(e.marshaller, acntDst, mectTokenRoleKey)
	if err != nil {
		return err
	}

	deleteRoles(roles, [][]byte{[]byte(core.MECTRoleNFTCreate)})
	return saveRolesToAccount(acntDst, mectTokenRoleKey, roles, e.marshaller)
}

func (e *mectNFTCreateRoleTransfer) addCreateRoleToAccount(
	acntDst vmcommon.UserAccountHandler,
	mectTokenRoleKey []byte,
) error {
	roles, _, err := getMECTRolesForAcnt(e.marshaller, acntDst, mectTokenRoleKey)
	if err != nil {
		return err
	}

	for _, role := range roles.Roles {
		if bytes.Equal(role, []byte(core.MECTRoleNFTCreate)) {
			return nil
		}
	}

	roles.Roles = append(roles.Roles, []byte(core.MECTRoleNFTCreate))
	return saveRolesToAccount(acntDst, mectTokenRoleKey, roles, e.marshaller)
}

func saveRolesToAccount(
	acntDst vmcommon.UserAccountHandler,
	mectTokenRoleKey []byte,
	roles *mect.MECTRoles,
	marshaller vmcommon.Marshalizer,
) error {
	marshaledData, err := marshaller.Marshal(roles)
	if err != nil {
		return err
	}
	err = acntDst.AccountDataHandler().SaveKeyValue(mectTokenRoleKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

func (e *mectNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtNextOwner(
	vmOutput *vmcommon.VMOutput,
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	if len(vmInput.Arguments) != 2 {
		return ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()

	err := saveLatestNonce(acntDst, tokenID, nonce)
	if err != nil {
		return err
	}

	mectTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.addCreateRoleToAccount(acntDst, mectTokenRoleKey)
	if err != nil {
		return err
	}

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(true)}
	addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectNFTCreateRoleTransfer) IsInterfaceNil() bool {
	return e == nil
}
