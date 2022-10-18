package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

type mectFreezeWipe struct {
	baseAlwaysActive
	marshaller vmcommon.Marshalizer
	keyPrefix  []byte
	wipe       bool
	freeze     bool
}

// NewMECTFreezeWipeFunc returns the mect freeze/un-freeze/wipe built-in function component
func NewMECTFreezeWipeFunc(
	marshaller vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*mectFreezeWipe, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	e := &mectFreezeWipe{
		marshaller: marshaller,
		keyPrefix:  []byte(baseMECTKeyPrefix),
		freeze:     freeze,
		wipe:       wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *mectFreezeWipe) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves MECT transfer function call
func (e *mectFreezeWipe) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
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
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	mectTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	var amount *big.Int
	var err error

	if e.wipe {
		amount, err = e.wipeIfApplicable(acntDst, mectTokenKey)
		if err != nil {
			return nil, err
		}

	} else {
		amount, err = e.toggleFreeze(acntDst, mectTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	identifier, nonce := extractTokenIdentifierAndNonceMECTWipe(vmInput.Arguments[0])
	addMECTEntryInVMOutput(vmOutput, []byte(vmInput.Function), identifier, nonce, amount, vmInput.CallerAddr, acntDst.AddressBytes())

	return vmOutput, nil
}

func (e *mectFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte) (*big.Int, error) {
	tokenData, err := getMECTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	mectUserMetadata := MECTUserMetadataFromBytes(tokenData.Properties)
	if !mectUserMetadata.Frozen {
		return nil, ErrCannotWipeAccountNotFrozen
	}

	err = acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
	if err != nil {
		return nil, err
	}

	wipedAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return wipedAmount, nil
}

func (e *mectFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) (*big.Int, error) {
	tokenData, err := getMECTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	mectUserMetadata := MECTUserMetadataFromBytes(tokenData.Properties)
	mectUserMetadata.Frozen = e.freeze
	tokenData.Properties = mectUserMetadata.ToBytes()

	err = saveMECTData(acntDst, tokenData, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	frozenAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return frozenAmount, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *mectFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
