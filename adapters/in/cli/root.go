// adapters/in/cli/root.go — point d'entrée Cobra, injection des services
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"myr/domain/channel"
	"myr/domain/identity"
	"myr/domain/model"
	"myr/domain/network"
	"myr/domain/payment"
	"myr/domain/role"
)

var (
	modelSvc    model.ModelService
	channelSvc  channel.ChannelService
	paymentSvc  payment.PaymentService
	networkSvc  network.NetworkService
	roleSvc     role.RoleService
	identitySvc identity.IdentityService
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
	rootCmd.AddCommand(modelCmd, channelCmd, paymentCmd, networkCmd, orgCmd, nodeCmd, roleCmd, identityCmd, manCmd)
	return rootCmd
}

// Execute injects services and starts the CLI.
func Execute(ms model.ModelService, cs channel.ChannelService, ps payment.PaymentService, ns network.NetworkService, rs role.RoleService, is identity.IdentityService) {
	modelSvc = ms
	channelSvc = cs
	paymentSvc = ps
	networkSvc = ns
	roleSvc = rs
	identitySvc = is

	rootCmd.AddCommand(modelCmd, channelCmd, paymentCmd, networkCmd, orgCmd, nodeCmd, roleCmd, identityCmd, manCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
