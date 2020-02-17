package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// TODO: Fill out some custom errors for the module
// You can see how they are constructed below:
var (
	ErrInvalidMainchainTxHash  = sdkerrors.Register(ModuleName, 1, "invalid mainchain tx hash")
	ErrInvalidMainchainAddress = sdkerrors.Register(ModuleName, 2, "invalid mainchain address")
)
