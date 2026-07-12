// adapters/in/cli/module.go — commandes Cobra pour les modules (assemblages nommés)
package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr/domain/model"
)

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Create and manage modules (named, versioned assemblies)",
	Long: `Commands to create and manage modules — components composed of other
components (or modules) connected together.

A module always starts in "draft" state, is built by adding instances and
assembly links, and requires at least one assembly before it can be
submitted to the blockchain.

Examples:
  myr module create --name "Chassis" --channel greenchannel
  myr module submit mod-123 --note "v1"`,
}

// ── create ────────────────────────────────────────────────────────────────────

func runModuleCreate(w io.Writer, svc model.ModelService, req model.ModuleRequest) error {
	m, err := svc.CreateModule(req)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Module créé (draft) : %s  id=%s\n", m.Name, m.ID)
	return nil
}

var moduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new module (draft state)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		channelID, _ := cmd.Flags().GetString("channel")
		description, _ := cmd.Flags().GetString("description")
		ownerID, _ := cmd.Flags().GetString("owner-id")
		licenseID, _ := cmd.Flags().GetString("license")

		return runModuleCreate(cmd.OutOrStdout(), modelSvc, model.ModuleRequest{
			Name:        name,
			Description: description,
			OwnerID:     ownerID,
			ChannelID:   channelID,
			LicenseID:   licenseID,
		})
	},
}

// ── get ───────────────────────────────────────────────────────────────────────

func runModuleGet(w io.Writer, svc model.ModelService, id string) error {
	m, err := svc.GetModule(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "ID        : %s\n", m.ID)
	fmt.Fprintf(w, "Nom       : %s\n", m.Name)
	fmt.Fprintf(w, "Canal     : %s\n", m.ChannelID)
	fmt.Fprintf(w, "Statut    : %s\n", m.Status)
	fmt.Fprintf(w, "Instances : %d\n", len(m.WorkspaceInstances))
	for _, inst := range m.WorkspaceInstances {
		fmt.Fprintf(w, "  - %s  asset=%s  (%.1f, %.1f)\n", inst.ID, inst.AssetID, inst.X, inst.Y)
	}
	fmt.Fprintf(w, "Liaisons  : %d\n", len(m.Assemblies))
	for _, connID := range m.Assemblies {
		fmt.Fprintf(w, "  - %s\n", connID)
	}
	return nil
}

var moduleGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show the composition of a module",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleGet(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

func runModuleList(w io.Writer, svc model.ModelService, channelID string) error {
	modules, err := svc.ListModules(channelID)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNOM\tSTATUT\tINSTANCES\tLIAISONS")
	for _, m := range modules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n", m.ID, m.Name, m.Status, len(m.WorkspaceInstances), len(m.Assemblies))
	}
	return tw.Flush()
}

var moduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List modules in a channel",
	RunE: func(cmd *cobra.Command, args []string) error {
		channelID, _ := cmd.Flags().GetString("channel")
		return runModuleList(cmd.OutOrStdout(), modelSvc, channelID)
	},
}

// ── interfaces ────────────────────────────────────────────────────────────────

func runModuleInterfaces(w io.Writer, svc model.ModelService, id string) error {
	ifaces, err := svc.GetModuleInterfaces(id)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCATÉGORIE\tTYPE\tDIRECTION\tUNITÉ")
	for _, i := range ifaces {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", i.ID, i.Category, i.Type, i.Direction, i.Unit)
	}
	return tw.Flush()
}

var moduleInterfacesCmd = &cobra.Command{
	Use:   "interfaces <id>",
	Short: "List the interfaces a module exposes (not internally connected)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleInterfaces(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── add-assembly / remove-assembly ───────────────────────────────────────────

func runModuleAddAssembly(w io.Writer, svc model.ModelService, moduleID, connID string) error {
	if err := svc.AddAssemblyToModule(moduleID, connID); err != nil {
		return err
	}
	fmt.Fprintf(w, "Liaison %s rattachée au module %s.\n", connID, moduleID)
	return nil
}

var moduleAddAssemblyCmd = &cobra.Command{
	Use:   "add-assembly <id> <connID>",
	Short: "Attach an assembly link to a module",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleAddAssembly(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

func runModuleRemoveAssembly(w io.Writer, svc model.ModelService, moduleID, connID string) error {
	if err := svc.RemoveAssemblyFromModule(moduleID, connID); err != nil {
		return err
	}
	fmt.Fprintf(w, "Liaison %s détachée du module %s.\n", connID, moduleID)
	return nil
}

var moduleRemoveAssemblyCmd = &cobra.Command{
	Use:   "remove-assembly <id> <connID>",
	Short: "Detach an assembly link from a module",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleRemoveAssembly(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

// ── submit ────────────────────────────────────────────────────────────────────

func runModuleSubmit(w io.Writer, svc model.ModelService, id, note string) error {
	m, err := svc.SubmitModule(id, note)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Module %s soumis (statut : %s).\n", m.ID, m.Status)
	return nil
}

var moduleSubmitCmd = &cobra.Command{
	Use:   "submit <id>",
	Short: "Submit a module to the blockchain",
	Long: `Anchor the current state of a module (instances + assembly links) on the
blockchain in a single transaction. Fails if the module has no assembly.

Example:
  myr module submit mod-123 --note "v1 release"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, _ := cmd.Flags().GetString("note")
		return runModuleSubmit(cmd.OutOrStdout(), modelSvc, args[0], note)
	},
}

// ── remove ────────────────────────────────────────────────────────────────────

func runModuleRemove(w io.Writer, svc model.ModelService, id string) error {
	if err := svc.RemoveModule(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Module %s retiré.\n", id)
	return nil
}

var moduleRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a module that has not been submitted yet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModuleRemove(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

func init() {
	moduleCreateCmd.Flags().String("name", "", "Nom du module (requis)")
	moduleCreateCmd.Flags().String("channel", "", "ID du canal")
	moduleCreateCmd.Flags().String("description", "", "Description du module")
	moduleCreateCmd.Flags().String("owner-id", "", "ID de l'identité propriétaire")
	moduleCreateCmd.Flags().String("license", "", "ID de licence dans le catalogue")
	moduleCreateCmd.MarkFlagRequired("name")

	moduleListCmd.Flags().String("channel", "", "ID du canal")

	moduleSubmitCmd.Flags().String("note", "", "Note de version associée à la soumission")

	moduleCmd.AddCommand(
		moduleCreateCmd, moduleGetCmd, moduleListCmd, moduleInterfacesCmd,
		moduleAddAssemblyCmd, moduleRemoveAssemblyCmd, moduleSubmitCmd, moduleRemoveCmd,
	)
}
