package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/atomic"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

const numArgsPerAdd = 3

type mectDeleteMetaData struct {
	*baseEnabled
	allowedAddress []byte
	delete         bool
	accounts       vmcommon.AccountsAdapter
	keyPrefix      []byte
	marshaller     vmcommon.Marshalizer
	funcGasCost    uint64
}

// ArgsNewMECTDeleteMetadata defines the argument list for new mect delete metadata built in function
type ArgsNewMECTDeleteMetadata struct {
	FuncGasCost     uint64
	Marshalizer     vmcommon.Marshalizer
	Accounts        vmcommon.AccountsAdapter
	ActivationEpoch uint32
	EpochNotifier   vmcommon.EpochNotifier
	AllowedAddress  []byte
	Delete          bool
}

// NewMECTDeleteMetadataFunc returns the mect metadata deletion built-in function component
func NewMECTDeleteMetadataFunc(
	args ArgsNewMECTDeleteMetadata,
) (*mectDeleteMetaData, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &mectDeleteMetaData{
		keyPrefix:      []byte(baseMECTKeyPrefix),
		marshaller:     args.Marshalizer,
		funcGasCost:    args.FuncGasCost,
		accounts:       args.Accounts,
		allowedAddress: args.AllowedAddress,
		delete:         args.Delete,
	}

	e.baseEnabled = &baseEnabled{
		function:        core.BuiltInFunctionMultiMECTNFTTransfer,
		activationEpoch: args.ActivationEpoch,
		flagActivated:   atomic.Flag{},
	}

	args.EpochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectDeleteMetaData) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT delete and add metadata function call
func (e *mectDeleteMetaData) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if !bytes.Equal(vmInput.CallerAddr, e.allowedAddress) {
		return nil, ErrAddressIsNotAllowed
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return nil, ErrInvalidRcvAddr
	}

	if e.delete {
		err := e.deleteMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.addMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	return vmOutput, nil
}

// input is list(tokenID-numIntervals-list(start,end))
func (e *mectDeleteMetaData) deleteMetadata(args [][]byte) error {
	lenArgs := uint64(len(args))
	if lenArgs < 4 {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := uint64(0); i+1 < uint64(len(args)); {
		tokenID := args[i]
		numIntervals := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		i += 2

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		if i >= lenArgs {
			return ErrInvalidNumOfArgs
		}

		err = e.deleteMetadataForListIntervals(systemAcc, tokenID, args, i, numIntervals)
		if err != nil {
			return err
		}

		i += numIntervals * 2
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *mectDeleteMetaData) deleteMetadataForListIntervals(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	args [][]byte,
	index, numIntervals uint64,
) error {
	lenArgs := uint64(len(args))
	for j := index; j < index+numIntervals*2; j += 2 {
		if j > lenArgs-2 {
			return ErrInvalidNumOfArgs
		}

		startIndex := big.NewInt(0).SetBytes(args[j]).Uint64()
		endIndex := big.NewInt(0).SetBytes(args[j+1]).Uint64()

		err := e.deleteMetadataForInterval(systemAcc, tokenID, startIndex, endIndex)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *mectDeleteMetaData) deleteMetadataForInterval(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	startIndex, endIndex uint64,
) error {
	if endIndex < startIndex {
		return ErrInvalidArguments
	}
	if startIndex == 0 {
		return ErrInvalidNonce
	}

	mectTokenKey := append(e.keyPrefix, tokenID...)
	for nonce := startIndex; nonce <= endIndex; nonce++ {
		mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)

		err := systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// input is list(tokenID-nonce-metadata)
func (e *mectDeleteMetaData) addMetadata(args [][]byte) error {
	if len(args)%numArgsPerAdd != 0 || len(args) < numArgsPerAdd {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := 0; i < len(args); i += numArgsPerAdd {
		tokenID := args[i]
		nonce := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		if nonce == 0 {
			return ErrInvalidNonce
		}

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		mectTokenKey := append(e.keyPrefix, tokenID...)
		mectNFTTokenKey := computeMECTNFTTokenKey(mectTokenKey, nonce)
		metaData := &mect.MetaData{}
		err = e.marshaller.Unmarshal(metaData, args[i+2])
		if err != nil {
			return err
		}
		if metaData.Nonce != nonce {
			return ErrInvalidMetadata
		}

		var tokenFromSystemSC *mect.MECToken
		tokenFromSystemSC, err = e.getMECTDigitalTokenDataFromSystemAccount(systemAcc, mectNFTTokenKey)
		if err != nil {
			return err
		}

		if tokenFromSystemSC != nil && tokenFromSystemSC.TokenMetaData != nil {
			return ErrTokenHasValidMetadata
		}

		if tokenFromSystemSC == nil {
			tokenFromSystemSC = &mect.MECToken{
				Value: big.NewInt(0),
				Type:  uint32(core.NonFungible),
			}
		}
		tokenFromSystemSC.TokenMetaData = metaData
		err = e.marshalAndSaveData(systemAcc, tokenFromSystemSC, mectNFTTokenKey)
		if err != nil {
			return err
		}
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *mectDeleteMetaData) getMECTDigitalTokenDataFromSystemAccount(
	systemAcc vmcommon.UserAccountHandler,
	mectNFTTokenKey []byte,
) (*mect.MECToken, error) {
	marshaledData, err := systemAcc.AccountDataHandler().RetrieveValue(mectNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, nil
	}

	mectData := &mect.MECToken{}
	err = e.marshaller.Unmarshal(mectData, marshaledData)
	if err != nil {
		return nil, err
	}

	return mectData, nil
}

func (e *mectDeleteMetaData) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

func (e *mectDeleteMetaData) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	mectData *mect.MECToken,
	mectNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(mectData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(mectNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *mectDeleteMetaData) IsInterfaceNil() bool {
	return e == nil
}
