package builtInFunctions

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/data/mect"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/ME-MotherEarth/me-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestNewMECTRolesFunc_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	mectRolesF, err := NewMECTRolesFunc(nil, false)

	require.Equal(t, ErrNilMarshalizer, err)
	require.Nil(t, mectRolesF)
}

func TestMectRoles_ProcessBuiltinFunction_NilVMInputShouldErr(t *testing.T) {
	t.Parallel()

	mectRolesF, _ := NewMECTRolesFunc(nil, false)

	_, err := mectRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, nil)
	require.Equal(t, ErrNilVmInput, err)
}

func TestMectRoles_ProcessBuiltinFunction_WrongCalledShouldErr(t *testing.T) {
	t.Parallel()

	mectRolesF, _ := NewMECTRolesFunc(nil, false)

	_, err := mectRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: []byte{},
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrAddressIsNotMECTSystemSC, err)
}

func TestMectRoles_ProcessBuiltinFunction_NilAccountDestShouldErr(t *testing.T) {
	t.Parallel()

	mectRolesF, _ := NewMECTRolesFunc(nil, false)

	_, err := mectRolesF.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrNilUserAccount, err)
}

func TestMectRoles_ProcessBuiltinFunction_GetRolesFailShouldErr(t *testing.T) {
	t.Parallel()

	mectRolesF, _ := NewMECTRolesFunc(&mock.MarshalizerMock{Fail: true}, false)

	_, err := mectRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Error(t, err)
}

func TestMectRoles_ProcessBuiltinFunction_GetRolesFailShouldWorkEvenIfAccntTrieIsNil(t *testing.T) {
	t.Parallel()

	saveKeyWasCalled := false
	mectRolesF, _ := NewMECTRolesFunc(&mock.MarshalizerMock{}, false)

	_, err := mectRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, error) {
					return nil, nil
				},
				SaveKeyValueCalled: func(_ []byte, _ []byte) error {
					saveKeyWasCalled = true
					return nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.NoError(t, err)
	require.True(t, saveKeyWasCalled)
}

func TestMectRoles_ProcessBuiltinFunction_SetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, true)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{}
					return marshaller.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &mect.MECTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Equal(t, roles.Roles, [][]byte{[]byte(core.MECTRoleLocalMint)})
					return nil
				},
			}
		},
	}
	_, err := mectRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.MECTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestMectRoles_ProcessBuiltinFunction_SetRolesMultiNFT(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, true)

	tokenID := []byte("tokenID")
	roleKey := append(roleKeyPrefix, tokenID...)

	saveNonceCalled := false
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{}
					return marshaller.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					if bytes.Equal(key, roleKey) {
						roles := &mect.MECTRoles{}
						_ = marshaller.Unmarshal(roles, value)
						require.Equal(t, roles.Roles, [][]byte{[]byte(core.MECTRoleNFTCreate), []byte(core.MECTRoleNFTCreateMultiShard)})
						return nil
					}

					if bytes.Equal(key, getNonceKey(tokenID)) {
						saveNonceCalled = true
						require.Equal(t, uint64(math.MaxUint64/256), big.NewInt(0).SetBytes(value).Uint64())
					}

					return nil
				},
			}
		},
	}
	dstAddr := bytes.Repeat([]byte{1}, 32)
	_, err := mectRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{tokenID, []byte(core.MECTRoleNFTCreate), []byte(core.MECTRoleNFTCreateMultiShard)},
		},
		RecipientAddr: dstAddr,
	})

	require.Nil(t, err)
	require.True(t, saveNonceCalled)
}

func TestMectRoles_ProcessBuiltinFunction_SaveFailedShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, true)

	localErr := errors.New("local err")
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{}
					return marshaller.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return localErr
				},
			}
		},
	}
	_, err := mectRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.MECTRoleLocalMint)},
		},
	})
	require.Equal(t, localErr, err)
}

func TestMectRoles_ProcessBuiltinFunction_UnsetRolesDoesNotExistsShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{}
					return marshaller.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &mect.MECTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := mectRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.MECTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestMectRoles_ProcessBuiltinFunction_UnsetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{
						Roles: [][]byte{[]byte(core.MECTRoleLocalMint)},
					}
					return marshaller.Marshal(roles)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &mect.MECTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := mectRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.MECTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.MECTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestMectRoles_CheckAllowedToExecuteNilAccountShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	err := mectRolesF.CheckAllowedToExecute(nil, []byte("ID"), []byte(core.MECTRoleLocalBurn))
	require.Equal(t, ErrNilUserAccount, err)
}

func TestMectRoles_CheckAllowedToExecuteCannotGetMECTRole(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{Fail: true}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	err := mectRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.MECTRoleLocalBurn))
	require.Error(t, err)
}

func TestMectRoles_CheckAllowedToExecuteIsNewNotAllowed(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	err := mectRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.MECTRoleLocalBurn))
	require.Equal(t, ErrActionNotAllowed, err)
}

func TestMectRoles_CheckAllowed_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	err := mectRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{
						Roles: [][]byte{[]byte(core.MECTRoleLocalMint)},
					}
					return marshaller.Marshal(roles)
				},
			}
		},
	}, []byte("ID"), []byte(core.MECTRoleLocalMint))
	require.Nil(t, err)
}

func TestMectRoles_CheckAllowedToExecuteRoleNotFind(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	mectRolesF, _ := NewMECTRolesFunc(marshaller, false)

	err := mectRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					roles := &mect.MECTRoles{
						Roles: [][]byte{[]byte(core.MECTRoleLocalBurn)},
					}
					return marshaller.Marshal(roles)
				},
			}
		},
	}, []byte("ID"), []byte(core.MECTRoleLocalMint))
	require.Equal(t, ErrActionNotAllowed, err)
}
