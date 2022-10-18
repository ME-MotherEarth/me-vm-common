package builtInFunctions

import (
	"github.com/ME-MotherEarth/me-core/core"
	"github.com/ME-MotherEarth/me-core/core/check"
	vmcommon "github.com/ME-MotherEarth/me-vm-common"
	"github.com/mitchellh/mapstructure"
)

var _ vmcommon.BuiltInFunctionFactory = (*builtInFuncCreator)(nil)

// ArgsCreateBuiltInFunctionContainer defines the input arguments to create built in functions container
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                              map[string]map[string]uint64
	MapDNSAddresses                     map[string]struct{}
	EnableUserNameChange                bool
	Marshalizer                         vmcommon.Marshalizer
	Accounts                            vmcommon.AccountsAdapter
	ShardCoordinator                    vmcommon.Coordinator
	EpochNotifier                       vmcommon.EpochNotifier
	MECTNFTImprovementV1ActivationEpoch uint32
	MECTTransferRoleEnableEpoch         uint32
	GlobalMintBurnDisableEpoch          uint32
	MECTTransferToMetaEnableEpoch       uint32
	NFTCreateMultiShardEnableEpoch      uint32
	SaveNFTToSystemAccountEnableEpoch   uint32
	CheckCorrectTokenIDEnableEpoch      uint32
	SendMECTMetadataAlwaysEnableEpoch   uint32
	CheckFunctionArgumentEnableEpoch    uint32
	FixAsyncCallbackCheckEnableEpoch    uint32
	FixOldTokenLiquidityEnableEpoch     uint32
	MaxNumOfAddressesForTransferRole    uint32
	ConfigAddress                       []byte
}

type builtInFuncCreator struct {
	mapDNSAddresses                     map[string]struct{}
	enableUserNameChange                bool
	marshaller                          vmcommon.Marshalizer
	accounts                            vmcommon.AccountsAdapter
	builtInFunctions                    vmcommon.BuiltInFunctionContainer
	gasConfig                           *vmcommon.GasCost
	shardCoordinator                    vmcommon.Coordinator
	epochNotifier                       vmcommon.EpochNotifier
	mectStorageHandler                  vmcommon.MECTNFTStorageHandler
	mectGlobalSettingsHandler           vmcommon.MECTGlobalSettingsHandler
	mectNFTImprovementV1ActivationEpoch uint32
	mectTransferRoleEnableEpoch         uint32
	globalMintBurnDisableEpoch          uint32
	mectTransferToMetaEnableEpoch       uint32
	nftCreateMultiShardEnableEpoch      uint32
	saveNFTToSystemAccountEnableEpoch   uint32
	checkCorrectTokenIDEnableEpoch      uint32
	sendMECTMetadataAlwaysEnableEpoch   uint32
	checkFunctionArgumentEnableEpoch    uint32
	fixAsnycCallbackCheckEnableEpoch    uint32
	fixOldTokenLiquidityEnableEpoch     uint32
	maxNumOfAddressesForTransferRole    uint32
	configAddress                       []byte
}

// NewBuiltInFunctionsCreator creates a component which will instantiate the built in functions contracts
func NewBuiltInFunctionsCreator(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncCreator, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.EpochNotifier) {
		return nil, ErrNilEpochHandler
	}

	b := &builtInFuncCreator{
		mapDNSAddresses:                     args.MapDNSAddresses,
		enableUserNameChange:                args.EnableUserNameChange,
		marshaller:                          args.Marshalizer,
		accounts:                            args.Accounts,
		shardCoordinator:                    args.ShardCoordinator,
		epochNotifier:                       args.EpochNotifier,
		mectNFTImprovementV1ActivationEpoch: args.MECTNFTImprovementV1ActivationEpoch,
		mectTransferRoleEnableEpoch:         args.MECTTransferRoleEnableEpoch,
		globalMintBurnDisableEpoch:          args.GlobalMintBurnDisableEpoch,
		mectTransferToMetaEnableEpoch:       args.MECTTransferToMetaEnableEpoch,
		nftCreateMultiShardEnableEpoch:      args.NFTCreateMultiShardEnableEpoch,
		saveNFTToSystemAccountEnableEpoch:   args.SaveNFTToSystemAccountEnableEpoch,
		checkCorrectTokenIDEnableEpoch:      args.CheckCorrectTokenIDEnableEpoch,
		sendMECTMetadataAlwaysEnableEpoch:   args.SendMECTMetadataAlwaysEnableEpoch,
		checkFunctionArgumentEnableEpoch:    args.CheckFunctionArgumentEnableEpoch,
		fixOldTokenLiquidityEnableEpoch:     args.FixOldTokenLiquidityEnableEpoch,
		maxNumOfAddressesForTransferRole:    args.MaxNumOfAddressesForTransferRole,
		configAddress:                       args.ConfigAddress,
	}

	var err error
	b.gasConfig, err = createGasConfig(args.GasMap)
	if err != nil {
		return nil, err
	}
	b.builtInFunctions = NewBuiltInFunctionContainer()

	return b, nil
}

