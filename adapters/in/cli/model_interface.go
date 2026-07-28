// adapters/in/cli/model_interface.go — interfaces physiques d'un asset
package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr-core/domain/model"
)

var modelInterfaceCmd = &cobra.Command{
	Use:   "interface",
	Short: "Manage the physical/virtual interfaces of an asset",
}

// ── add ───────────────────────────────────────────────────────────────────────

func runModelInterfaceAdd(w io.Writer, svc model.ModelService, iface *model.AssetInterface) error {
	if err := svc.AddInterface(iface); err != nil {
		return err
	}
	fmt.Fprintf(w, "Interface ajoutée : id=%s\n", iface.ID)
	return nil
}

var modelInterfaceAddCmd = &cobra.Command{
	Use:   "add <assetID>",
	Short: "Define a new interface on an asset",
	Long: `Define a new physical connection point on a component or module.

Example:
  myr model interface add abc123 --category ELEC --type USB-C --direction out --unit V --value-min 5`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		iface := &model.AssetInterface{AssetID: args[0]}
		applyInterfaceFlags(cmd, iface)
		return runModelInterfaceAdd(cmd.OutOrStdout(), modelSvc, iface)
	},
}

// ── update ────────────────────────────────────────────────────────────────────

func runModelInterfaceUpdate(w io.Writer, svc model.ModelService, id string, apply func(*model.AssetInterface)) error {
	iface, err := svc.GetInterface(id)
	if err != nil {
		return err
	}
	apply(iface)
	if err := svc.UpdateInterface(iface); err != nil {
		return err
	}
	fmt.Fprintf(w, "Interface %s mise à jour.\n", iface.ID)
	return nil
}

var modelInterfaceUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInterfaceUpdate(cmd.OutOrStdout(), modelSvc, args[0], func(iface *model.AssetInterface) {
			applyInterfaceFlags(cmd, iface)
		})
	},
}

// ── remove ────────────────────────────────────────────────────────────────────

func runModelInterfaceRemove(w io.Writer, svc model.ModelService, id string) error {
	if err := svc.RemoveInterface(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Interface %s supprimée.\n", id)
	return nil
}

var modelInterfaceRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInterfaceRemove(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

func runModelInterfaceList(w io.Writer, svc model.ModelService, assetID string) error {
	ifaces, err := svc.ListInterfacesForAsset(assetID)
	if err != nil {
		return err
	}
	if len(ifaces) == 0 {
		if mod, err := svc.GetModule(assetID); err == nil && mod != nil {
			ifaces, err = svc.GetModuleInterfaces(assetID)
			if err != nil {
				return err
			}
		}
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCATÉGORIE\tTYPE\tDIRECTION\tVALEUR\tUNITÉ\tVIRTUELLE")
	for _, i := range ifaces {
		value := fmt.Sprintf("%g", i.ValueMin)
		if i.IsRange {
			value = fmt.Sprintf("%g..%g", i.ValueMin, i.ValueMax)
		}
		virtual := ""
		if i.Virtual {
			virtual = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", i.ID, i.Category, i.Type, i.Direction, value, i.Unit, virtual)
	}
	return tw.Flush()
}

var modelInterfaceListCmd = &cobra.Command{
	Use:   "list <assetID>",
	Short: "List the interfaces of an asset (component or module)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInterfaceList(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── get ───────────────────────────────────────────────────────────────────────

func runModelInterfaceGet(w io.Writer, svc model.ModelService, id string) error {
	i, err := svc.GetInterface(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "ID        : %s\n", i.ID)
	fmt.Fprintf(w, "Asset     : %s\n", i.AssetID)
	fmt.Fprintf(w, "Nom       : %s\n", i.Name)
	fmt.Fprintf(w, "Catégorie : %s\n", i.Category)
	fmt.Fprintf(w, "Type      : %s\n", i.Type)
	fmt.Fprintf(w, "Direction : %s\n", i.Direction)
	fmt.Fprintf(w, "Valeur    : %g..%g (plage=%v)\n", i.ValueMin, i.ValueMax, i.IsRange)
	fmt.Fprintf(w, "Unité     : %s\n", i.Unit)
	fmt.Fprintf(w, "Virtuelle : %v\n", i.Virtual)
	return nil
}

var modelInterfaceGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a single interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelInterfaceGet(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── flags partagés add/update ────────────────────────────────────────────────

func applyInterfaceFlags(cmd *cobra.Command, iface *model.AssetInterface) {
	if v, _ := cmd.Flags().GetString("category"); cmd.Flags().Changed("category") {
		iface.Category = v
	}
	if v, _ := cmd.Flags().GetString("type"); cmd.Flags().Changed("type") {
		iface.Type = v
	}
	if v, _ := cmd.Flags().GetString("direction"); cmd.Flags().Changed("direction") {
		iface.Direction = model.IfaceDirection(v)
	}
	if v, _ := cmd.Flags().GetString("name"); cmd.Flags().Changed("name") {
		iface.Name = v
	}
	if v, _ := cmd.Flags().GetString("unit"); cmd.Flags().Changed("unit") {
		iface.Unit = v
	}
	if cmd.Flags().Changed("value-min") {
		v, _ := cmd.Flags().GetFloat64("value-min")
		iface.ValueMin = v
	}
	if cmd.Flags().Changed("value-max") {
		v, _ := cmd.Flags().GetFloat64("value-max")
		iface.ValueMax = v
		iface.IsRange = true
	}
}

func addInterfaceFlags(cmd *cobra.Command) {
	cmd.Flags().String("category", "", "Catégorie (ELEC, MECA, HYD, ou personnalisée)")
	cmd.Flags().String("type", "", "Type dans la catégorie (ex: USB-C, Vis M3)")
	cmd.Flags().String("direction", "", "Direction : in, out, bidir")
	cmd.Flags().String("name", "", "Label optionnel")
	cmd.Flags().String("unit", "", "Unité (ex: V, mm, bar)")
	cmd.Flags().Float64("value-min", 0, "Valeur unique ou borne basse")
	cmd.Flags().Float64("value-max", 0, "Borne haute (implique une plage)")
}

func init() {
	addInterfaceFlags(modelInterfaceAddCmd)
	addInterfaceFlags(modelInterfaceUpdateCmd)

	modelInterfaceCmd.AddCommand(
		modelInterfaceAddCmd, modelInterfaceUpdateCmd, modelInterfaceRemoveCmd,
		modelInterfaceListCmd, modelInterfaceGetCmd,
	)
	modelCmd.AddCommand(modelInterfaceCmd)
}
