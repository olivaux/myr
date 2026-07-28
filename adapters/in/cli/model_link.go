// adapters/in/cli/model_link.go — liaisons d'assemblage entre interfaces
package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr-core/domain/model"
)

var modelLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage assembly links between interfaces",
}

// ── add ───────────────────────────────────────────────────────────────────────

func runModelLinkAdd(w io.Writer, svc model.ModelService, from, to, label, fromInstance, toInstance, fastener string) error {
	if from == "" || to == "" {
		return fmt.Errorf("--from et --to sont requis")
	}
	conn, err := svc.AddAssemblyLink(from, to, label, fromInstance, toInstance, fastener)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Liaison créée : id=%s incompatible=%v\n", conn.ID, conn.Incompatible)
	return nil
}

var modelLinkAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create an assembly link between two interfaces",
	Long: `Create a link between two physical interfaces, directly or through a
fastener asset (screw, cable, clip...). Compatibility is checked
automatically by the domain service.

Examples:
  myr model link add --from iface-1 --to iface-2
  myr model link add --from iface-1 --to iface-2 --fastener asset-screw-m3 --label "vis M3"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		label, _ := cmd.Flags().GetString("label")
		fastener, _ := cmd.Flags().GetString("fastener")
		fromInstance, _ := cmd.Flags().GetString("from-instance")
		toInstance, _ := cmd.Flags().GetString("to-instance")
		return runModelLinkAdd(cmd.OutOrStdout(), modelSvc, from, to, label, fromInstance, toInstance, fastener)
	},
}

// ── connect-virtual ───────────────────────────────────────────────────────────

func runModelLinkConnectVirtual(w io.Writer, svc model.ModelService, virtual, physical string, popup model.AssetInterface, fromInstance, toInstance string) error {
	if virtual == "" || physical == "" {
		return fmt.Errorf("--virtual et --physical sont requis")
	}
	conn, err := svc.ConnectVirtualToPhysical(virtual, physical, popup, fromInstance, toInstance)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Liaison créée : id=%s\n", conn.ID)
	return nil
}

var modelLinkConnectVirtualCmd = &cobra.Command{
	Use:   "connect-virtual",
	Short: "Materialize a virtual interface slot against a physical interface",
	Long: `Bind a virtual interface slot (EnsureVirtualSlot placeholder) to a
concrete physical interface, adopting its category/type and the opposite
direction.

Example:
  myr model link connect-virtual --virtual iface-virtual --physical iface-3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		virtual, _ := cmd.Flags().GetString("virtual")
		physical, _ := cmd.Flags().GetString("physical")
		name, _ := cmd.Flags().GetString("name")
		unit, _ := cmd.Flags().GetString("unit")
		valueMin, _ := cmd.Flags().GetFloat64("value-min")
		valueMax, _ := cmd.Flags().GetFloat64("value-max")
		fromInstance, _ := cmd.Flags().GetString("from-instance")
		toInstance, _ := cmd.Flags().GetString("to-instance")

		popup := model.AssetInterface{
			Name:     name,
			Unit:     unit,
			ValueMin: valueMin,
			ValueMax: valueMax,
			IsRange:  cmd.Flags().Changed("value-max"),
		}
		return runModelLinkConnectVirtual(cmd.OutOrStdout(), modelSvc, virtual, physical, popup, fromInstance, toInstance)
	},
}

// ── remove ────────────────────────────────────────────────────────────────────

func runModelLinkRemove(w io.Writer, svc model.ModelService, id string) error {
	if err := svc.RemoveConnection(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Liaison %s supprimée.\n", id)
	return nil
}

var modelLinkRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an assembly link",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelLinkRemove(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

func runModelLinkList(w io.Writer, svc model.ModelService) error {
	conns, err := svc.ListConnections()
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tDE\tVERS\tLABEL\tACCROCHE\tINCOMPATIBLE")
	for _, c := range conns {
		incompatible := ""
		if c.Incompatible {
			incompatible = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", c.ID, c.From, c.To, c.Label, c.FastenerAssetID, incompatible)
	}
	return tw.Flush()
}

var modelLinkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all assembly links",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelLinkList(cmd.OutOrStdout(), modelSvc)
	},
}

func init() {
	modelLinkAddCmd.Flags().String("from", "", "ID interface source (requis)")
	modelLinkAddCmd.Flags().String("to", "", "ID interface cible (requis)")
	modelLinkAddCmd.Flags().String("label", "", "Label de la liaison")
	modelLinkAddCmd.Flags().String("fastener", "", "ID de l'asset d'accroche (optionnel)")
	modelLinkAddCmd.Flags().String("from-instance", "", "ID instance de module pour l'interface source (optionnel)")
	modelLinkAddCmd.Flags().String("to-instance", "", "ID instance de module pour l'interface cible (optionnel)")

	modelLinkConnectVirtualCmd.Flags().String("virtual", "", "ID de l'interface virtuelle (requis)")
	modelLinkConnectVirtualCmd.Flags().String("physical", "", "ID de l'interface physique (requis)")
	modelLinkConnectVirtualCmd.Flags().String("name", "", "Label (sinon reprend celui du physique)")
	modelLinkConnectVirtualCmd.Flags().String("unit", "", "Unité (sinon reprend celle du physique)")
	modelLinkConnectVirtualCmd.Flags().Float64("value-min", 0, "Valeur unique ou borne basse")
	modelLinkConnectVirtualCmd.Flags().Float64("value-max", 0, "Borne haute (implique une plage)")
	modelLinkConnectVirtualCmd.Flags().String("from-instance", "", "ID instance portant le slot virtuel (optionnel)")
	modelLinkConnectVirtualCmd.Flags().String("to-instance", "", "ID instance portant l'interface physique (optionnel)")

	modelLinkCmd.AddCommand(modelLinkAddCmd, modelLinkConnectVirtualCmd, modelLinkRemoveCmd, modelLinkListCmd)
	modelCmd.AddCommand(modelLinkCmd)
}
