package datafield

import (
	"bytes"

	"github.com/ME-MotherEarth/me-core/core"
)

func (odp *operationDataFieldParser) parseSingleMECTNFTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedMECTTransfers, ok := odp.extractMECTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(parsedMECTTransfers.RcvAddr) && isASCIIString(parsedMECTTransfers.CallFunction) {
		responseParse.Function = parsedMECTTransfers.CallFunction
	}

	if len(parsedMECTTransfers.MECTTransfers) == 0 || !isASCIIString(string(parsedMECTTransfers.MECTTransfers[0].MECTTokenName)) {
		return responseParse
	}

	rcvAddr := receiver
	if bytes.Equal(sender, receiver) {
		rcvAddr = parsedMECTTransfers.RcvAddr
	}

	mectNFTTransfer := parsedMECTTransfers.MECTTransfers[0]
	receiverShardID := odp.shardCoordinator.ComputeId(rcvAddr)
	token := computeTokenIdentifier(string(mectNFTTransfer.MECTTokenName), mectNFTTransfer.MECTTokenNonce)

	responseParse.Tokens = append(responseParse.Tokens, token)
	responseParse.MECTValues = append(responseParse.MECTValues, mectNFTTransfer.MECTValue.String())
	responseParse.Receivers = append(responseParse.Receivers, rcvAddr)
	responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)

	return responseParse
}
