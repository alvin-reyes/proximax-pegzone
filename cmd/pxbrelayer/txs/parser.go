package txs

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tmKv "github.com/tendermint/tendermint/libs/kv"

	msgTypes "github.com/lcnem/proximax-pegzone/x/proximax-bridge"
)

func PegClaimEventToCosmosMsg(attributes []tmKv.Pair) (*msgTypes.MsgPegClaim, error) {
	var cosmosSender sdk.ValAddress
	var mainchainTxHash string
	var toAddress sdk.AccAddress
	var amount sdk.Coins
	var err error

	for _, attribute := range attributes {
		key := string(attribute.GetKey())
		val := string(attribute.GetValue())
		switch key {
		case "cosmos_sender":
			cosmosSender, err = sdk.ValAddressFromBech32(val)
			break
		case "mainchain_tx_hash":
			mainchainTxHash = val
			break
		case "to_address":
			toAddress, err = sdk.AccAddressFromBech32(val)
			break
		case "amount":
			amount, err = sdk.ParseCoins(val)
			break
		}
	}
	if err != nil {
		return nil, err
	}
	cosmosMsg := msgTypes.NewMsgPegClaim(cosmosSender, mainchainTxHash, toAddress, amount)
	return &cosmosMsg, nil
}

func UnpegNotCosignedClaimEventToCosmosMsg(attributes []tmKv.Pair) (*msgTypes.MsgUnpegNotCosignedClaim, error) {
	var address sdk.ValAddress
	var txHash string
	var notCosignedValidators []sdk.ValAddress
	var err error

	for _, attribute := range attributes {
		key := string(attribute.GetKey())
		val := string(attribute.GetValue())
		switch key {
		case "cosmos_sender":
			address, err = sdk.ValAddressFromBech32(val)
			break
		case "tx_hash":
			txHash = val
			break
		case "not_cosigned_validators":
			for _, addr := range strings.Split(val, ",") {
				valAddress, err := sdk.ValAddressFromBech32(addr)
				if err != nil {
					break
				}
				notCosignedValidators = append(notCosignedValidators, valAddress)
			}
			break
		}
	}
	if err != nil {
		return nil, err
	}
	cosmosMsg := msgTypes.NewMsgUnpegNotCosignedClaim(address, txHash, notCosignedValidators)
	return &cosmosMsg, nil
}

