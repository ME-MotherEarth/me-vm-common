package vmcommon

// BaseOperationCost defines cost for base operation cost
type BaseOperationCost struct {
	StorePerByte      uint64
	ReleasePerByte    uint64
	DataCopyPerByte   uint64
	PersistPerByte    uint64
	CompilePerByte    uint64
	AoTPreparePerByte uint64
}

// BuiltInCost defines cost for built-in methods
type BuiltInCost struct {
	ChangeOwnerAddress       uint64
	ClaimDeveloperRewards    uint64
	SaveUserName             uint64
	SaveKeyValue             uint64
	MECTTransfer             uint64
	MECTBurn                 uint64
	MECTLocalMint            uint64
	MECTLocalBurn            uint64
	MECTNFTCreate            uint64
	MECTNFTAddQuantity       uint64
	MECTNFTBurn              uint64
	MECTNFTTransfer          uint64
	MECTNFTChangeCreateOwner uint64
	MECTNFTMultiTransfer     uint64
	MECTNFTAddURI            uint64
	MECTNFTUpdateAttributes  uint64
}

// GasCost holds all the needed gas costs for system smart contracts
type GasCost struct {
	BaseOperationCost BaseOperationCost
	BuiltInCost       BuiltInCost
}

// SafeSubUint64 performs subtraction on uint64 and returns an error if it overflows
func SafeSubUint64(a, b uint64) (uint64, error) {
	if a < b {
		return 0, ErrSubtractionOverflow
	}
	return a - b, nil
}
