// adapters/in/cli/model_license_test.go — tests des handlers CLI licences
package cli

import (
	"bytes"
	"strings"
	"testing"

	"myr/domain/model"
)

/// @brief  runModelLicenseList affiche une ligne par licence du catalogue
/// @input  service retournant 2 licences
/// @expect Les deux ID apparaissent dans la sortie
func TestRunModelLicenseList_NominalCase(t *testing.T) {
	svc := &mockModelSvc{listLicenses: func() []*model.License {
		return []*model.License{{ID: "mit", Name: "MIT"}, {ID: "proprietary", Name: "Tous droits réservés"}}
	}}
	var buf bytes.Buffer
	if err := runModelLicenseList(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "mit") || !strings.Contains(out, "proprietary") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runModelLicenseGet affiche le détail d'une licence
/// @input  id="mit", service retournant License{ID: "mit", Permissions: [...]}
/// @expect Sortie contient l'id et les permissions
func TestRunModelLicenseGet_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getLicense: func(id string) (*model.License, error) {
		return &model.License{ID: id, Name: "MIT", Permissions: []string{"commercial-use", "modifications"}}, nil
	}}
	var buf bytes.Buffer
	if err := runModelLicenseGet(&buf, svc, "mit"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "mit") || !strings.Contains(out, "commercial-use") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runModelLicenseCheck avec --parent appelle CheckLicenseCompatibility et affiche OK
/// @input  parent="mit", proposed="cc-by-4.0", service retournant Compatible: true
/// @expect Sortie contient "OK"
func TestRunModelLicenseCheck_Parent_Compatible(t *testing.T) {
	var gotParent, gotProposed string
	svc := &mockModelSvc{checkLicenseCompatibility: func(parentLicenseID, proposedLicenseID string) *model.LicenseCheck {
		gotParent, gotProposed = parentLicenseID, proposedLicenseID
		return &model.LicenseCheck{Compatible: true}
	}}
	var buf bytes.Buffer
	err := runModelLicenseCheck(&buf, svc, "mit", nil, "cc-by-4.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParent != "mit" || gotProposed != "cc-by-4.0" {
		t.Fatalf("unexpected forwarded args: parent=%q proposed=%q", gotParent, gotProposed)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Fatalf("expected OK, got: %q", buf.String())
	}
}

/// @brief  runModelLicenseCheck avec --component (répété) appelle CheckModuleLicenseCompatibility
/// @input  components=["mit","cc-by-4.0"], proposed="proprietary", service retournant Compatible: false
/// @expect CheckModuleLicenseCompatibility est appelé (pas CheckLicenseCompatibility), sortie contient "KO"
func TestRunModelLicenseCheck_Components_Incompatible(t *testing.T) {
	calledSingle := false
	svc := &mockModelSvc{
		checkLicenseCompatibility: func(string, string) *model.LicenseCheck {
			calledSingle = true
			return &model.LicenseCheck{Compatible: true}
		},
		checkModuleLicenseCompatibility: func(componentLicenseIDs []string, proposedProductLicenseID string) *model.LicenseCheck {
			return &model.LicenseCheck{Compatible: false, Reason: "licence propriétaire incompatible"}
		},
	}
	var buf bytes.Buffer
	err := runModelLicenseCheck(&buf, svc, "", []string{"mit", "cc-by-4.0"}, "proprietary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calledSingle {
		t.Fatalf("CheckLicenseCompatibility should not be called when --component is used")
	}
	if !strings.Contains(buf.String(), "KO") {
		t.Fatalf("expected KO, got: %q", buf.String())
	}
}

/// @brief  runModelLicenseCheck rejette l'absence de --parent et --component
/// @input  parentLicenseID="", componentLicenseIDs=nil, proposed="mit"
/// @expect Une erreur est retournée sans appeler le service
func TestRunModelLicenseCheck_NoParentNoComponent_Rejected(t *testing.T) {
	svc := &mockModelSvc{}
	var buf bytes.Buffer
	if err := runModelLicenseCheck(&buf, svc, "", nil, "mit"); err == nil {
		t.Fatalf("expected error when neither --parent nor --component is given")
	}
}
