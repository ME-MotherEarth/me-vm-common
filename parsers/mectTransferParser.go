package parsers

import (
	"bytes"
	"math/big"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

// MinArgsForMECTTransfer defines the minimum arguments needed for an mect transfer
const MinArgsForMECTTransfer = 2

// MinArgsForMECTNFTTransfer defines the minimum arguments needed for an nft transfer
const MinArgsForMECTNFTTransfer = 4

// MinArgsForMultiMECTNFTTransfer defines the minimum arguments needed for a multi transfer
const MinArgsForMultiMECTNFTTransfer = 4

// ArgsPerTransfer defines the number of arguments per transfer in multi transfer
const ArgsPerTransfer = 3

type mectTransferParser struct {
	marshaller vmcommon.Marshalizer
}

// NewMECTTransferParser creates a new mect transfer parser
func NewMECTTransferParser(
	marshaller vmcommon.Marshalizer,
) (*mectTransferParser, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	return &mectTransferParser{marshaller: marshaller}, nil
}

// ParseMECTTransfers returns the list of mect transfers, the callFunction and callArgs from the given arguments
func (e *mectTransferParser) ParseMECTTransfers(
	sndAddr []byte,
	rcvAddr []byte,
	function string,
	args [][]byte,
) (*vmcommon.ParsedMECTTransfers, error) {
	switch function {
	case core.BuiltInFunctionMECTTransfer:
		return e.parseSingleMECTTransfer(rcvAddr, args)
	case core.BuiltInFunctionMECTNFTTransfer:
		return e.parseSingleMECTNFTTransfer(sndAddr, rcvAddr, args)
	case core.BuiltInFunctionMultiMECTNFTTransfer:
		return e.parseMultiMECTNFTTransfer(rcvAddr, args)
	default:
		return nil, ErrNotMECTTransferInput
	}
}

func (e *mectTransferParser) parseSingleMECTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedMECTTransfers, error) {
	if len(args) < MinArgsForMECTTransfer {
		return nil, ErrNotEnoughArguments
	}
	mectTransfers := &vmcommon.ParsedMECTTransfers{
		MECTTransfers: make([]*vmcommon.MECTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}
	if len(args) > MinArgsForMECTTransfer {
		mectTransfers.CallFunction = string(args[MinArgsForMECTTransfer])
	}
	if len(args) > MinArgsForMECTTransfer+1 {
		mectTransfers.CallArgs = append(mectTransfers.CallArgs, args[MinArgsForMECTTransfer+1:]...)
	}
	mectTransfers.MECTTransfers[0] = &vmcommon.MECTTransfer{
		MECTValue:      big.NewInt(0).SetBytes(args[1]),
		MECTTokenName:  args[0],
		MECTTokenType:  uint32(core.Fungible),
		MECTTokenNonce: 0,
	}

	return mectTransfers, nil
}

func (e *mectTransferParser) parseSingleMECTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*vmcommon.ParsedMECTTransfers, error) {
	if len(args) < MinArgsForMECTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	mectTransfers := &vmcommon.ParsedMECTTransfers{
		MECTTransfers: make([]*vmcommon.MECTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}

	if bytes.Equal(sndAddr, rcvAddr) {
		mectTransfers.RcvAddr = args[3]
	}
	if len(args) > MinArgsForMECTNFTTransfer {
		mectTransfers.CallFunction = string(args[MinArgsForMECTNFTTransfer])
	}
	if len(args) > MinArgsForMECTNFTTransfer+1 {
		mectTransfers.CallArgs = append(mectTransfers.CallArgs, args[MinArgsForMECTNFTTransfer+1:]...)
	}
	mectTransfers.MECTTransfers[0] = &vmcommon.MECTTransfer{
		MECTValue:      big.NewInt(0).SetBytes(args[2]),
		MECTTokenName:  args[0],
		MECTTokenType:  uint32(core.NonFungible),
		MECTTokenNonce: big.NewInt(0).SetBytes(args[1]).Uint64(),
	}

	return mectTransfers, nil
}

func (e *mectTransferParser) parseMultiMECTNFTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedMECTTransfers, error) {
	if len(args) < MinArgsForMultiMECTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	mectTransfers := &vmcommon.ParsedMECTTransfers{
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	numOfTransfer := big.NewInt(0).SetBytes(args[0])
	startIndex := uint64(1)
	isTxAtSender := false

	isFirstArgumentAnAddress := len(args[0]) == len(rcvAddr) && !numOfTransfer.IsUint64()
	if isFirstArgumentAnAddress {
		mectTransfers.RcvAddr = args[0]
		numOfTransfer.SetBytes(args[1])
		startIndex = 2
		isTxAtSender = true
	}

	minLenArgs := ArgsPerTransfer*numOfTransfer.Uint64() + startIndex
	if uint64(len(args)) < minLenArgs {
		return nil, ErrNotEnoughArguments
	}

	if uint64(len(args)) > minLenArgs {
		mectTransfers.CallFunction = string(args[minLenArgs])
	}
	if uint64(len(args)) > minLenArgs+1 {
		mectTransfers.CallArgs = append(mectTransfers.CallArgs, args[minLenArgs+1:]...)
	}

	var err error
	mectTransfers.MECTTransfers = make([]*vmcommon.MECTTransfer, numOfTransfer.Uint64())
	for i := uint64(0); i < numOfTransfer.Uint64(); i++ {
		tokenStartIndex := startIndex + i*ArgsPerTransfer
		mectTransfers.MECTTransfers[i], err = e.createNewMECTTransfer(tokenStartIndex, args, isTxAtSender)
		if err != nil {
			return nil, err
		}
	}

	return mectTransfers, nil
}

func (e *mectTransferParser) createNewMECTTransfer(
	tokenStartIndex uint64,
	args [][]byte,
	isTxAtSender bool,
) (*vmcommon.MECTTransfer, error) {
	mectTransfer := &vmcommon.MECTTransfer{
		MECTValue:      big.NewInt(0).SetBytes(args[tokenStartIndex+2]),
		MECTTokenName:  args[tokenStartIndex],
		MECTTokenType:  uint32(core.Fungible),
		MECTTokenNonce: big.NewInt(0).SetBytes(args[tokenStartIndex+1]).Uint64(),
	}
	if mectTransfer.MECTTokenNonce > 0 {
		mectTransfer.MECTTokenType = uint32(core.NonFungible)

		if !isTxAtSender && len(args[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			transferMECTData := &mect.MECToken{}
			err := e.marshaller.Unmarshal(transferMECTData, args[tokenStartIndex+2])
			if err != nil {
				return nil, err
			}
			mectTransfer.MECTValue.Set(transferMECTData.Value)
		}
	}

	return mectTransfer, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *mectTransferParser) IsInterfaceNil() bool {
	return e == nil
}
