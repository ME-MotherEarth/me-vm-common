package mock

import vmcommon "github.com/ME-MotherEarth/me-vm-common"

// MECTRoleHandlerStub -
type MECTRoleHandlerStub struct {
	CheckAllowedToExecuteCalled func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error
}

// CheckAllowedToExecute -
func (e *MECTRoleHandlerStub) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if e.CheckAllowedToExecuteCalled != nil {
		return e.CheckAllowedToExecuteCalled(account, tokenID, action)
	}

	return nil
}

// IsInterfaceNil -
func (e *MECTRoleHandlerStub) IsInterfaceNil() bool {
	return e == nil
}
