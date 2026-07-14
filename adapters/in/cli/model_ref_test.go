// adapters/in/cli/model_ref_test.go — tests des handlers CLI vocabulaire de référence
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"myr/domain/model"
)

/// @brief  runModelRefList affiche chaque catégorie avec ses types et unités
/// @input  service retournant le vocabulaire par défaut
/// @expect Sortie contient "ELEC" et "MECA"
func TestRunModelRefList_NominalCase(t *testing.T) {
	svc := &mockModelSvc{getRefs: func() (*model.InterfaceRefs, error) { return model.DefaultInterfaceRefs(), nil }}
	var buf bytes.Buffer
	if err := runModelRefList(&buf, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ELEC") || !strings.Contains(out, "MECA") {
		t.Fatalf("unexpected output: %q", out)
	}
}

/// @brief  runModelRefAddCategory transmet la catégorie au service et confirme en sortie
/// @input  cat="PNEU"
/// @expect AddRefCategory reçoit "PNEU", sortie confirme l'ajout
func TestRunModelRefAddCategory_NominalCase(t *testing.T) {
	var got string
	svc := &mockModelSvc{addRefCategory: func(cat string) error { got = cat; return nil }}
	var buf bytes.Buffer
	if err := runModelRefAddCategory(&buf, svc, "PNEU"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "PNEU" {
		t.Fatalf("expected AddRefCategory called with PNEU, got %q", got)
	}
	if !strings.Contains(buf.String(), "PNEU") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

/// @brief  runModelRefAddType propage l'erreur si la catégorie n'existe pas
/// @input  service retournant une erreur
/// @expect L'erreur est retournée
func TestRunModelRefAddType_UnknownCategory_Rejected(t *testing.T) {
	wantErr := errors.New("catégorie inconnue")
	svc := &mockModelSvc{addRefType: func(string, string) error { return wantErr }}
	var buf bytes.Buffer
	err := runModelRefAddType(&buf, svc, "UNKNOWN", "Foo")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

/// @brief  runModelRefAddUnit transmet catégorie et unité au service
/// @input  cat="ELEC", unit="dBm"
/// @expect AddRefUnit reçoit les deux valeurs, sortie confirme l'ajout
func TestRunModelRefAddUnit_NominalCase(t *testing.T) {
	var gotCat, gotUnit string
	svc := &mockModelSvc{addRefUnit: func(cat, unit string) error { gotCat, gotUnit = cat, unit; return nil }}
	var buf bytes.Buffer
	if err := runModelRefAddUnit(&buf, svc, "ELEC", "dBm"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCat != "ELEC" || gotUnit != "dBm" {
		t.Fatalf("unexpected forwarded args: cat=%q unit=%q", gotCat, gotUnit)
	}
}
