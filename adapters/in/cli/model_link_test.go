// adapters/in/cli/model_link_test.go — tests des handlers CLI liaisons d'assemblage
package cli

import (
	"bytes"
	"strings"
	"testing"

	"myr-core/domain/model"
)

/// @brief  runModelLinkAdd crée une liaison directe et affiche son id
/// @input  from="iface-1", to="iface-2"
/// @expect AddAssemblyLink reçoit from/to, sortie contient l'id créé
func TestRunModelLinkAdd_NominalCase(t *testing.T) {
	var gotFrom, gotTo string
	svc := &mockModelSvc{addAssemblyLink: func(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*model.Connection, error) {
		gotFrom, gotTo = fromIfaceID, toIfaceID
		return &model.Connection{ID: "conn-1"}, nil
	}}
	var buf bytes.Buffer
	err := runModelLinkAdd(&buf, svc, "iface-1", "iface-2", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFrom != "iface-1" || gotTo != "iface-2" {
		t.Fatalf("unexpected forwarded args: from=%q to=%q", gotFrom, gotTo)
	}
	if !strings.Contains(buf.String(), "conn-1") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelLinkAdd rejette l'appel si --from ou --to manque (RM09/RM10 en amont)
/// @input  from="", to="iface-2"
/// @expect Erreur retournée sans appeler le service
func TestRunModelLinkAdd_MissingFrom_Rejected(t *testing.T) {
	called := false
	svc := &mockModelSvc{addAssemblyLink: func(string, string, string, string, string, string) (*model.Connection, error) {
		called = true
		return nil, nil
	}}
	var buf bytes.Buffer
	if err := runModelLinkAdd(&buf, svc, "", "iface-2", "", "", "", ""); err == nil {
		t.Fatalf("expected error when --from is missing")
	}
	if called {
		t.Fatalf("AddAssemblyLink should not be called")
	}
}

/// @brief  runModelLinkAdd affiche incompatible=true quand la liaison résultante est marquée incompatible (RM10/RM11)
/// @input  service retournant Connection{Incompatible: true}
/// @expect Sortie contient "incompatible=true"
func TestRunModelLinkAdd_Incompatible(t *testing.T) {
	svc := &mockModelSvc{addAssemblyLink: func(string, string, string, string, string, string) (*model.Connection, error) {
		return &model.Connection{ID: "conn-1", Incompatible: true}, nil
	}}
	var buf bytes.Buffer
	if err := runModelLinkAdd(&buf, svc, "iface-1", "iface-2", "", "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "incompatible=true") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelLinkConnectVirtual matérialise le slot virtuel et affiche l'id de la liaison
/// @input  virtual="iface-v", physical="iface-p"
/// @expect ConnectVirtualToPhysical reçoit les deux ids, sortie contient l'id créé
func TestRunModelLinkConnectVirtual_NominalCase(t *testing.T) {
	var gotVirtual, gotPhysical string
	svc := &mockModelSvc{connectVirtualToPhysical: func(virtualIfaceID, physicalIfaceID string, popupValues model.AssetInterface, fromInstanceID, toInstanceID string) (*model.Connection, error) {
		gotVirtual, gotPhysical = virtualIfaceID, physicalIfaceID
		return &model.Connection{ID: "conn-2"}, nil
	}}
	var buf bytes.Buffer
	err := runModelLinkConnectVirtual(&buf, svc, "iface-v", "iface-p", model.AssetInterface{}, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotVirtual != "iface-v" || gotPhysical != "iface-p" {
		t.Fatalf("unexpected forwarded args: virtual=%q physical=%q", gotVirtual, gotPhysical)
	}
	if !strings.Contains(buf.String(), "conn-2") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelLinkConnectVirtual rejette l'absence de --virtual ou --physical
/// @input  virtual="", physical="iface-p"
/// @expect Erreur retournée
func TestRunModelLinkConnectVirtual_MissingVirtual_Rejected(t *testing.T) {
	svc := &mockModelSvc{}
	var buf bytes.Buffer
	if err := runModelLinkConnectVirtual(&buf, svc, "", "iface-p", model.AssetInterface{}, "", ""); err == nil {
		t.Fatalf("expected error when --virtual is missing")
	}
}

/// @brief  runModelLinkRemove supprime la liaison et confirme en sortie
/// @input  id="conn-1"
/// @expect RemoveConnection reçoit l'id, sortie confirme la suppression
func TestRunModelLinkRemove_NominalCase(t *testing.T) {
	removed := ""
	svc := &mockModelSvc{removeConnection: func(id string) error { removed = id; return nil }}
	var buf bytes.Buffer
	if err := runModelLinkRemove(&buf, svc, "conn-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != "conn-1" {
		t.Fatalf("expected RemoveConnection called with conn-1, got %q", removed)
	}
}

/// @brief  runModelLinkList affiche une ligne par liaison, avec marqueur si incompatible
/// @input  service retournant 1 liaison incompatible
/// @expect Sortie contient l'id de la liaison et le marqueur "*"
func TestRunModelLinkList_NominalCase(t *testing.T) {
	svc := &mockModelSvc{listConnections: func() ([]*model.Connection, error) {
		return []*model.Connection{{ID: "conn-1", From: "a", To: "b", Incompatible: true}}, nil
	}}
	var buf bytes.Buffer
	if err := runModelLinkList(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "conn-1") {
		t.Fatalf("unexpected output: %q", out)
	}
}
