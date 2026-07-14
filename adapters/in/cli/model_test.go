// adapters/in/cli/model_test.go — tests des handlers CLI model (composant)
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"myr/domain/model"
)

/// @brief  runModelAdd doit appeler AddFull avec la requête construite et afficher l'id créé
/// @input  AddRequest{Name: "Roue"}, service retournant un Model3D{ID: "abc", Name: "Roue"}
/// @expect Sortie contient l'id et le nom, pas d'erreur
func TestRunModelAdd_NominalCase(t *testing.T) {
	var captured model.AddRequest
	svc := &mockModelSvc{addFull: func(req model.AddRequest) (*model.Model3D, error) {
		captured = req
		return &model.Model3D{ID: "abc", Name: req.Name}, nil
	}}
	var buf bytes.Buffer
	err := runModelAdd(&buf, svc, model.AddRequest{Name: "Roue", ChannelID: "green"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Name != "Roue" || captured.ChannelID != "green" {
		t.Fatalf("request not forwarded correctly: %+v", captured)
	}
	if !strings.Contains(buf.String(), "abc") || !strings.Contains(buf.String(), "Roue") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelAdd doit propager l'erreur du service (ex : incompatibilité de licence RM03)
/// @input  service retournant une erreur
/// @expect L'erreur est retournée telle quelle
func TestRunModelAdd_ServiceError_Rejected(t *testing.T) {
	wantErr := errors.New("incompatibilité de licence")
	svc := &mockModelSvc{addFull: func(model.AddRequest) (*model.Model3D, error) { return nil, wantErr }}
	var buf bytes.Buffer
	err := runModelAdd(&buf, svc, model.AddRequest{Name: "Roue"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModelGet doit afficher les métadonnées d'un modèle
/// @input  id="abc", service retournant Model3D{ID: "abc", Name: "Roue"}
/// @expect Sortie contient l'id et le nom
func TestRunModelGet_NominalCase(t *testing.T) {
	svc := &mockModelSvc{get: func(id, _ string) (*model.Model3D, error) {
		return &model.Model3D{ID: id, Name: "Roue"}, nil
	}}
	var buf bytes.Buffer
	if err := runModelGet(&buf, svc, "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "abc") || !strings.Contains(buf.String(), "Roue") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelGet doit propager l'erreur "introuvable"
/// @input  service retournant model.ErrNotFound
/// @expect L'erreur est retournée
func TestRunModelGet_NotFound_Rejected(t *testing.T) {
	svc := &mockModelSvc{get: func(string, string) (*model.Model3D, error) { return nil, model.ErrNotFound }}
	var buf bytes.Buffer
	err := runModelGet(&buf, svc, "missing")
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

/// @brief  runModelList doit afficher un tableau avec une ligne par modèle
/// @input  service retournant 2 modèles
/// @expect Les deux ID apparaissent dans la sortie
func TestRunModelList_NominalCase(t *testing.T) {
	svc := &mockModelSvc{list: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, nil
	}}
	var buf bytes.Buffer
	if err := runModelList(&buf, svc, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runModelList sur un canal vide ne doit pas planter et affiche seulement l'en-tête
/// @input  service retournant une liste vide
/// @expect Pas d'erreur, sortie non vide (en-tête tableau)
func TestRunModelList_Empty(t *testing.T) {
	svc := &mockModelSvc{list: func(string) ([]*model.Model3D, error) { return nil, nil }}
	var buf bytes.Buffer
	if err := runModelList(&buf, svc, "green"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "ID") {
		t.Fatalf("expected table header, got: %q", buf.String())
	}
}

/// @brief  runModelVerify affiche OK quand l'intégrité est vérifiée
/// @input  service retournant true
/// @expect Sortie contient "OK"
func TestRunModelVerify_OK(t *testing.T) {
	svc := &mockModelSvc{verify: func(string, string) (bool, error) { return true, nil }}
	var buf bytes.Buffer
	if err := runModelVerify(&buf, svc, "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Fatalf("expected OK in output, got: %q", buf.String())
	}
}

/// @brief  runModelVerify affiche KO quand l'intégrité est compromise
/// @input  service retournant false
/// @expect Sortie contient "KO"
func TestRunModelVerify_Compromised(t *testing.T) {
	svc := &mockModelSvc{verify: func(string, string) (bool, error) { return false, nil }}
	var buf bytes.Buffer
	if err := runModelVerify(&buf, svc, "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "KO") {
		t.Fatalf("expected KO in output, got: %q", buf.String())
	}
}

/// @brief  runModelUpdate transmet la requête de patch et affiche l'id mis à jour
/// @input  UpdateRequest{ID: "abc", Description: "new"}
/// @expect Requête transmise telle quelle, sortie contient l'id
func TestRunModelUpdate_NominalCase(t *testing.T) {
	var captured model.UpdateRequest
	svc := &mockModelSvc{updateAsset: func(req model.UpdateRequest) (*model.Model3D, error) {
		captured = req
		return &model.Model3D{ID: req.ID}, nil
	}}
	var buf bytes.Buffer
	err := runModelUpdate(&buf, svc, model.UpdateRequest{ID: "abc", Description: "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Description != "new" {
		t.Fatalf("request not forwarded: %+v", captured)
	}
	if !strings.Contains(buf.String(), "abc") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelRemove supprime l'asset et confirme en sortie
/// @input  id="abc"
/// @expect Sortie confirme la suppression, pas d'erreur
func TestRunModelRemove_NominalCase(t *testing.T) {
	removed := ""
	svc := &mockModelSvc{remove: func(id string) error { removed = id; return nil }}
	var buf bytes.Buffer
	if err := runModelRemove(&buf, svc, "abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != "abc" {
		t.Fatalf("expected Remove called with abc, got %q", removed)
	}
	if !strings.Contains(buf.String(), "abc") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelRemove propage une erreur de suppression non supportée (Fabric, règle 9)
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunModelRemove_NotSupported_Rejected(t *testing.T) {
	wantErr := errors.New("suppression non supportée")
	svc := &mockModelSvc{remove: func(string) error { return wantErr }}
	var buf bytes.Buffer
	err := runModelRemove(&buf, svc, "abc")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModelChildren liste les dérivés d'un composant parent
/// @input  service retournant 1 enfant
/// @expect L'id de l'enfant apparaît dans la sortie
func TestRunModelChildren_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getChildren: func(parentID string) ([]*model.Model3D, error) {
		return []*model.Model3D{{ID: "child-1", ParentID: parentID}}, nil
	}}
	var buf bytes.Buffer
	if err := runModelChildren(&buf, svc, "parent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "child-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelThumbnailSet encode le fichier en data URL et l'enregistre
/// @input  fichier "preview.png", contenu binaire arbitraire
/// @expect SaveThumbnail reçoit une data URL commençant par "data:image/png;base64,"
func TestRunModelThumbnailSet_NominalCase(t *testing.T) {
	var savedURL string
	svc := &mockModelSvc{saveThumbnail: func(assetID, dataURL string) error {
		savedURL = dataURL
		return nil
	}}
	var buf bytes.Buffer
	err := runModelThumbnailSet(&buf, svc, "abc", "preview.png", []byte{0x89, 0x50, 0x4e, 0x47})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(savedURL, "data:image/png;base64,") {
		t.Fatalf("unexpected data URL: %q", savedURL)
	}
}

/// @brief  runModelThumbnailGet imprime la data URL brute quand --out n'est pas fourni
/// @input  out="", GetThumbnail retourne une data URL
/// @expect La data URL est écrite sur la sortie
func TestRunModelThumbnailGet_PrintsDataURL(t *testing.T) {
	svc := &mockModelSvc{getThumbnail: func(string) (string, error) { return "data:image/png;base64,AAAA", nil }}
	var buf bytes.Buffer
	err := runModelThumbnailGet(&buf, svc, "abc", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "data:image/png;base64,AAAA") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelThumbnailGet décode le base64 et écrit le fichier quand --out est fourni
/// @input  out="preview.png", GetThumbnail retourne une data URL base64 valide
/// @expect writeFile est appelé avec les octets décodés
func TestRunModelThumbnailGet_WritesFile(t *testing.T) {
	svc := &mockModelSvc{getThumbnail: func(string) (string, error) { return "data:image/png;base64,QUJD", nil }}
	var writtenName string
	var writtenData []byte
	err := runModelThumbnailGet(&bytes.Buffer{}, svc, "abc", "preview.png", func(name string, data []byte) error {
		writtenName = name
		writtenData = data
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writtenName != "preview.png" || string(writtenData) != "ABC" {
		t.Fatalf("unexpected write: name=%q data=%q", writtenName, writtenData)
	}
}
