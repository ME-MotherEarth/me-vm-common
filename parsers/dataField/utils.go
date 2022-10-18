package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode"

	"github.com/ME-MotherEarth/me-core/core"
)

const (
	mectIdentifierSeparator  = "-"
	mectRandomSequenceLength = 6
)

func getAllBuiltInFunctions() []string {
	return []string{
		core.BuiltInFunctionClaimDeveloperRewards,
		core.BuiltInFunctionChangeOwnerAddress,
		core.BuiltInFunctionSetUserName,
		core.BuiltInFunctionSaveKeyValue,
		core.BuiltInFunctionMECTTransfer,
		core.BuiltInFunctionMECTBurn,
		core.BuiltInFunctionMECTFreeze,
		core.BuiltInFunctionMECTUnFreeze,
		core.BuiltInFunctionMECTWipe,
		core.BuiltInFunctionMECTPause,
		core.BuiltInFunctionMECTUnPause,
		core.BuiltInFunctionSetMECTRole,
		core.BuiltInFunctionUnSetMECTRole,
		core.BuiltInFunctionMECTSetLimitedTransfer,
		core.BuiltInFunctionMECTUnSetLimitedTransfer,
		core.BuiltInFunctionMECTLocalMint,
		core.BuiltInFunctionMECTLocalBurn,
		core.BuiltInFunctionMECTNFTTransfer,
		core.BuiltInFunctionMECTNFTCreate,
		core.BuiltInFunctionMECTNFTAddQuantity,
		core.BuiltInFunctionMECTNFTCreateRoleTransfer,
		core.BuiltInFunctionMECTNFTBurn,
		core.BuiltInFunctionMECTNFTAddURI,
		core.BuiltInFunctionMECTNFTUpdateAttributes,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		core.MECTRoleLocalMint,
		core.MECTRoleLocalBurn,
		core.MECTRoleNFTCreate,
		core.MECTRoleNFTCreateMultiShard,
		core.MECTRoleNFTAddQuantity,
		core.MECTRoleNFTBurn,
		core.MECTRoleNFTAddURI,
		core.MECTRoleNFTUpdateAttributes,
		core.MECTRoleTransfer,
	}
}

func isBuiltInFunction(builtInFunctionsList []string, function string) bool {
	for _, builtInFunction := range builtInFunctionsList {
		if builtInFunction == function {
			return true
		}
	}

	return false
}

// EncodeBytesSlice will encode the provided bytes slice with a provided function
func EncodeBytesSlice(encodeFunc func(b []byte) string, rcvs [][]byte) []string {
	if encodeFunc == nil {
		return nil
	}

	encodedSlice := make([]string, 0, len(rcvs))
	for _, rcv := range rcvs {
		encodedSlice = append(encodedSlice, encodeFunc(rcv))
	}

	return encodedSlice
}

func computeTokenIdentifier(token string, nonce uint64) string {
	if token == "" || nonce == 0 {
		return ""
	}

	nonceBig := big.NewInt(0).SetUint64(nonce)
	hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())
	return fmt.Sprintf("%s-%s", token, hexEncodedNonce)
}

func extractTokenAndNonce(arg []byte) (string, uint64) {
	argsSplit := bytes.Split(arg, []byte(mectIdentifierSeparator))
	if len(argsSplit) < 2 {
		return string(arg), 0
	}

	if len(argsSplit[1]) <= mectRandomSequenceLength {
		return string(arg), 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:mectRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][mectRandomSequenceLength:])

	return string(identifier), nonce.Uint64()
}

func isEmptyAddr(addrLength int, address []byte) bool {
	emptyAddr := make([]byte, addrLength)

	return bytes.Equal(address, emptyAddr)
}

func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
