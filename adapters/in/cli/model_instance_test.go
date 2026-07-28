// adapters/in/cli/model_instance_test.go — tests des handlers CLI instances de module
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"myr-core/domain/model"
)

/// @brief  runModelInstanceAdd ajoute un asset comme instance et affiche le total
/// @input  moduleID="mod-1", assetID="asset-1"
/// @expect AddAssetToWorkspace reçoit les deux ids, sortie contient l'id du module
func TestRunModelInstanceAdd_NominalCase(t *testing.T) {
	var gotModule, gotAsset string
	svc := &mockModelSvc{addAssetToWorkspace: func(moduleID, assetID string) (*model.Model3D, error) {
		gotModule, gotAsset = moduleID, assetID
		return &model.Model3D{ID: moduleID, WorkspaceInstances: []model.WorkspaceInstance{{ID: "inst-1", AssetID: assetID}}}, nil
	}}
	var buf bytes.Buffer
	if err := runModelInstanceAdd(&buf, svc, "mod-1", "asset-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotModule != "mod-1" || gotAsset != "asset-1" {
		t.Fatalf("unexpected forwarded args: module=%q asset=%q", gotModule, gotAsset)
	}
	if !strings.Contains(buf.String(), "mod-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelInstanceAdd propage l'erreur du service (ex : module introuvable)
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunModelInstanceAdd_ServiceError_Rejected(t *testing.T) {
	wantErr := errors.New("module introuvable")
	svc := &mockModelSvc{addAssetToWorkspace: func(string, string) (*model.Model3D, error) { return nil, wantErr }}
	var buf bytes.Buffer
	err := runModelInstanceAdd(&buf, svc, "mod-1", "asset-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModelInstanceRemove retire l'instance et affiche le nombre restant (cascade RM15)
/// @input  moduleID="mod-1", instanceID="inst-1"
/// @expect RemoveAssetFromWorkspace reçoit les deux ids, sortie contient l'id du module
func TestRunModelInstanceRemove_NominalCase(t *testing.T) {
	var gotModule, gotInstance string
	svc := &mockModelSvc{removeAssetFromWorkspace: func(moduleID, instanceID string) (*model.Model3D, error) {
		gotModule, gotInstance = moduleID, instanceID
		return &model.Model3D{ID: moduleID}, nil
	}}
	var buf bytes.Buffer
	if err := runModelInstanceRemove(&buf, svc, "mod-1", "inst-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotModule != "mod-1" || gotInstance != "inst-1" {
		t.Fatalf("unexpected forwarded args: module=%q instance=%q", gotModule, gotInstance)
	}
	if !strings.Contains(buf.String(), "mod-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
