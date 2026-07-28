// adapters/in/cli/model_license.go — catalogue de licences et vérification de compatibilité
package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr-core/domain/model"
)

var modelLicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Browse the license catalogue and check compatibility",
}

// ── list ──────────────────────────────────────────────────────────────────────

func runModelLicenseList(w io.Writer, svc model.ModelService) error {
	licenses := svc.ListLicenses()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNOM\tSPDX")
	for _, l := range licenses {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", l.ID, l.Name, l.SPDX)
	}
	return tw.Flush()
}

var modelLicenseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the license catalogue",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelLicenseList(cmd.OutOrStdout(), modelSvc)
	},
}

// ── get ───────────────────────────────────────────────────────────────────────

func runModelLicenseGet(w io.Writer, svc model.ModelService, id string) error {
	l, err := svc.GetLicense(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "ID          : %s\n", l.ID)
	fmt.Fprintf(w, "Nom         : %s\n", l.Name)
	fmt.Fprintf(w, "SPDX        : %s\n", l.SPDX)
	fmt.Fprintf(w, "Description : %s\n", l.Description)
	fmt.Fprintf(w, "Permissions : %s\n", strings.Join(l.Permissions, ", "))
	fmt.Fprintf(w, "Conditions  : %s\n", strings.Join(l.Conditions, ", "))
	fmt.Fprintf(w, "Limitations : %s\n", strings.Join(l.Limitations, ", "))
	fmt.Fprintf(w, "Compatible avec : %s\n", strings.Join(l.CompatibleWith, ", "))
	return nil
}

var modelLicenseGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show the detail of a license",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelLicenseGet(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── check ─────────────────────────────────────────────────────────────────────

func runModelLicenseCheck(w io.Writer, svc model.ModelService, parentLicenseID string, componentLicenseIDs []string, proposedLicenseID string) error {
	if proposedLicenseID == "" {
		return fmt.Errorf("--proposed est requis")
	}

	var check *model.LicenseCheck
	if len(componentLicenseIDs) > 0 {
		check = svc.CheckModuleLicenseCompatibility(componentLicenseIDs, proposedLicenseID)
	} else {
		if parentLicenseID == "" {
			return fmt.Errorf("--parent ou --component est requis")
		}
		check = svc.CheckLicenseCompatibility(parentLicenseID, proposedLicenseID)
	}

	if check.Compatible {
		fmt.Fprintln(w, "OK  licences compatibles")
	} else {
		fmt.Fprintf(w, "KO  licences incompatibles : %s\n", check.Reason)
	}
	if len(check.Allowed) > 0 {
		fmt.Fprintf(w, "Licences autorisées : %s\n", strings.Join(check.Allowed, ", "))
	}
	return nil
}

var (
	modelLicenseCheckParent     string
	modelLicenseCheckComponents []string
)

var modelLicenseCheckCmd = &cobra.Command{
	Use:   "check --proposed <id> (--parent <id> | --component <id> [--component <id> ...])",
	Short: "Check license compatibility for a derivation or a module product",
	Long: `Check whether a proposed license is compatible with either a single
parent license (component derivation) or a set of component licenses
(module product license).

Examples:
  myr model license check --parent mit --proposed cc-by-4.0
  myr model license check --component mit --component cc-by-4.0 --proposed proprietary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		proposed, _ := cmd.Flags().GetString("proposed")
		return runModelLicenseCheck(cmd.OutOrStdout(), modelSvc, modelLicenseCheckParent, modelLicenseCheckComponents, proposed)
	},
}

func init() {
	modelLicenseCheckCmd.Flags().StringVar(&modelLicenseCheckParent, "parent", "", "ID de la licence parente (dérivation d'un composant)")
	modelLicenseCheckCmd.Flags().StringArrayVar(&modelLicenseCheckComponents, "component", nil, "ID de licence d'un composant du module (répétable)")
	modelLicenseCheckCmd.Flags().String("proposed", "", "ID de la licence proposée (requis)")
	modelLicenseCheckCmd.MarkFlagRequired("proposed")

	modelLicenseCmd.AddCommand(modelLicenseListCmd, modelLicenseGetCmd, modelLicenseCheckCmd)
	modelCmd.AddCommand(modelLicenseCmd)
}
