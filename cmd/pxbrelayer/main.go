package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lcnem/proximax-pegzone/app"
	"github.com/lcnem/proximax-pegzone/cmd/pxbrelayer/relayer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/types"
	amino "github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/client/rpc"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/cli"
	tmLog "github.com/tendermint/tendermint/libs/log"
)

var appCodec *amino.Codec

const FlagRPCURL = "rpc-url"

func init() {

	// Read in the configuration file for the sdk
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
	config.Seal()

	appCodec = app.MakeCodec()

	DefaultCLIHome := os.ExpandEnv("$HOME/.pxbcli")

	// Add --chain-id to persistent flags and mark it required
	rootCmd.PersistentFlags().String(flags.FlagChainID, "PXB", "Chain ID of tendermint node")
	rootCmd.PersistentFlags().String(FlagRPCURL, "PXB", "RPC URL of tendermint node")
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		return initConfig(rootCmd)
	}

	// Construct Initialization Commands
	initCmd.AddCommand(
		proximaxRelayerCmd(),
		flags.LineBreak,
		cosmosRelayerCmd(),
	)

	// Construct Root Command
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		initCmd,
	)

	executor := cli.PrepareMainCmd(rootCmd, "PXB", DefaultCLIHome)
	err := executor.Execute()
	if err != nil {
		log.Fatal("failed executing CLI command", err)
	}
}

var rootCmd = &cobra.Command{
	Use:          "pxbrelayer",
	Short:        "Relayer service which listens for and relays ProximaX network events",
	SilenceUsage: true,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialization subcommands",
}

func proximaxRelayerCmd() *cobra.Command {
	ethereumRelayerCmd := &cobra.Command{
		Use:     "proximax init [validator_from_name] [tendermint_node] [proximax_node] [proximax_private_key] [proximax_multisig_address] --chain-id [chain-id]",
		Short:   "Initializes a web socket which streams live events from the ProximaX network and relays them to the Cosmos network",
		Args:    cobra.ExactArgs(5),
		Example: "pxbrelayer init proximax validator http://localhost:7475 http://bctestnet1.brimstone.xpxsirius.io:3000 3A700AE4105431BB0F24440AAA0CD08E1FD0D87D60EB805278F7DB3EA7C7D62D VBK6ZOASJSJOFUOX7XUHHZVCBO4Q11GCF726AKHG --chain-id=testing",
		RunE:    RunProximaxRelayerCmd,
	}

	return ethereumRelayerCmd
}

func cosmosRelayerCmd() *cobra.Command {
	cosmosRelayerCmd := &cobra.Command{
		Use:     "cosmos init [tendermint_node] [proximax_node] [validator_moniker] [proximax_cosigner_private_key] [multisig_account_public_key] --chain-id [chain-id]",
		Short:   "Initializes a web socket which streams live events from the Cosmos network and relays them to the ProximaX network",
		Args:    cobra.ExactArgs(5),
		Example: "pxbrelayer init cosmos tcp://localhost:26657 http://localhost:7545 --chain-id=testing",
		RunE:    RunCosmosRelayerCmd,
	}

	return cosmosRelayerCmd
}

