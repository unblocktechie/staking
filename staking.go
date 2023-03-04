package staking

import (
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/helper/keccak"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/0xPolygon/polygon-edge/validators"
)

var (
	MinValidatorCount = uint64(1)
	MaxValidatorCount = common.MaxSafeJSInt
)

// getAddressMapping returns the key for the SC storage mapping (address => something)
//
// More information:
// https://docs.soliditylang.org/en/latest/internals/layout_in_storage.html
func getAddressMapping(address types.Address, slot int64) []byte {
	bigSlot := big.NewInt(slot)

	finalSlice := append(
		common.PadLeftOrTrim(address.Bytes(), 32),
		common.PadLeftOrTrim(bigSlot.Bytes(), 32)...,
	)

	return keccak.Keccak256(nil, finalSlice)
}

// getIndexWithOffset is a helper method for adding an offset to the already found keccak hash
func getIndexWithOffset(keccakHash []byte, offset uint64) []byte {
	bigOffset := big.NewInt(int64(offset))
	bigKeccak := big.NewInt(0).SetBytes(keccakHash)

	bigKeccak.Add(bigKeccak, bigOffset)

	return bigKeccak.Bytes()
}

// getStorageIndexes is a helper function for getting the correct indexes
// of the storage slots which need to be modified during bootstrap.
//
// It is SC dependant, and based on the SC located at:
// https://github.com/0xPolygon/staking-contracts/
func getStorageIndexes(validator validators.Validator, index int) *StorageIndexes {
	storageIndexes := &StorageIndexes{}
	address := validator.Addr()

	// Get the indexes for the mappings
	// The index for the mapping is retrieved with:
	// keccak(address . slot)
	// . stands for concatenation (basically appending the bytes)
	storageIndexes.AddressToIsValidatorIndex = getAddressMapping(
		address,
		addressToIsValidatorSlot,
	)

	storageIndexes.AddressToStakedAmountIndex = getAddressMapping(
		address,
		addressToStakedAmountSlot,
	)

	storageIndexes.AddressToValidatorIndexIndex = getAddressMapping(
		address,
		addressToValidatorIndexSlot,
	)

	storageIndexes.ValidatorBLSPublicKeyIndex = getAddressMapping(
		address,
		addressToBLSPublicKeySlot,
	)

	// Index for array types is calculated as keccak(slot) + index
	// The slot for the dynamic arrays that's put in the keccak needs to be in hex form (padded 64 chars)
	storageIndexes.ValidatorsIndex = getIndexWithOffset(
		keccak.Keccak256(nil, common.PadLeftOrTrim(big.NewInt(validatorsSlot).Bytes(), 32)),
		uint64(index),
	)

	return storageIndexes
}

// setBytesToStorage sets bytes data into storage map from specified base index
func setBytesToStorage(
	storageMap map[types.Hash]types.Hash,
	baseIndexBytes []byte,
	data []byte,
) {
	dataLen := len(data)
	baseIndex := types.BytesToHash(baseIndexBytes)

	if dataLen <= 31 {
		bytes := types.Hash{}

		copy(bytes[:len(data)], data)

		// Set 2*Size at the first byte
		bytes[len(bytes)-1] = byte(dataLen * 2)

		storageMap[baseIndex] = bytes

		return
	}

	// Set size at the base index
	baseSlot := types.Hash{}
	baseSlot[31] = byte(2*dataLen + 1)
	storageMap[baseIndex] = baseSlot

	zeroIndex := keccak.Keccak256(nil, baseIndexBytes)
	numBytesInSlot := 256 / 8

	for i := 0; i < dataLen; i++ {
		offset := i / numBytesInSlot

		slotIndex := types.BytesToHash(getIndexWithOffset(zeroIndex, uint64(offset)))
		byteIndex := i % numBytesInSlot

		slot := storageMap[slotIndex]
		slot[byteIndex] = data[i]

		storageMap[slotIndex] = slot
	}
}

// PredeployParams contains the values used to predeploy the PoS staking contract
type PredeployParams struct {
	MinValidatorCount uint64
	MaxValidatorCount uint64
}

