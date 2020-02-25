package proximax_bridge

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/peggy/x/oracle"
	"github.com/lcnem/proximax-pegzone/x/proximax-bridge/internal/types"
)

// NewHandler creates an sdk.Handler for all the proximax-bridge type messages
func NewHandler(cdc *codec.Codec, accountKeeper auth.AccountKeeper, bridgeKeeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		// TODO: Define your msg cases
		//
		case MsgPegClaim:
			return handleMsgPegClaim(ctx, cdc, bridgeKeeper, msg)
		case MsgUnpeg:
			return handleMsgUnpeg(ctx, cdc, accountKeeper, bridgeKeeper, msg)
		case MsgUnpegNotCosignedClaim:
			return handleMsgUnpegNotCosignedClaim(ctx, cdc, accountKeeper, bridgeKeeper, msg)

		//Example:
		// case MsgSet<Action>:
		// 	return handleMsg<Action>(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

// Handle a message to create a bridge claim
func handleMsgPegClaim(
	ctx sdk.Context, cdc *codec.Codec, bridgeKeeper Keeper, msg MsgPegClaim,
) (*sdk.Result, error) {
	status, err := bridgeKeeper.ProcessPegClaim(ctx, msg)
	if err != nil {
		return nil, err
	}
	if status.Text == oracle.SuccessStatusText {
		if err := bridgeKeeper.ProcessSuccessfulPegClaim(ctx, status.FinalClaim); err != nil {
			return nil, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Address.String()),
		),
		sdk.NewEvent(
			types.EventTypeCreateClaim,
			sdk.NewAttribute(types.AttributeKeyMainchainTxHash, msg.MainchainTxHash),
			sdk.NewAttribute(types.AttributeKeyCosmosReceiver, msg.Address.String()),
			// sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount),
			// sdk.NewAttribute(types.AttributeKeyClaimType, msg.ClaimType.String()),
		),
		sdk.NewEvent(
			types.EventTypeProphecyStatus,
			sdk.NewAttribute(types.AttributeKeyStatus, status.Text.String()),
		),
	})

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func handleMsgUnpeg(
	ctx sdk.Context, cdc *codec.Codec, accountKeeper auth.AccountKeeper,
	bridgeKeeper Keeper, msg MsgUnpeg,
) (*sdk.Result, error) {

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Address.String()),
		),
		sdk.NewEvent(
			types.EventTypeUnpeg,
			sdk.NewAttribute(types.AttributeKeyCosmosSender, msg.Address.String()),
			sdk.NewAttribute(types.AttributeKeyMainchainReceiver, msg.MainchainAddress),
			// sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
		),
	})

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil

}

func handleMsgUnpegNotCosignedClaim(
	ctx sdk.Context, cdc *codec.Codec, accountKeeper auth.AccountKeeper,
	bridgeKeeper Keeper, msg MsgUnpegNotCosignedClaim,
) (*sdk.Result, error) {
	status, err := bridgeKeeper.ProcessUnpegNotCosignedClaim(ctx, msg)
	if err != nil {
		return nil, err
	}
	if status.Text == oracle.SuccessStatusText {
		if err := bridgeKeeper.ProcessSuccessfulUnpegNotCosignedClaim(ctx, status.FinalClaim); err != nil {
			return nil, err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Address.String()),
		),
	})

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil

}