// RunProximaxRelayerCmd executes the initProximaxRelayerCmd with the provided parameters
func RunProximaxRelayerCmd(cmd *cobra.Command, args []string) error {
	chainID := viper.GetString(flags.FlagChainID)
	if strings.TrimSpace(chainID) == "" {
		return errors.New("Must specify a 'chain-id'")
	}

	inBuf := bufio.NewReader(cmd.InOrStdin())
	// Get the validator's name and account address using their moniker
	validatorAccAddress, validatorName, err := sdkContext.GetFromFields(inBuf, args[0], false)
	if err != nil {
		return err
	}
	validatorAddress := sdk.ValAddress(validatorAccAddress)

	tendermintNode := args[1]
	if len(strings.Trim(tendermintNode, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [tendermint_node]: %s", tendermintNode))
	}

	proximaxNode := args[2]
	if len(strings.Trim(proximaxNode, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [proximax_node]: %s", proximaxNode))
	}

	proximaxCosignerPrivateKey := args[3]
	if len(strings.Trim(proximaxCosignerPrivateKey, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [proximax_private_key]: %s", proximaxCosignerPrivateKey))
	}

	proximaxMultisigPublicKey := args[4]
	if len(strings.Trim(proximaxMultisigPublicKey, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [proximax_multisig_address]: %s", proximaxMultisigPublicKey))
	}

	// Set up our CLIContext
	cliCtx := sdkContext.NewCLIContext().
		WithCodec(appCodec).
		WithFromAddress(sdk.AccAddress(validatorAddress)).
		WithFromName(validatorName)

	logger := tmLog.NewTMLogger(tmLog.NewSyncWriter(os.Stdout))

	return relayer.InitProximaXRelayer(appCodec, cliCtx, logger, tendermintNode, chainID, proximaxNode, proximaxCosignerPrivateKey, proximaxMultisigPublicKey, validatorAddress, validatorName)
}

// RunCosmosRelayerCmd executes the initCosmosRelayerCmd with the provided parameters
func RunCosmosRelayerCmd(cmd *cobra.Command, args []string) error {
	chainID := viper.GetString(flags.FlagChainID)
	if strings.TrimSpace(chainID) == "" {
		return errors.New("Must specify a 'chain-id'")
	}

	rpcURL := viper.GetString(FlagRPCURL)
	if rpcURL != "" {
		_, err := url.Parse(rpcURL)
		if rpcURL != "" && err != nil {
			return errors.New(fmt.Sprintf("invalid RPC URL: %v", rpcURL))
		}
	}

	tendermintNode := args[0]
	if len(strings.Trim(tendermintNode, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [tendermint-node]: %s", tendermintNode))
	}

	proximaXNode := args[1]
	if len(strings.Trim(proximaXNode, "")) > 0 {
		_, err := url.Parse(proximaXNode)
		if proximaXNode != "" && err != nil {
			return errors.New(fmt.Sprintf("invalid ProximaX URL: %v", proximaXNode))
		}
	}

	validatorMoniker := args[2]
	if len(strings.Trim(validatorMoniker, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [validator-moniker]: %s", validatorMoniker))
	}

	cosignerPrivateKey := args[3]
	if len(strings.Trim(cosignerPrivateKey, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [proximax_cosigner_private_key]: %s", cosignerPrivateKey))
	}

	multisigPubicKey := args[4]
	if len(strings.Trim(multisigPubicKey, "")) == 0 {
		return errors.New(fmt.Sprintf("invalid [multisig_account_public_key]: %s", multisigPubicKey))
	}

	inBuf := bufio.NewReader(cmd.InOrStdin())
	validatorAddress, _, err := loadValidatorCredentials(validatorMoniker, inBuf)
	if err != nil {
		return errors.New(fmt.Sprintf("faild to get validator address: %s", validatorMoniker))
	}

	logger := tmLog.NewTMLogger(tmLog.NewSyncWriter(os.Stdout))

	cosmosSub := relayer.NewCosmosSub(rpcURL, appCodec, validatorMoniker, validatorAddress, chainID, tendermintNode, proximaXNode, cosignerPrivateKey, multisigPubicKey, logger)

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)

	go cosmosSub.Start(exitSignal)
	<-exitSignal

	return nil
}

func initConfig(cmd *cobra.Command) error {
	return viper.BindPFlag(flags.FlagChainID, cmd.PersistentFlags().Lookup(flags.FlagChainID))
}

func loadValidatorCredentials(validatorFrom string, inBuf io.Reader) (sdk.ValAddress, string, error) {
	validatorAccAddress, validatorName, err := sdkContext.GetFromFields(inBuf, validatorFrom, false)
	if err != nil {
		return sdk.ValAddress{}, "", err
	}
	validatorAddress := sdk.ValAddress(validatorAccAddress)

	_, err = authtxb.MakeSignature(nil, validatorName, keys.DefaultKeyPass, authtxb.StdSignMsg{})
	if err != nil {
		return sdk.ValAddress{}, "", err
	}

	return validatorAddress, validatorName, nil
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
