// adapters/in/cli/model.go — commandes Cobra pour les modeles 3D
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage 3D models",
	Long: `Commands to publish, inspect and verify 3D models on the Myr platform.

Models are stored on the active blockchain network. Each model has a
unique ID and belongs to a channel.`,
}

var modelAddCmd = &cobra.Command{
	Use:   "add <file>",
	Short: "Publish a 3D model to the blockchain",
	Long: `Publish a 3D file on the blockchain of the specified channel.

The format is designer-agnostic: .stl, .step, .obj, .3mf and any other
format are accepted. The file is hashed (SHA-256), stored on the blockchain
network and indexed with the provided name and tags.

Examples:
  myr model add wheel.stl  --name "Front wheel"  --channel greenchannel --tags "3dprint"
  myr model add pivot.step --name "Left pivot"   --channel greenchannel --tags "mechanical"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		channelID, _ := cmd.Flags().GetString("channel")
		tagsStr, _ := cmd.Flags().GetString("tags")

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		m, err := modelSvc.Add(args[0], name, channelID, "", tags)
		if err != nil {
			return err
		}
		fmt.Printf("Modele ajoute : %s  id=%s\n", m.Name, m.ID)
		return nil
	},
}

var modelGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a model",
	Long: `Display the metadata of a model by its blockchain identifier.

Example:
  myr model get abc123def456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := modelSvc.Get(args[0], "")
		if err != nil {
			return err
		}
		fmt.Printf("ID      : %s\nNom     : %s\nCanal   : %s\nOwner   : %s\nTags    : %s\n",
			m.ID, m.Name, m.ChannelID, m.OwnerID, strings.Join(m.Tags, ", "))
		return nil
	},
}

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List models in a channel",
	Long: `List all registered models, with optional filtering by channel.

Examples:
  myr model list
  myr model list --channel greenchannel`,
	RunE: func(cmd *cobra.Command, args []string) error {
		channelID, _ := cmd.Flags().GetString("channel")
		models, err := modelSvc.List(channelID)
		if err != nil {
			return err
		}
		for _, m := range models {
			fmt.Printf("%s  %-30s  [%s]\n", m.ID, m.Name, strings.Join(m.Tags, ", "))
		}
		return nil
	},
}

var modelVerifyCmd = &cobra.Command{
	Use:   "verify <id>",
	Short: "Verify the integrity of a model",
	Long: `Recompute the hash of the file associated with the model and compare it
to the hash recorded on the blockchain. Returns OK if the file has not
been altered, KO if a divergence is detected.

Example:
  myr model verify abc123def456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := modelSvc.Verify(args[0], "")
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("OK  modele %s : integrite verifiee\n", args[0])
		} else {
			fmt.Printf("KO  modele %s : integrite compromise\n", args[0])
		}
		return nil
	},
}

func init() {
	modelAddCmd.Flags().String("name", "", "Nom du modele (requis)")
	modelAddCmd.Flags().String("channel", "", "ID du canal")
	modelAddCmd.Flags().String("tags", "", "Tags separes par des virgules")
	modelAddCmd.MarkFlagRequired("name")

	modelListCmd.Flags().String("channel", "", "ID du canal")

	modelCmd.AddCommand(modelAddCmd, modelGetCmd, modelListCmd, modelVerifyCmd)
}
