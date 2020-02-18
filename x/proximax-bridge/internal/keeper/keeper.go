package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/peggy/x/oracle"
	"github.com/lcnem/proximax-pegzone/x/proximax-bridge/internal/types"
)

// Keeper of the proximax-bridge store
type Keeper struct {
	storeKey     sdk.StoreKey
	cdc          *codec.Codec
	supplyKeeper types.SupplyKeeper
	oracleKeeper types.OracleKeeper
}

// NewKeeper creates a proximax-bridge keeper
func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, supplyKeeper types.SupplyKeeper, oracleKeeper types.OracleKeeper) Keeper {
	keeper := Keeper{
		storeKey:     key,
		cdc:          cdc,
		supplyKeeper: supplyKeeper,
		oracleKeeper: oracleKeeper,
	}
	return keeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ProcessClaim processes a new claim coming in from a validator
func (k Keeper) ProcessClaim(ctx sdk.Context, claim types.EthBridgeClaim) (oracle.Status, error) {
	oracleClaim, err := types.CreateOracleClaimFromEthClaim(k.cdc, claim)
	if err != nil {
		return oracle.Status{}, err
	}

	return k.oracleKeeper.ProcessClaim(ctx, oracleClaim)
}

// ProcessSuccessfulClaim processes a claim that has just completed successfully with consensus
func (k Keeper) ProcessSuccessfulClaim(ctx sdk.Context, claim string) error {
	oracleClaim, err := types.CreateOracleClaimFromOracleString(claim)
	if err != nil {
		return err
	}

	receiverAddress := oracleClaim.CosmosReceiver

	switch oracleClaim.ClaimType {
	case types.LockText:
		err = k.supplyKeeper.MintCoins(ctx, types.ModuleName, oracleClaim.Amount)
	default:
		err = types.ErrInvalidClaimType
	}

	if err != nil {
		return err
	}

	if err := k.supplyKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, receiverAddress, oracleClaim.Amount,
	); err != nil {
		panic(err)
	}

	return nil
}

// ProcessBurn processes the burn of bridged coins from the given sender
func (k Keeper) ProcessBurn(ctx sdk.Context, cosmosSender sdk.AccAddress, amount sdk.Coins) error {
	if err := k.supplyKeeper.SendCoinsFromAccountToModule(
		ctx, cosmosSender, types.ModuleName, amount,
	); err != nil {
		return err
	}

	if err := k.supplyKeeper.BurnCoins(ctx, types.ModuleName, amount); err != nil {
		panic(err)
	}

	return nil
}

// ProcessLock processes the lockup of cosmos coins from the given sender
func (k Keeper) ProcessLock(ctx sdk.Context, cosmosSender sdk.AccAddress, amount sdk.Coins) error {
	return k.supplyKeeper.SendCoinsFromAccountToModule(ctx, cosmosSender, types.ModuleName, amount)
}
