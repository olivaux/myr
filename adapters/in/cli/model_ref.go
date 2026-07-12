// adapters/in/cli/model_ref.go — vocabulaire de référence des interfaces (catégories/types/unités)
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"myr/domain/model"
)

var modelRefCmd = &cobra.Command{
	Use:   "ref",
	Short: "Manage the interface vocabulary (categories, types, units)",
}

func runModelRefList(w io.Writer, svc model.ModelService) error {
	refs, err := svc.GetRefs()
	if err != nil {
		return err
	}
	for _, cat := range refs.Categories {
		fmt.Fprintf(w, "%s\n", cat)
		fmt.Fprintf(w, "  types : %v\n", refs.Types[cat])
		fmt.Fprintf(w, "  unités: %v\n", refs.Units[cat])
	}
	return nil
}

var modelRefListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the current interface vocabulary",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelRefList(cmd.OutOrStdout(), modelSvc)
	},
}

func runModelRefAddCategory(w io.Writer, svc model.ModelService, cat string) error {
	if err := svc.AddRefCategory(cat); err != nil {
		return err
	}
	fmt.Fprintf(w, "Catégorie %q ajoutée.\n", cat)
	return nil
}

var modelRefAddCategoryCmd = &cobra.Command{
	Use:   "add-category <cat>",
	Short: "Add a new interface category to the vocabulary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelRefAddCategory(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

func runModelRefAddType(w io.Writer, svc model.ModelService, cat, typeName string) error {
	if err := svc.AddRefType(cat, typeName); err != nil {
		return err
	}
	fmt.Fprintf(w, "Type %q ajouté à la catégorie %q.\n", typeName, cat)
	return nil
}

var modelRefAddTypeCmd = &cobra.Command{
	Use:   "add-type <cat> <type>",
	Short: "Add a new interface type under a category",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelRefAddType(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

func runModelRefAddUnit(w io.Writer, svc model.ModelService, cat, unit string) error {
	if err := svc.AddRefUnit(cat, unit); err != nil {
		return err
	}
	fmt.Fprintf(w, "Unité %q ajoutée à la catégorie %q.\n", unit, cat)
	return nil
}

var modelRefAddUnitCmd = &cobra.Command{
	Use:   "add-unit <cat> <unit>",
	Short: "Add a new unit under a category",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelRefAddUnit(cmd.OutOrStdout(), modelSvc, args[0], args[1])
	},
}

func init() {
	modelRefCmd.AddCommand(modelRefListCmd, modelRefAddCategoryCmd, modelRefAddTypeCmd, modelRefAddUnitCmd)
	modelCmd.AddCommand(modelRefCmd)
}
