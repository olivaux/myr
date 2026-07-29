// adapters/in/cli/model_instance.go — instances de composants dans un module
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"myr-core/domain/model"
)

var modelInstanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage component instances placed inside a module",
}

func runModelInstanceAdd(w io.Writer, svc model.ModelService, moduleID, assetID string) error {
	m, err := svc.AddAssetToWorkspace(moduleID, assetID)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Instance ajoutée au module %s (total : %d instances).\n", m.ID, len(m.WorkspaceInstances))
	return nil
}

var modelInstanceAddCmd = &cobra.Command{
	Use:   "add <moduleID> <assetID>",
	Short: "Add an existing asset as an instance of a module (local draft, no blockchain effect)",
	Long: `Add an existing asset as an instance of a module. This only updates the local
draft — even if the module was already submitted before, this reopens it as a
draft; the change reaches the blockchain only at the next "myr module submit".`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInstanceAdd(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

func runModelInstanceRemove(w io.Writer, svc model.ModelService, moduleID, instanceID string) error {
	m, err := svc.RemoveAssetFromWorkspace(moduleID, instanceID)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Instance retirée du module %s (restant : %d instances).\n", m.ID, len(m.WorkspaceInstances))
	return nil
}

var modelInstanceRemoveCmd = &cobra.Command{
	Use:   "remove <moduleID> <instanceID>",
	Short: "Remove an instance from a module (cascades to its connections; local draft, no blockchain effect)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInstanceRemove(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

func init() {
	modelInstanceCmd.AddCommand(modelInstanceAddCmd, modelInstanceRemoveCmd)
	modelCmd.AddCommand(modelInstanceCmd)
}
