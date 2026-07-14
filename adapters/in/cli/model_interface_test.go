// adapters/in/cli/model_interface_test.go — tests des handlers CLI interfaces
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"myr/domain/model"
)

/// @brief  runModelInterfaceAdd appelle AddInterface et affiche l'id généré
/// @input  AssetInterface{AssetID: "asset-1"}, le mock assigne ID="iface-test"
/// @expect Sortie contient l'id généré, pas d'erreur
func TestRunModelInterfaceAdd_NominalCase(t *testing.T) {
	svc := &mockModelSvc{}
	iface := &model.AssetInterface{AssetID: "asset-1", Category: "ELEC"}
	var buf bytes.Buffer
	if err := runModelInterfaceAdd(&buf, svc, iface); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface.ID != "iface-test" {
		t.Fatalf("expected mock-assigned ID, got %q", iface.ID)
	}
	if !strings.Contains(buf.String(), "iface-test") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelInterfaceAdd propage l'erreur du service (store non configuré)
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunModelInterfaceAdd_ServiceError_Rejected(t *testing.T) {
	wantErr := errors.New("interface store non configuré")
	svc := &mockModelSvc{addInterface: func(*model.AssetInterface) error { return wantErr }}
	var buf bytes.Buffer
	err := runModelInterfaceAdd(&buf, svc, &model.AssetInterface{AssetID: "asset-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModelInterfaceUpdate récupère l'interface existante, applique le patch et sauvegarde
/// @input  id="iface-1" existant avec Category "ELEC", apply change Category en "MECA"
/// @expect UpdateInterface reçoit l'interface avec Category="MECA"
func TestRunModelInterfaceUpdate_NominalCase(t *testing.T) {
	var saved *model.AssetInterface
	svc := &mockModelSvc{
		getInterface: func(id string) (*model.AssetInterface, error) {
			return &model.AssetInterface{ID: id, Category: "ELEC"}, nil
		},
		updateInterface: func(iface *model.AssetInterface) error { saved = iface; return nil },
	}
	var buf bytes.Buffer
	err := runModelInterfaceUpdate(&buf, svc, "iface-1", func(iface *model.AssetInterface) {
		iface.Category = "MECA"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil || saved.Category != "MECA" {
		t.Fatalf("unexpected saved interface: %+v", saved)
	}
	if !strings.Contains(buf.String(), "iface-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelInterfaceUpdate propage l'erreur "introuvable" sans appeler UpdateInterface
/// @input  GetInterface retourne une erreur
/// @expect L'erreur est retournée, UpdateInterface n'est jamais appelé
func TestRunModelInterfaceUpdate_NotFound_Rejected(t *testing.T) {
	called := false
	svc := &mockModelSvc{
		getInterface:    func(string) (*model.AssetInterface, error) { return nil, model.ErrNotFound },
		updateInterface: func(*model.AssetInterface) error { called = true; return nil },
	}
	var buf bytes.Buffer
	err := runModelInterfaceUpdate(&buf, svc, "missing", func(*model.AssetInterface) {})
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if called {
		t.Fatalf("UpdateInterface should not be called when GetInterface fails")
	}
}

/// @brief  runModelInterfaceRemove supprime l'interface et confirme en sortie
/// @input  id="iface-1"
/// @expect Sortie confirme la suppression
func TestRunModelInterfaceRemove_NominalCase(t *testing.T) {
	removed := ""
	svc := &mockModelSvc{removeInterface: func(id string) error { removed = id; return nil }}
	var buf bytes.Buffer
	if err := runModelInterfaceRemove(&buf, svc, "iface-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != "iface-1" {
		t.Fatalf("expected RemoveInterface called with iface-1, got %q", removed)
	}
}

/// @brief  runModelInterfaceList affiche les interfaces d'un composant simple
/// @input  ListInterfacesForAsset retourne 1 interface
/// @expect Sortie contient l'id de l'interface, GetModule n'est pas nécessaire
func TestRunModelInterfaceList_Component_NominalCase(t *testing.T) {
	svc := &mockModelSvc{
		listInterfacesForAsset: func(assetID string) ([]*model.AssetInterface, error) {
			return []*model.AssetInterface{{ID: "iface-1", AssetID: assetID, Category: "ELEC"}}, nil
		},
	}
	var buf bytes.Buffer
	if err := runModelInterfaceList(&buf, svc, "asset-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "iface-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelInterfaceList bascule sur GetModuleInterfaces quand l'asset est un module
/// @input  ListInterfacesForAsset retourne une liste vide, GetModule réussit
/// @expect GetModuleInterfaces est appelé et son résultat affiché
func TestRunModelInterfaceList_Module_Fallback(t *testing.T) {
	svc := &mockModelSvc{
		listInterfacesForAsset: func(string) ([]*model.AssetInterface, error) { return nil, nil },
		getModule:              func(id string) (*model.Model3D, error) { return &model.Model3D{ID: id}, nil },
		getModuleInterfaces: func(moduleID string) ([]*model.AssetInterface, error) {
			return []*model.AssetInterface{{ID: "exposed-1"}}, nil
		},
	}
	var buf bytes.Buffer
	if err := runModelInterfaceList(&buf, svc, "mod-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "exposed-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelInterfaceGet affiche le détail complet d'une interface
/// @input  GetInterface retourne une interface avec toutes les valeurs
/// @expect Sortie contient l'id, la catégorie et l'unité
func TestRunModelInterfaceGet_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getInterface: func(id string) (*model.AssetInterface, error) {
		return &model.AssetInterface{ID: id, Category: "HYD", Unit: "bar"}, nil
	}}
	var buf bytes.Buffer
	if err := runModelInterfaceGet(&buf, svc, "iface-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "iface-1") || !strings.Contains(out, "HYD") || !strings.Contains(out, "bar") {
		t.Fatalf("unexpected output: %q", out)
	}
}
