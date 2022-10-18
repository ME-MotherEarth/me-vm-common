package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/stretchr/testify/require"
)

func TestNewEntryForNFT(t *testing.T) {
	t.Parallel()

	vmOutput := &vmcommon.VMOutput{}
	addMECTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionMECTNFTCreate), []byte("my-token"), 5, big.NewInt(1), []byte("caller"), []byte("receiver"))
	require.Equal(t, &vmcommon.LogEntry{
		Identifier: []byte(core.BuiltInFunctionMECTNFTCreate),
		Address:    []byte("caller"),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).SetUint64(5).Bytes(), big.NewInt(1).Bytes(), []byte("receiver")},
		Data:       nil,
	}, vmOutput.Logs[0])
}

func TestExtractTokenIdentifierAndNonceMECTWipe(t *testing.T) {
	t.Parallel()

	hexArg := "534b4537592d37336262636404"
	args, _ := hex.DecodeString(hexArg)

	identifier, nonce := extractTokenIdentifierAndNonceMECTWipe(args)
	require.Equal(t, uint64(4), nonce)
	require.Equal(t, []byte("SKE7Y-73bbcd"), identifier)

	hexArg = "574d4f412d376662623930"
	args, _ = hex.DecodeString(hexArg)

	identifier, nonce = extractTokenIdentifierAndNonceMECTWipe(args)
	require.Equal(t, uint64(0), nonce)
	require.Equal(t, []byte("WMOA-7fbb90"), identifier)
}
