package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

const (
	mectIdentifierSeparator  = "-"
	mectRandomSequenceLength = 6
)

func addMECTEntryInVMOutput(vmOutput *vmcommon.VMOutput, identifier []byte, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) {
	entry := newEntryForMECT(identifier, tokenID, nonce, value, args...)

	if vmOutput.Logs == nil {
		vmOutput.Logs = make([]*vmcommon.LogEntry, 0, 1)
	}

	vmOutput.Logs = append(vmOutput.Logs, entry)
}

func newEntryForMECT(identifier, tokenID []byte, nonce uint64, value *big.Int, args ...[]byte) *vmcommon.LogEntry {
	nonceBig := big.NewInt(0).SetUint64(nonce)

	logEntry := &vmcommon.LogEntry{
		Identifier: identifier,
		Topics:     [][]byte{tokenID, nonceBig.Bytes(), value.Bytes()},
	}

	if len(args) > 0 {
		logEntry.Address = args[0]
	}

	if len(args) > 1 {
		logEntry.Topics = append(logEntry.Topics, args[1:]...)
	}

	return logEntry
}

func extractTokenIdentifierAndNonceMECTWipe(args []byte) ([]byte, uint64) {
	argsSplit := bytes.Split(args, []byte(mectIdentifierSeparator))
	if len(argsSplit) < 2 {
		return args, 0
	}

	if len(argsSplit[1]) <= mectRandomSequenceLength {
		return args, 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:mectRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][mectRandomSequenceLength:])

	return identifier, nonce.Uint64()
}

func boolToSlice(b bool) []byte {
	return []byte(strconv.FormatBool(b))
}
