package datafield

import "github.com/ME-MotherEarth/me-core/core"

func (odp *operationDataFieldParser) parseMultiMECTNFTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedMECTTransfers, ok := odp.extractMECTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}
	if core.IsSmartContractAddress(parsedMECTTransfers.RcvAddr) && isASCIIString(parsedMECTTransfers.CallFunction) {
		responseParse.Function = parsedMECTTransfers.CallFunction
	}

	receiverShardID := odp.shardCoordinator.ComputeId(parsedMECTTransfers.RcvAddr)
	for _, mectTransferData := range parsedMECTTransfers.MECTTransfers {
		if !isASCIIString(string(mectTransferData.MECTTokenName)) {
			return &ResponseParseData{
				Operation: function,
			}
		}

		token := string(mectTransferData.MECTTokenName)
		if mectTransferData.MECTTokenNonce != 0 {
			token = computeTokenIdentifier(token, mectTransferData.MECTTokenNonce)
		}

		responseParse.Tokens = append(responseParse.Tokens, token)
		responseParse.MECTValues = append(responseParse.MECTValues, mectTransferData.MECTValue.String())
		responseParse.Receivers = append(responseParse.Receivers, parsedMECTTransfers.RcvAddr)
		responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)
	}

	return responseParse
}
