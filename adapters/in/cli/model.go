// adapters/in/cli/model.go — commandes Cobra pour les modeles 3D
package cli

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"myr/domain/model"
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage 3D models",
	Long: `Commands to publish, inspect and verify 3D models on the Myr platform.

Models are stored on the active blockchain network. Each model has a
unique ID and belongs to a channel.`,
}

// ── add ───────────────────────────────────────────────────────────────────────

func runModelAdd(w io.Writer, svc model.ModelService, req model.AddRequest) error {
	m, err := svc.AddFull(req)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Modèle ajouté : %s  id=%s\n", m.Name, m.ID)
	return nil
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
  myr model add pivot.step --name "Left pivot"   --channel greenchannel --tags "mechanical" --category amelioration --parent abc123 --license mit`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		channelID, _ := cmd.Flags().GetString("channel")
		category, _ := cmd.Flags().GetString("category")
		parentID, _ := cmd.Flags().GetString("parent")
		licenseID, _ := cmd.Flags().GetString("license")
		ownerID, _ := cmd.Flags().GetString("owner-id")
		tagsStr, _ := cmd.Flags().GetString("tags")

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		return runModelAdd(cmd.OutOrStdout(), modelSvc, model.AddRequest{
			FilePath:    args[0],
			Name:        name,
			Description: description,
			Category:    model.Category(category),
			ChannelID:   channelID,
			OwnerID:     ownerID,
			ParentID:    parentID,
			LicenseID:   licenseID,
			Tags:        tags,
		})
	},
}

// ── get ───────────────────────────────────────────────────────────────────────

func runModelGet(w io.Writer, svc model.ModelService, id string) error {
	m, err := svc.Get(id, "")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "ID      : %s\nNom     : %s\nCanal   : %s\nOwner   : %s\nParent  : %s\nLicence : %s\nTags    : %s\n",
		m.ID, m.Name, m.ChannelID, m.OwnerID, m.ParentID, m.LicenseID, strings.Join(m.Tags, ", "))
	return nil
}

var modelGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a model",
	Long: `Display the metadata of a model by its blockchain identifier.

Example:
  myr model get abc123def456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelGet(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── list ──────────────────────────────────────────────────────────────────────

func runModelList(w io.Writer, svc model.ModelService, channelID string) error {
	models, err := svc.List(channelID)
	if err != nil {
		return err
	}
	return printModelTable(w, models)
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
		return runModelList(cmd.OutOrStdout(), modelSvc, channelID)
	},
}

// ── verify ────────────────────────────────────────────────────────────────────

func runModelVerify(w io.Writer, svc model.ModelService, id string) error {
	ok, err := svc.Verify(id, "")
	if err != nil {
		return err
	}
	if ok {
		fmt.Fprintf(w, "OK  modèle %s : intégrité vérifiée\n", id)
	} else {
		fmt.Fprintf(w, "KO  modèle %s : intégrité compromise\n", id)
	}
	return nil
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
		return runModelVerify(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── update ────────────────────────────────────────────────────────────────────

func runModelUpdate(w io.Writer, svc model.ModelService, req model.UpdateRequest) error {
	m, err := svc.UpdateAsset(req)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Modèle %s mis à jour.\n", m.ID)
	return nil
}

var modelUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update the metadata of an existing asset",
	Long: `Patch the metadata of an existing component or module. Only flags
explicitly provided are changed — everything else is left untouched.

Example:
  myr model update abc123def456 --description "Nouvelle description" --tags "3dprint,mechanical"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		licenseID, _ := cmd.Flags().GetString("license")
		tagsStr, _ := cmd.Flags().GetString("tags")
		linksStr, _ := cmd.Flags().GetString("links")

		var tags, links []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}
		if linksStr != "" {
			links = strings.Split(linksStr, ",")
		}

		return runModelUpdate(cmd.OutOrStdout(), modelSvc, model.UpdateRequest{
			ID:          args[0],
			Name:        name,
			Description: description,
			LicenseID:   licenseID,
			Tags:        tags,
			Links:       links,
		})
	},
}

// ── remove ────────────────────────────────────────────────────────────────────

func runModelRemove(w io.Writer, svc model.ModelService, id string) error {
	if err := svc.Remove(id); err != nil {
		return err
	}
	fmt.Fprintf(w, "Modèle %s retiré.\n", id)
	return nil
}

var modelRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelRemove(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

// ── children ──────────────────────────────────────────────────────────────────

func runModelChildren(w io.Writer, svc model.ModelService, parentID string) error {
	children, err := svc.GetChildren(parentID)
	if err != nil {
		return err
	}
	return printModelTable(w, children)
}

var modelChildrenCmd = &cobra.Command{
	Use:   "children <parentID>",
	Short: "List the assets derived from a parent asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runModelChildren(cmd.OutOrStdout(), modelSvc, args[0])
	},
}

func printModelTable(w io.Writer, models []*model.Model3D) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNOM\tCATÉGORIE\tTAGS")
	for _, m := range models {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Category, strings.Join(m.Tags, ", "))
	}
	return tw.Flush()
}

// ── thumbnail ─────────────────────────────────────────────────────────────────

var modelThumbnailCmd = &cobra.Command{
	Use:   "thumbnail",
	Short: "Manage the thumbnail image associated with an asset",
}

var thumbnailMimeByExt = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func runModelThumbnailSet(w io.Writer, svc model.ModelService, id string, fileName string, data []byte) error {
	mime := thumbnailMimeByExt[strings.ToLower(filepath.Ext(fileName))]
	if mime == "" {
		mime = "application/octet-stream"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))
	if err := svc.SaveThumbnail(id, dataURL); err != nil {
		return err
	}
	fmt.Fprintf(w, "Miniature enregistrée pour %s.\n", id)
	return nil
}

var modelThumbnailSetCmd = &cobra.Command{
	Use:   "set <id> <file>",
	Short: "Attach an image file as an asset's thumbnail",
	Long: `Read an image file, encode it as a base64 data URL and attach it to the
asset as its thumbnail.

Example:
  myr model thumbnail set abc123def456 preview.png`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("lecture fichier : %w", err)
		}
		return runModelThumbnailSet(cmd.OutOrStdout(), modelSvc, args[0], args[1], data)
	},
}

func runModelThumbnailGet(w io.Writer, svc model.ModelService, id string, out string, writeFile func(name string, data []byte) error) error {
	dataURL, err := svc.GetThumbnail(id)
	if err != nil {
		return err
	}
	if out == "" {
		fmt.Fprintln(w, dataURL)
		return nil
	}
	_, b64, found := strings.Cut(dataURL, ",")
	if !found {
		return fmt.Errorf("miniature invalide : format data URL attendu")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("décodage miniature : %w", err)
	}
	if err := writeFile(out, data); err != nil {
		return fmt.Errorf("écriture fichier : %w", err)
	}
	fmt.Fprintf(w, "Miniature écrite dans %s.\n", out)
	return nil
}

var modelThumbnailGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Retrieve an asset's thumbnail",
	Long: `Print the thumbnail as a base64 data URL, or decode it to a file with --out.

Examples:
  myr model thumbnail get abc123def456
  myr model thumbnail get abc123def456 --out preview.png`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		return runModelThumbnailGet(cmd.OutOrStdout(), modelSvc, args[0], out, func(name string, data []byte) error {
			return os.WriteFile(name, data, 0o644)
		})
	},
}

func init() {
	modelAddCmd.Flags().String("name", "", "Nom du modele (requis)")
	modelAddCmd.Flags().String("description", "", "Description du modèle")
	modelAddCmd.Flags().String("channel", "", "ID du canal")
	modelAddCmd.Flags().String("category", "", "Catégorie (base, amelioration, variation, adaptation, derivation, extension, regression)")
	modelAddCmd.Flags().String("parent", "", "ID du composant parent (lignée)")
	modelAddCmd.Flags().String("license", "", "ID de licence dans le catalogue")
	modelAddCmd.Flags().String("owner-id", "", "ID de l'identité propriétaire")
	modelAddCmd.Flags().String("tags", "", "Tags separes par des virgules")
	modelAddCmd.MarkFlagRequired("name")

	modelListCmd.Flags().String("channel", "", "ID du canal")

	modelUpdateCmd.Flags().String("name", "", "Nouveau nom")
	modelUpdateCmd.Flags().String("description", "", "Nouvelle description")
	modelUpdateCmd.Flags().String("license", "", "Nouvel ID de licence")
	modelUpdateCmd.Flags().String("tags", "", "Nouveaux tags séparés par des virgules (remplace l'existant)")
	modelUpdateCmd.Flags().String("links", "", "Nouveaux liens séparés par des virgules (remplace l'existant)")

	modelThumbnailGetCmd.Flags().String("out", "", "Fichier de sortie (décode le base64 au lieu d'imprimer la data URL)")
	modelThumbnailCmd.AddCommand(modelThumbnailSetCmd, modelThumbnailGetCmd)

	modelCmd.AddCommand(
		modelAddCmd, modelGetCmd, modelListCmd, modelVerifyCmd,
		modelUpdateCmd, modelRemoveCmd, modelChildrenCmd, modelThumbnailCmd,
	)
}
