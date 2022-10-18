package datafield

import (
	"github.com/ME-MotherEarth/me-core/marshal"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
)

// ArgsOperationDataFieldParser holds all the components required to create a new instance of data field parser
type ArgsOperationDataFieldParser struct {
	AddressLength    int
	Marshalizer      marshal.Marshalizer
	ShardCoordinator vmcommon.Coordinator
}
