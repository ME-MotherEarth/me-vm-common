package builtInFunctions

const lengthOfMECTMetadata = 2

const (
	// MetadataPaused is the location of paused flag in the mect global meta data
	MetadataPaused = 1
	// MetadataLimitedTransfer is the location of limited transfer flag in the mect global meta data
	MetadataLimitedTransfer = 2
	// BurnRoleForAll is the location of burn role for all flag in the mect global meta data
	BurnRoleForAll = 4
)

const (
	// MetadataFrozen is the location of frozen flag in the mect user meta data
	MetadataFrozen = 1
)

// MECTGlobalMetadata represents mect global metadata saved on system account
type MECTGlobalMetadata struct {
	Paused          bool
	LimitedTransfer bool
	BurnRoleForAll  bool
}

// MECTGlobalMetadataFromBytes creates a metadata object from bytes
func MECTGlobalMetadataFromBytes(bytes []byte) MECTGlobalMetadata {
	if len(bytes) != lengthOfMECTMetadata {
		return MECTGlobalMetadata{}
	}

	return MECTGlobalMetadata{
		Paused:          (bytes[0] & MetadataPaused) != 0,
		LimitedTransfer: (bytes[0] & MetadataLimitedTransfer) != 0,
		BurnRoleForAll:  (bytes[0] & BurnRoleForAll) != 0,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *MECTGlobalMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfMECTMetadata)

	if metadata.Paused {
		bytes[0] |= MetadataPaused
	}
	if metadata.LimitedTransfer {
		bytes[0] |= MetadataLimitedTransfer
	}
	if metadata.BurnRoleForAll {
		bytes[0] |= BurnRoleForAll
	}

	return bytes
}

// MECTUserMetadata represents mect user metadata saved on every account
type MECTUserMetadata struct {
	Frozen bool
}

// MECTUserMetadataFromBytes creates a metadata object from bytes
func MECTUserMetadataFromBytes(bytes []byte) MECTUserMetadata {
	if len(bytes) != lengthOfMECTMetadata {
		return MECTUserMetadata{}
	}

	return MECTUserMetadata{
		Frozen: (bytes[0] & MetadataFrozen) != 0,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *MECTUserMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfMECTMetadata)

	if metadata.Frozen {
		bytes[0] |= MetadataFrozen
	}

	return bytes
}
