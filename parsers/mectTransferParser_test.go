package parsers

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
)

var sndAddr = bytes.Repeat([]byte{1}, 32)
var dstAddr = bytes.Repeat([]byte{1}, 32)

func TestNewMECTTransferParser(t *testing.T) {
	t.Parallel()

	mectParser, err := NewMECTTransferParser(nil)
	assert.Nil(t, mectParser)
	assert.Equal(t, err, ErrNilMarshalizer)

	mectParser, err = NewMECTTransferParser(&mock.MarshalizerMock{})
	assert.Nil(t, err)
	assert.False(t, mectParser.IsInterfaceNil())
}

func TestMectTransferParser_ParseMECTTransfersWrongFunction(t *testing.T) {
	t.Parallel()

	mectParser, _ := NewMECTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := mectParser.ParseMECTTransfers(nil, nil, "some", nil)
	assert.Equal(t, err, ErrNotMECTTransferInput)
	assert.Nil(t, parsedData)
}

func TestMectTransferParser_ParseSingleMECTFunction(t *testing.T) {
	t.Parallel()

	mectParser, _ := NewMECTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := mectParser.ParseMECTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionMECTTransfer,
		[][]byte{[]byte("one")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = mectParser.ParseMECTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionMECTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(10).Uint64())

	parsedData, err = mectParser.ParseMECTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionMECTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}

func TestMectTransferParser_ParseSingleNFTTransfer(t *testing.T) {
	t.Parallel()

	mectParser, _ := NewMECTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := mectParser.ParseMECTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionMECTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMECTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMECTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMECTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())
}

func TestMectTransferParser_ParseMultiNFTTransferTransferOne(t *testing.T) {
	t.Parallel()

	mectParser, _ := NewMECTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := mectParser.ParseMECTTransfers(
		nil,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	mectData := &mect.MECToken{Value: big.NewInt(20)}
	marshaled, _ := mectParser.marshaller.Marshal(mectData)

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())
}

func TestMectTransferParser_ParseMultiNFTTransferTransferMore(t *testing.T) {
	t.Parallel()

	mectParser, _ := NewMECTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[1].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[1].MECTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.MECTTransfers[1].MECTTokenType, uint32(core.Fungible))

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	mectData := &mect.MECToken{Value: big.NewInt(20)}
	marshaled, _ := mectParser.marshaller.Marshal(mectData)
	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.MECTTransfers[0].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[0].MECTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[1].MECTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.MECTTransfers[1].MECTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.MECTTransfers[1].MECTTokenType, uint32(core.Fungible))

	parsedData, err = mectParser.ParseMECTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiMECTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.MECTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}
