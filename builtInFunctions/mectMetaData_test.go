package builtInFunctions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMECTGlobalMetaData_ToBytesWhenPaused(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTGlobalMetadata{
		Paused: true,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 1
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTGlobalMetaData_ToBytesWhenTransfer(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTGlobalMetadata{
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 2
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTGlobalMetaData_ToBytesWhenTransferAndPause(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTGlobalMetadata{
		Paused:          true,
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 3
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTGlobalMetaData_ToBytesWhenNotPaused(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTGlobalMetadata{
		Paused: false,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 0
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTGlobalMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyMectGlobalMetaData := MECTGlobalMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfMECTMetadata+1)

	result := MECTGlobalMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyMectGlobalMetaData, result)
}

func TestMECTGlobalMetadataFromBytes_ShouldSetPausedToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfMECTMetadata)
	input[0] = 1

	result := MECTGlobalMetadataFromBytes(input)
	require.True(t, result.Paused)
}

func TestMECTGlobalMetadataFromBytes_ShouldSetPausedToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfMECTMetadata)
	input[0] = 0

	result := MECTGlobalMetadataFromBytes(input)
	require.False(t, result.Paused)
}

func TestMECTUserMetaData_ToBytesWhenFrozen(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTUserMetadata{
		Frozen: true,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 1
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTUserMetaData_ToBytesWhenNotFrozen(t *testing.T) {
	t.Parallel()

	mectMetaData := &MECTUserMetadata{
		Frozen: false,
	}

	expected := make([]byte, lengthOfMECTMetadata)
	expected[0] = 0
	actual := mectMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestMECTUserMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyMectUserMetaData := MECTUserMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfMECTMetadata+1)

	result := MECTUserMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyMectUserMetaData, result)
}

func TestMECTUserMetadataFromBytes_ShouldSetFrozenToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfMECTMetadata)
	input[0] = 1

	result := MECTUserMetadataFromBytes(input)
	require.True(t, result.Frozen)
}

func TestMECTUserMetadataFromBytes_ShouldSetFrozenToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfMECTMetadata)
	input[0] = 0

	result := MECTUserMetadataFromBytes(input)
	require.False(t, result.Frozen)
}

func TestMECTGlobalMetadata_FromBytes(t *testing.T) {
	require.True(t, MECTGlobalMetadataFromBytes([]byte{1, 0}).Paused)
	require.False(t, MECTGlobalMetadataFromBytes([]byte{1, 0}).LimitedTransfer)
	require.True(t, MECTGlobalMetadataFromBytes([]byte{2, 0}).LimitedTransfer)
	require.False(t, MECTGlobalMetadataFromBytes([]byte{2, 0}).Paused)
	require.False(t, MECTGlobalMetadataFromBytes([]byte{0, 0}).LimitedTransfer)
	require.False(t, MECTGlobalMetadataFromBytes([]byte{0, 0}).Paused)
	require.True(t, MECTGlobalMetadataFromBytes([]byte{3, 0}).Paused)
	require.True(t, MECTGlobalMetadataFromBytes([]byte{3, 0}).LimitedTransfer)
}
