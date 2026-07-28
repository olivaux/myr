// adapters/in/cli/module_test.go — tests des handlers CLI module
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"myr-core/domain/model"
)

/// @brief  runModuleCreate crée un module en état draft et affiche son id
/// @input  ModuleRequest{Name: "Chassis"}
/// @expect Requête transmise telle quelle, sortie contient l'id créé
func TestRunModuleCreate_NominalCase(t *testing.T) {
	var captured model.ModuleRequest
	svc := &mockModelSvc{createModule: func(req model.ModuleRequest) (*model.Model3D, error) {
		captured = req
		return &model.Model3D{ID: "mod-1", Name: req.Name}, nil
	}}
	var buf bytes.Buffer
	err := runModuleCreate(&buf, svc, model.ModuleRequest{Name: "Chassis", ChannelID: "green"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Name != "Chassis" || captured.ChannelID != "green" {
		t.Fatalf("request not forwarded correctly: %+v", captured)
	}
	if !strings.Contains(buf.String(), "mod-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModuleGet affiche la composition (instances + liaisons) d'un module
/// @input  GetModule retourne un module avec 1 instance et 1 liaison
/// @expect Sortie contient l'id de l'instance et l'id de la liaison
func TestRunModuleGet_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getModule: func(id string) (*model.Model3D, error) {
		return &model.Model3D{
			ID:                 id,
			WorkspaceInstances: []model.WorkspaceInstance{{ID: "inst-1", AssetID: "asset-1"}},
			Assemblies:         []string{"conn-1"},
		}, nil
	}}
	var buf bytes.Buffer
	if err := runModuleGet(&buf, svc, "mod-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "inst-1") || !strings.Contains(out, "conn-1") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runModuleList affiche une ligne par module avec compteurs d'instances/liaisons
/// @input  service retournant 1 module
/// @expect Sortie contient l'id du module
func TestRunModuleList_NominalCase(t *testing.T) {
	svc := &mockModelSvc{listModules: func(string) ([]*model.Model3D, error) {
		return []*model.Model3D{{ID: "mod-1", Status: model.ModuleDraft}}, nil
	}}
	var buf bytes.Buffer
	if err := runModuleList(&buf, svc, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "mod-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModuleInterfaces affiche les interfaces exposées (non connectées en interne)
/// @input  GetModuleInterfaces retourne 1 interface
/// @expect Sortie contient l'id de l'interface
func TestRunModuleInterfaces_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getModuleInterfaces: func(moduleID string) ([]*model.AssetInterface, error) {
		return []*model.AssetInterface{{ID: "exposed-1"}}, nil
	}}
	var buf bytes.Buffer
	if err := runModuleInterfaces(&buf, svc, "mod-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "exposed-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModuleAddAssembly rattache une liaison au module (UCMOD02)
/// @input  moduleID="mod-1", connID="conn-1"
/// @expect AddAssemblyToModule reçoit les deux ids
func TestRunModuleAddAssembly_NominalCase(t *testing.T) {
	var gotModule, gotConn string
	svc := &mockModelSvc{addAssemblyToModule: func(moduleID, connID string) error {
		gotModule, gotConn = moduleID, connID
		return nil
	}}
	var buf bytes.Buffer
	if err := runModuleAddAssembly(&buf, svc, "mod-1", "conn-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotModule != "mod-1" || gotConn != "conn-1" {
		t.Fatalf("unexpected forwarded args: module=%q conn=%q", gotModule, gotConn)
	}
}

/// @brief  runModuleRemoveAssembly détache une liaison du module
/// @input  moduleID="mod-1", connID="conn-1"
/// @expect RemoveAssemblyFromModule reçoit les deux ids
func TestRunModuleRemoveAssembly_NominalCase(t *testing.T) {
	var gotModule, gotConn string
	svc := &mockModelSvc{removeAssemblyFromModule: func(moduleID, connID string) error {
		gotModule, gotConn = moduleID, connID
		return nil
	}}
	var buf bytes.Buffer
	if err := runModuleRemoveAssembly(&buf, svc, "mod-1", "conn-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotModule != "mod-1" || gotConn != "conn-1" {
		t.Fatalf("unexpected forwarded args: module=%q conn=%q", gotModule, gotConn)
	}
}

/// @brief  runModuleSubmit soumet le module et affiche son nouveau statut
/// @input  id="mod-1", note="v1"
/// @expect SubmitModule reçoit id et note, sortie contient le statut "submitted"
func TestRunModuleSubmit_NominalCase(t *testing.T) {
	var gotID, gotNote string
	svc := &mockModelSvc{submitModule: func(moduleID, note string) (*model.Model3D, error) {
		gotID, gotNote = moduleID, note
		return &model.Model3D{ID: moduleID, Status: model.ModuleSubmitted}, nil
	}}
	var buf bytes.Buffer
	if err := runModuleSubmit(&buf, svc, "mod-1", "v1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "mod-1" || gotNote != "v1" {
		t.Fatalf("unexpected forwarded args: id=%q note=%q", gotID, gotNote)
	}
	if !strings.Contains(buf.String(), "submitted") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModuleSubmit propage l'erreur RM17 (aucun assemblage)
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunModuleSubmit_NoAssembly_Rejected(t *testing.T) {
	wantErr := errors.New("le module n'a aucun assemblage")
	svc := &mockModelSvc{submitModule: func(string, string) (*model.Model3D, error) { return nil, wantErr }}
	var buf bytes.Buffer
	err := runModuleSubmit(&buf, svc, "mod-1", "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModuleRemove retire un module non soumis
/// @input  id="mod-1"
/// @expect RemoveModule reçoit l'id, sortie confirme le retrait
func TestRunModuleRemove_NominalCase(t *testing.T) {
	removed := ""
	svc := &mockModelSvc{removeModule: func(id string) error { removed = id; return nil }}
	var buf bytes.Buffer
	if err := runModuleRemove(&buf, svc, "mod-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != "mod-1" {
		t.Fatalf("expected RemoveModule called with mod-1, got %q", removed)
	}
}