// GasScheduleChange is called when gas schedule is changed, thus all contracts must be updated
func (b *builtInFuncCreator) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
	newGasConfig, err := createGasConfig(gasSchedule)
	if err != nil {
		return
	}

	b.gasConfig = newGasConfig
	for key := range b.builtInFunctions.Keys() {
		builtInFunc, errGet := b.builtInFunctions.Get(key)
		if errGet != nil {
			return
		}

		builtInFunc.SetNewGasConfig(b.gasConfig)
	}
}

// NFTStorageHandler will return the mect storage handler from the built in functions factory
func (b *builtInFuncCreator) NFTStorageHandler() vmcommon.SimpleMECTNFTStorageHandler {
	return b.mectStorageHandler
}

// MECTGlobalSettingsHandler will return the mect global settings handler from the built in functions factory
func (b *builtInFuncCreator) MECTGlobalSettingsHandler() vmcommon.MECTGlobalSettingsHandler {
	return b.mectGlobalSettingsHandler
}

// BuiltInFunctionContainer will return the built in function container
func (b *builtInFuncCreator) BuiltInFunctionContainer() vmcommon.BuiltInFunctionContainer {
	return b.builtInFunctions
}

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncCreator) CreateBuiltInFunctionContainer() error {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return err
	}

	newFunc = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress)
	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.enableUserNameChange)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return err
	}

	globalSettingsFunc, err := NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, true, core.BuiltInFunctionMECTPause, 0, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTPause, globalSettingsFunc)
	if err != nil {
		return err
	}
	b.mectGlobalSettingsHandler = globalSettingsFunc

	setRoleFunc, err := NewMECTRolesFunc(b.marshaller, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetMECTRole, setRoleFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTTransferFunc(
		b.gasConfig.BuiltInCost.MECTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.shardCoordinator,
		setRoleFunc,
		b.mectTransferToMetaEnableEpoch,
		b.checkCorrectTokenIDEnableEpoch,
		b.epochNotifier,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTBurnFunc(b.gasConfig.BuiltInCost.MECTBurn, b.marshaller, globalSettingsFunc, b.globalMintBurnDisableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTFreezeWipeFunc(b.marshaller, true, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTFreezeWipeFunc(b.marshaller, false, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTUnFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTFreezeWipeFunc(b.marshaller, false, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTWipe, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, false, core.BuiltInFunctionMECTUnPause, 0, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTUnPause, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTRolesFunc(b.marshaller, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetMECTRole, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTLocalBurnFunc(b.gasConfig.BuiltInCost.MECTLocalBurn, b.marshaller, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTLocalBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTLocalMintFunc(b.gasConfig.BuiltInCost.MECTLocalMint, b.marshaller, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTLocalMint, newFunc)
	if err != nil {
		return err
	}

	args := ArgsNewMECTDataStorage{
		Accounts:                        b.accounts,
		GlobalSettingsHandler:           globalSettingsFunc,
		Marshalizer:                     b.marshaller,
		SaveToSystemEnableEpoch:         b.saveNFTToSystemAccountEnableEpoch,
		EpochNotifier:                   b.epochNotifier,
		ShardCoordinator:                b.shardCoordinator,
		SendAlwaysEnableEpoch:           b.sendMECTMetadataAlwaysEnableEpoch,
		FixOldTokenLiquidityEnableEpoch: b.fixOldTokenLiquidityEnableEpoch,
	}
	b.mectStorageHandler, err = NewMECTDataStorage(args)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.MECTNFTAddQuantity, b.mectStorageHandler, globalSettingsFunc, setRoleFunc, b.saveNFTToSystemAccountEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTAddQuantity, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTBurnFunc(b.gasConfig.BuiltInCost.MECTNFTBurn, b.mectStorageHandler, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTCreateFunc(b.gasConfig.BuiltInCost.MECTNFTCreate, b.gasConfig.BaseOperationCost, b.marshaller, globalSettingsFunc, setRoleFunc, b.mectStorageHandler, b.accounts, b.saveNFTToSystemAccountEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTCreate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTTransferFunc(
		b.gasConfig.BuiltInCost.MECTNFTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		setRoleFunc,
		b.mectTransferToMetaEnableEpoch,
		b.saveNFTToSystemAccountEnableEpoch,
		b.checkCorrectTokenIDEnableEpoch,
		b.mectStorageHandler,
		b.epochNotifier,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTCreateRoleTransfer(b.marshaller, b.accounts, b.shardCoordinator)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.MECTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.mectStorageHandler, globalSettingsFunc, setRoleFunc, b.mectNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTUpdateAttributes, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTAddUriFunc(b.gasConfig.BuiltInCost.MECTNFTAddURI, b.gasConfig.BaseOperationCost, b.mectStorageHandler, globalSettingsFunc, setRoleFunc, b.mectNFTImprovementV1ActivationEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTNFTAddURI, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTNFTMultiTransferFunc(
		b.gasConfig.BuiltInCost.MECTNFTMultiTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		b.mectNFTImprovementV1ActivationEpoch,
		b.epochNotifier,
		setRoleFunc,
		b.mectTransferToMetaEnableEpoch,
		b.checkCorrectTokenIDEnableEpoch,
		b.mectStorageHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMultiMECTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, true, core.BuiltInFunctionMECTSetLimitedTransfer, b.mectTransferRoleEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, false, core.BuiltInFunctionMECTUnSetLimitedTransfer, b.mectTransferRoleEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMECTUnSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc := ArgsNewMECTDeleteMetadata{
		FuncGasCost:     b.gasConfig.BuiltInCost.MECTNFTBurn,
		Marshalizer:     b.marshaller,
		Accounts:        b.accounts,
		ActivationEpoch: b.sendMECTMetadataAlwaysEnableEpoch,
		EpochNotifier:   b.epochNotifier,
		AllowedAddress:  b.configAddress,
		Delete:          true,
	}
	newFunc, err = NewMECTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.MECTDeleteMetadata, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc.Delete = false
	newFunc, err = NewMECTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.MECTAddMetadata, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, true, vmcommon.BuiltInFunctionMECTSetBurnRoleForAll, b.sendMECTMetadataAlwaysEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionMECTSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTGlobalSettingsFunc(b.accounts, b.marshaller, false, vmcommon.BuiltInFunctionMECTUnSetBurnRoleForAll, b.sendMECTMetadataAlwaysEnableEpoch, b.epochNotifier)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionMECTUnSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTTransferRoleAddressFunc(b.accounts, b.marshaller, b.sendMECTMetadataAlwaysEnableEpoch, b.epochNotifier, b.maxNumOfAddressesForTransferRole, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionMECTTransferRoleDeleteAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMECTTransferRoleAddressFunc(b.accounts, b.marshaller, b.sendMECTMetadataAlwaysEnableEpoch, b.epochNotifier, b.maxNumOfAddressesForTransferRole, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionMECTTransferRoleAddAddress, newFunc)
	if err != nil {
		return err
	}

	return nil
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCostString], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCostString], builtInOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return nil, err
	}

	gasCost := vmcommon.GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return &gasCost, nil
}

// SetPayableHandler sets the payableCheck interface to the needed functions
func (b *builtInFuncCreator) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	payableChecker, err := NewPayableCheckFunc(
		payableHandler,
		b.checkFunctionArgumentEnableEpoch,
		b.fixAsnycCallbackCheckEnableEpoch,
		b.epochNotifier,
	)
	if err != nil {
		return err
	}

	listOfTransferFunc := []string{
		core.BuiltInFunctionMultiMECTNFTTransfer,
		core.BuiltInFunctionMECTNFTTransfer,
		core.BuiltInFunctionMECTTransfer}

	for _, transferFunc := range listOfTransferFunc {
		builtInFunc, err := b.builtInFunctions.Get(transferFunc)
		if err != nil {
			return err
		}

		mectTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableChecker)
		if !ok {
			return ErrWrongTypeAssertion
		}

		err = mectTransferFunc.SetPayableChecker(payableChecker)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (b *builtInFuncCreator) IsInterfaceNil() bool {
	return b == nil
}