// StorageIndexes is a wrapper for different storage indexes that
// need to be modified
type StorageIndexes struct {
	ValidatorsIndex              []byte // []address
	ValidatorBLSPublicKeyIndex   []byte // mapping(address => byte[])
	AddressToIsValidatorIndex    []byte // mapping(address => bool)
	AddressToStakedAmountIndex   []byte // mapping(address => uint256)
	AddressToValidatorIndexIndex []byte // mapping(address => uint256)
}

// Slot definitions for SC storage
var (
	validatorsSlot              = int64(0) // Slot 0
	addressToIsValidatorSlot    = int64(1) // Slot 1
	addressToStakedAmountSlot   = int64(2) // Slot 2
	addressToValidatorIndexSlot = int64(3) // Slot 3
	stakedAmountSlot            = int64(4) // Slot 4
	minNumValidatorSlot         = int64(5) // Slot 5
	maxNumValidatorSlot         = int64(6) // Slot 6
	addressToBLSPublicKeySlot   = int64(7) // Slot 7
)

const (
	DefaultStakedBalance = "0x8AC7230489E80000" // 10 ETH
	//nolint: lll
	StakingSCBytecode = "0x6080604052600436106101235760003560e01c80637a6eea37116100a0578063d94c111b11610064578063d94c111b14610440578063e387a7ed14610469578063e804fbf614610494578063f90ecacc146104bf578063facd743b146104fc57610191565b80637a6eea37146103575780637dceceb814610382578063af6da36e146103bf578063c795c077146103ea578063ca1e78191461041557610191565b8063373d6132116100e7578063373d61321461028f5780633a4b66f1146102ba5780633c561f04146102c457806351a9ab32146102ef578063714ff4251461032c57610191565b806302b7519914610196578063065ae171146101d35780632367f6b5146102105780632def66201461024d57806332e43a111461026457610191565b36610191576101473373ffffffffffffffffffffffffffffffffffffffff16610539565b15610187576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161017e90611835565b60405180910390fd5b61018f61054c565b005b600080fd5b3480156101a257600080fd5b506101bd60048036038101906101b8919061142b565b610623565b6040516101ca9190611890565b60405180910390f35b3480156101df57600080fd5b506101fa60048036038101906101f5919061142b565b61063b565b6040516102079190611798565b60405180910390f35b34801561021c57600080fd5b506102376004803603810190610232919061142b565b61065b565b6040516102449190611890565b60405180910390f35b34801561025957600080fd5b506102626106a4565b005b34801561027057600080fd5b5061027961078f565b6040516102869190611739565b60405180910390f35b34801561029b57600080fd5b506102a46107b3565b6040516102b19190611890565b60405180910390f35b6102c26107bd565b005b3480156102d057600080fd5b506102d9610826565b6040516102e69190611776565b60405180910390f35b3480156102fb57600080fd5b506103166004803603810190610311919061142b565b6109ce565b60405161032391906117b3565b60405180910390f35b34801561033857600080fd5b50610341610a6e565b60405161034e9190611890565b60405180910390f35b34801561036357600080fd5b5061036c610a78565b6040516103799190611875565b60405180910390f35b34801561038e57600080fd5b506103a960048036038101906103a4919061142b565b610a84565b6040516103b69190611890565b60405180910390f35b3480156103cb57600080fd5b506103d4610a9c565b6040516103e19190611890565b60405180910390f35b3480156103f657600080fd5b506103ff610aa2565b60405161040c9190611890565b60405180910390f35b34801561042157600080fd5b5061042a610aa8565b6040516104379190611754565b60405180910390f35b34801561044c57600080fd5b5061046760048036038101906104629190611458565b610b36565b005b34801561047557600080fd5b5061047e610bdb565b60405161048b9190611890565b60405180910390f35b3480156104a057600080fd5b506104a9610be1565b6040516104b69190611890565b60405180910390f35b3480156104cb57600080fd5b506104e660048036038101906104e191906114a1565b610beb565b6040516104f39190611739565b60405180910390f35b34801561050857600080fd5b50610523600480360381019061051e919061142b565b610c2a565b6040516105309190611798565b60405180910390f35b600080823b905060008111915050919050565b346005600082825461055e91906119b1565b9250508190555034600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008282546105b491906119b1565b925050819055506105c433610c80565b156105d3576105d233610cf8565b5b3373ffffffffffffffffffffffffffffffffffffffff167f9e71bc8eea02a63969f509818f2dafb9254532904319f9dbda79b67bd34a5f3d346040516106199190611890565b60405180910390a2565b60046020528060005260406000206000915090505481565b60026020528060005260406000206000915054906101000a900460ff1681565b6000600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6106c33373ffffffffffffffffffffffffffffffffffffffff16610539565b15610703576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106fa90611835565b60405180910390fd5b6000600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205411610785576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161077c906117d5565b60405180910390fd5b61078d610e48565b565b60008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600554905090565b6107dc3373ffffffffffffffffffffffffffffffffffffffff16610539565b1561081c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161081390611835565b60405180910390fd5b61082461054c565b565b6060600060018054905067ffffffffffffffff81111561084957610848611c49565b5b60405190808252806020026020018201604052801561087c57816020015b60608152602001906001900390816108675790505b50905060005b6001805490508110156109c65760086000600183815481106108a7576108a6611c1a565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020805461091790611ae1565b80601f016020809104026020016040519081016040528092919081815260200182805461094390611ae1565b80156109905780601f1061096557610100808354040283529160200191610990565b820191906000526020600020905b81548152906001019060200180831161097357829003601f168201915b50505050508282815181106109a8576109a7611c1a565b5b602002602001018190525080806109be90611b44565b915050610882565b508091505090565b600860205280600052604060002060009150905080546109ed90611ae1565b80601f0160208091040260200160405190810160405280929190818152602001828054610a1990611ae1565b8015610a665780601f10610a3b57610100808354040283529160200191610a66565b820191906000526020600020905b815481529060010190602001808311610a4957829003601f168201915b505050505081565b6000600654905090565b678ac7230489e8000081565b60036020528060005260406000206000915090505481565b60075481565b60065481565b60606001805480602002602001604051908101604052809291908181526020018280548015610b2c57602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610ae2575b5050505050905090565b80600860003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000209080519060200190610b899291906112ee565b503373ffffffffffffffffffffffffffffffffffffffff167f472da4d064218fa97032725fbcff922201fa643fed0765b5ffe0ceef63d7b3dc82604051610bd091906117b3565b60405180910390a250565b60055481565b6000600754905090565b60018181548110610bfb57600080fd5b906000526020600020016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b6000610c8b82610f9a565b158015610cf15750678ac7230489e800006fffffffffffffffffffffffffffffffff16600360008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410155b9050919050565b60075460018054905010610d41576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d38906117f5565b60405180910390fd5b6001600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff021916908315150217905550600180549050600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506001819080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6000600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490506000600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508060056000828254610ee39190611a07565b92505081905550610ef333610f9a565b15610f0257610f0133610ff0565b5b3373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f19350505050158015610f48573d6000803e3d6000fd5b503373ffffffffffffffffffffffffffffffffffffffff167f0f5bb82176feb1b5e747e28471aa92156a04d9f3ab9f45f28e2d704232b93f7582604051610f8f9190611890565b60405180910390a250565b6000600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b60065460018054905011611039576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161103090611855565b60405180910390fd5b600180549050600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054106110bf576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016110b690611815565b60405180910390fd5b6000600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205490506000600180805490506111169190611a07565b90508082146112055760006001828154811061113557611134611c1a565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050806001848154811061117757611176611c1a565b5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555082600460008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550505b6000600260008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055506000600460008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555060018054806112b4576112b3611beb565b5b6001900381819060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690559055505050565b8280546112fa90611ae1565b90600052602060002090601f01602090048101928261131c5760008555611363565b82601f1061133557805160ff1916838001178555611363565b82800160010185558215611363579182015b82811115611362578251825591602001919060010190611347565b5b5090506113709190611374565b5090565b5b8082111561138d576000816000905550600101611375565b5090565b60006113a461139f846118d0565b6118ab565b9050828152602081018484840111156113c0576113bf611c7d565b5b6113cb848285611a9f565b509392505050565b6000813590506113e281611db6565b92915050565b600082601f8301126113fd576113fc611c78565b5b813561140d848260208601611391565b91505092915050565b60008135905061142581611dcd565b92915050565b60006020828403121561144157611440611c87565b5b600061144f848285016113d3565b91505092915050565b60006020828403121561146e5761146d611c87565b5b600082013567ffffffffffffffff81111561148c5761148b611c82565b5b611498848285016113e8565b91505092915050565b6000602082840312156114b7576114b6611c87565b5b60006114c584828501611416565b91505092915050565b60006114da83836114fa565b60208301905092915050565b60006114f283836115fa565b905092915050565b61150381611a3b565b82525050565b61151281611a3b565b82525050565b600061152382611921565b61152d818561195c565b935061153883611901565b8060005b8381101561156957815161155088826114ce565b975061155b83611942565b92505060018101905061153c565b5085935050505092915050565b60006115818261192c565b61158b818561196d565b93508360208202850161159d85611911565b8060005b858110156115d957848403895281516115ba85826114e6565b94506115c58361194f565b925060208a019950506001810190506115a1565b50829750879550505050505092915050565b6115f481611a4d565b82525050565b600061160582611937565b61160f818561197e565b935061161f818560208601611aae565b61162881611c8c565b840191505092915050565b600061163e82611937565b611648818561198f565b9350611658818560208601611aae565b61166181611c8c565b840191505092915050565b6000611679601d836119a0565b915061168482611c9d565b602082019050919050565b600061169c6027836119a0565b91506116a782611cc6565b604082019050919050565b60006116bf6012836119a0565b91506116ca82611d15565b602082019050919050565b60006116e2601a836119a0565b91506116ed82611d3e565b602082019050919050565b60006117056040836119a0565b915061171082611d67565b604082019050919050565b61172481611a59565b82525050565b61173381611a95565b82525050565b600060208201905061174e6000830184611509565b92915050565b6000602082019050818103600083015261176e8184611518565b905092915050565b600060208201905081810360008301526117908184611576565b905092915050565b60006020820190506117ad60008301846115eb565b92915050565b600060208201905081810360008301526117cd8184611633565b905092915050565b600060208201905081810360008301526117ee8161166c565b9050919050565b6000602082019050818103600083015261180e8161168f565b9050919050565b6000602082019050818103600083015261182e816116b2565b9050919050565b6000602082019050818103600083015261184e816116d5565b9050919050565b6000602082019050818103600083015261186e816116f8565b9050919050565b600060208201905061188a600083018461171b565b92915050565b60006020820190506118a5600083018461172a565b92915050565b60006118b56118c6565b90506118c18282611b13565b919050565b6000604051905090565b600067ffffffffffffffff8211156118eb576118ea611c49565b5b6118f482611c8c565b9050602081019050919050565b6000819050602082019050919050565b6000819050602082019050919050565b600081519050919050565b600081519050919050565b600081519050919050565b6000602082019050919050565b6000602082019050919050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b600082825260208201905092915050565b60006119bc82611a95565b91506119c783611a95565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156119fc576119fb611b8d565b5b828201905092915050565b6000611a1282611a95565b9150611a1d83611a95565b925082821015611a3057611a2f611b8d565b5b828203905092915050565b6000611a4682611a75565b9050919050565b60008115159050919050565b60006fffffffffffffffffffffffffffffffff82169050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b82818337600083830152505050565b60005b83811015611acc578082015181840152602081019050611ab1565b83811115611adb576000848401525b50505050565b60006002820490506001821680611af957607f821691505b60208210811415611b0d57611b0c611bbc565b5b50919050565b611b1c82611c8c565b810181811067ffffffffffffffff82111715611b3b57611b3a611c49565b5b80604052505050565b6000611b4f82611a95565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff821415611b8257611b81611b8d565b5b600182019050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4f6e6c79207374616b65722063616e2063616c6c2066756e6374696f6e000000600082015250565b7f56616c696461746f72207365742068617320726561636865642066756c6c206360008201527f6170616369747900000000000000000000000000000000000000000000000000602082015250565b7f696e646578206f7574206f662072616e67650000000000000000000000000000600082015250565b7f4f6e6c7920454f412063616e2063616c6c2066756e6374696f6e000000000000600082015250565b7f56616c696461746f72732063616e2774206265206c657373207468616e20746860008201527f65206d696e696d756d2072657175697265642076616c696461746f72206e756d602082015250565b611dbf81611a3b565b8114611dca57600080fd5b50565b611dd681611a95565b8114611de157600080fd5b5056fea2646970667358221220c49057f5cecf8004854d139d54ce63f88afdb16f93d1102e6d26a7b081d22f5f64736f6c63430008070033"
)

