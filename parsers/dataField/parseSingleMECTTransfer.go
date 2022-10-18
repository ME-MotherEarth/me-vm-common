package datafield

import (
	"github.com/ME-MotherEarth/me-core/core"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

func (odp *operationDataFieldParser) parseSingleMECTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedMECTTransfers, ok := odp.extractMECTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(receiver) && isASCIIString(parsedMECTTransfers.CallFunction) {
		responseParse.Function = parsedMECTTransfers.CallFunction
	}

	if len(parsedMECTTransfers.MECTTransfers) == 0 || !isASCIIString(string(parsedMECTTransfers.MECTTransfers[0].MECTTokenName)) {
		return responseParse
	}

	firstTransfer := parsedMECTTransfers.MECTTransfers[0]
	responseParse.Tokens = append(responseParse.Tokens, string(firstTransfer.MECTTokenName))
	responseParse.MECTValues = append(responseParse.MECTValues, firstTransfer.MECTValue.String())

	return responseParse
}

func (odp *operationDataFieldParser) extractMECTData(args [][]byte, function string, sender, receiver []byte) (*ResponseParseData, *vmcommon.ParsedMECTTransfers, bool) {
	responseParse := &ResponseParseData{
		Operation: function,
	}

	parsedMECTTransfers, err := odp.mectTransferParser.ParseMECTTransfers(sender, receiver, function, args)
	if err != nil {
		return responseParse, nil, false
	}

	return responseParse, parsedMECTTransfers, true
}
