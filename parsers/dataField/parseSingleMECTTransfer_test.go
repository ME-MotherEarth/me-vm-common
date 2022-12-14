package datafield

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMECTTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgumentsOperationParser()
	parser, _ := NewOperationDataFieldParser(args)

	t.Run("TransferNonHexArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("MECTTransfer@1234@011")
		res := parser.Parse(dataField, sender, receiver)
		require.Equal(t, &ResponseParseData{
			Operation: operationTransfer,
		}, res)
	})

	t.Run("TransferNotEnoughArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("MECTTransfer@1234")
		res := parser.Parse(dataField, sender, receiver)
		require.Equal(t, &ResponseParseData{
			Operation: "MECTTransfer",
		}, res)
	})

	t.Run("TransferEmptyArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("MECTTransfer@544f4b454e@")
		res := parser.Parse(dataField, sender, receiver)
		require.Equal(t, &ResponseParseData{
			Operation:  "MECTTransfer",
			Tokens:     []string{"TOKEN"},
			MECTValues: []string{"0"},
		}, res)
	})

	t.Run("TransferWithSCCall", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("MECTTransfer@544f4b454e@01@63616c6c4d65")
		res := parser.Parse(dataField, sender, receiverSC)
		require.Equal(t, &ResponseParseData{
			Operation:  "MECTTransfer",
			Function:   "callMe",
			MECTValues: []string{"1"},
			Tokens:     []string{"TOKEN"},
		}, res)
	})

	t.Run("TransferNonAsciiStringToken", func(t *testing.T) {
		dataField := []byte("MECTTransfer@055de6a779bbac0000@01")
		res := parser.Parse(dataField, sender, receiverSC)
		require.Equal(t, &ResponseParseData{
			Operation: "MECTTransfer",
		}, res)
	})
}