// PredeployStakingSC is a helper method for setting up the staking smart contract account,
// using the passed in validators as pre-staked validators
func PredeployStakingSC(
	vals validators.Validators,
	params PredeployParams,
) (*chain.GenesisAccount, error) {
	// Set the code for the staking smart contract
	// Code retrieved from https://github.com/0xPolygon/staking-contracts
	scHex, _ := hex.DecodeHex(StakingSCBytecode)
	stakingAccount := &chain.GenesisAccount{
		Code: scHex,
	}

	// Parse the default staked balance value into *big.Int
	val := DefaultStakedBalance
	bigDefaultStakedBalance, err := types.ParseUint256orHex(&val)

	if err != nil {
		return nil, fmt.Errorf("unable to generate DefaultStatkedBalance, %w", err)
	}

	// Generate the empty account storage map
	storageMap := make(map[types.Hash]types.Hash)
	bigTrueValue := big.NewInt(1)
	stakedAmount := big.NewInt(0)
	bigMinNumValidators := big.NewInt(int64(params.MinValidatorCount))
	bigMaxNumValidators := big.NewInt(int64(params.MaxValidatorCount))
	valsLen := big.NewInt(0)

	if vals != nil {
		valsLen = big.NewInt(int64(vals.Len()))

		for idx := 0; idx < vals.Len(); idx++ {
			validator := vals.At(uint64(idx))

			// Update the total staked amount
			stakedAmount = stakedAmount.Add(stakedAmount, bigDefaultStakedBalance)

			// Get the storage indexes
			storageIndexes := getStorageIndexes(validator, idx)

			// Set the value for the validators array
			storageMap[types.BytesToHash(storageIndexes.ValidatorsIndex)] =
				types.BytesToHash(
					validator.Addr().Bytes(),
				)

			if blsValidator, ok := validator.(*validators.BLSValidator); ok {
				setBytesToStorage(
					storageMap,
					storageIndexes.ValidatorBLSPublicKeyIndex,
					blsValidator.BLSPublicKey,
				)
			}

			// Set the value for the address -> validator array index mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToIsValidatorIndex)] =
				types.BytesToHash(bigTrueValue.Bytes())

			// Set the value for the address -> staked amount mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToStakedAmountIndex)] =
				types.StringToHash(hex.EncodeBig(bigDefaultStakedBalance))

			// Set the value for the address -> validator index mapping
			storageMap[types.BytesToHash(storageIndexes.AddressToValidatorIndexIndex)] =
				types.StringToHash(hex.EncodeUint64(uint64(idx)))
		}
	}

	// Set the value for the total staked amount
	storageMap[types.BytesToHash(big.NewInt(stakedAmountSlot).Bytes())] =
		types.BytesToHash(stakedAmount.Bytes())

	// Set the value for the size of the validators array
	storageMap[types.BytesToHash(big.NewInt(validatorsSlot).Bytes())] =
		types.BytesToHash(valsLen.Bytes())

	// Set the value for the minimum number of validators
	storageMap[types.BytesToHash(big.NewInt(minNumValidatorSlot).Bytes())] =
		types.BytesToHash(bigMinNumValidators.Bytes())

	// Set the value for the maximum number of validators
	storageMap[types.BytesToHash(big.NewInt(maxNumValidatorSlot).Bytes())] =
		types.BytesToHash(bigMaxNumValidators.Bytes())

	// Save the storage map
	stakingAccount.Storage = storageMap

	// Set the Staking SC balance to numValidators * defaultStakedBalance
	stakingAccount.Balance = stakedAmount

	return stakingAccount, nil
}
