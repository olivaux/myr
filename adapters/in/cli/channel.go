// adapters/in/cli/channel.go — commandes Cobra pour les canaux
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Query blockchain channels",
	Long: `Commands to query the blockchain channels available on this instance.

A channel is an isolated blockchain sub-network on which 3D models are
published. Each channel has its own membership and access-control configuration.`,
}

var channelGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a channel",
	Long: `Display the identifier and name of a channel.

Example:
  myr channel get greenchannel`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ch, err := channelSvc.Get(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("ID  : %s\nNom : %s\n", ch.ID, ch.Name)
		return nil
	},
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels",
	Long: `List all channels known to this Myr instance.

Example:
  myr channel list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		channels, err := channelSvc.List()
		if err != nil {
			return err
		}
		for _, ch := range channels {
			fmt.Printf("%-20s  %s\n", ch.ID, ch.Name)
		}
		return nil
	},
}

func init() {
	channelCmd.AddCommand(channelGetCmd, channelListCmd)
}
