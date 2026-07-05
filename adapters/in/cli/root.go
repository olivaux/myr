// adapters/in/cli/root.go — point d'entrée Cobra, injection des services
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"myr/domain/channel"
	"myr/domain/model"
	"myr/domain/network"
	"myr/domain/payment"
)

var (
	modelSvc   model.ModelService
	channelSvc channel.ChannelService
	paymentSvc payment.PaymentService
	networkSvc network.NetworkService
)

var rootCmd = &cobra.Command{
	Use:   "myr",
	Short: "Myr — 3D model platform on blockchain",
	Long: `Myr is a command-line tool to manage 3D models on a blockchain
network.

Publish, inspect and verify 3D assets, manage blockchain channels, process
payments and administer network nodes.

When no blockchain network is reachable, Myr operates in local JSON
simulation mode.

Run 'myr man [command]' to read the full manual for any command.`,
}

// CommandTree returns rootCmd with all subcommands registered.
// Used by man page generation (no services required).
func CommandTree() *cobra.Command {
	rootCmd.AddCommand(modelCmd, channelCmd, paymentCmd, networkCmd, orgCmd, nodeCmd, manCmd)
	return rootCmd
}

// Execute injects services and starts the CLI.
func Execute(ms model.ModelService, cs channel.ChannelService, ps payment.PaymentService, ns network.NetworkService) {
	modelSvc = ms
	channelSvc = cs
	paymentSvc = ps
	networkSvc = ns

	rootCmd.AddCommand(modelCmd, channelCmd, paymentCmd, networkCmd, orgCmd, nodeCmd, manCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
