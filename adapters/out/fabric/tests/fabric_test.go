// adapters/out/fabric/fabric_test.go
// Tests unitaires des adapters Fabric (FabricAdapter et FabricBlockchain).
// Aucune connexion réseau requise : *client.Contract et *GatewayClient sont
// remplacés par des stubs implémentant contractCaller / gatewayProvider.
package fabric_test

import (
	"encoding/json"
	"errors"
	"myr/adapters/out/fabric"
	"testing"

	"myr/domain/model"
)

// ── Stubs ─────────────────────────────────────────────────────────────────────

// stubContract implémente contractCaller.
// Les fonctions submitFn / evaluateFn sont configurables par test.
type stubContract struct {
	submitFn   func(name string, args ...string) ([]byte, error)
	evaluateFn func(name string, args ...string) ([]byte, error)
	// derniers appels enregistrés pour assertions
	lastSubmitName   string
	lastSubmitArgs   []string
	lastEvaluateName string
	lastEvaluateArgs []string
}

func (s *stubContract) SubmitTransaction(name string, args ...string) ([]byte, error) {
	s.lastSubmitName = name
	s.lastSubmitArgs = args
	if s.submitFn != nil {
		return s.submitFn(name, args...)
	}
	return nil, nil
}

func (s *stubContract) EvaluateTransaction(name string, args ...string) ([]byte, error) {
	s.lastEvaluateName = name
	s.lastEvaluateArgs = args
	if s.evaluateFn != nil {
		return s.evaluateFn(name, args...)
	}
	return nil, nil
}

// stubGateway implémente gatewayProvider.
// Retourne toujours le même stubContract ; enregistre le channelName passé à ContractFor.
type stubGateway struct {
	contract            *stubContract
	lastContractForChan string
}

func (g *stubGateway) Contract() fabric.ContractCaller {
	return g.contract
}

func (g *stubGateway) ContractFor(channelName string) fabric.ContractCaller {
	g.lastContractForChan = channelName
	return g.contract
}

// newAsset construit un Model3D minimal pour les tests.
func newAsset(id, channelID string) *model.Model3D {
	return &model.Model3D{
		ID:        id,
		Name:      "Test Asset",
		ChannelID: channelID,
		Hash:      "sha256:abc123",
	}
}

// ── FabricAdapter ─────────────────────────────────────────────────────────────
// FabricAdapter.StoreModelRecord appelle SubmitTransaction("CreateModel", <json>)

// / @brief  Vérifie que StoreModelRecord soumet correctement la transaction CreateModel au chaincode
// / @details Chemin nominal : l'asset est sérialisé en JSON et transmis à SubmitTransaction sans erreur.
// / @input  stubContract dont SubmitTransaction retourne (nil, nil) ; asset Model3D id="a1", channelID="ch1"
// / @expect retourne nil ; lastSubmitName == "CreateModel" ; l'argument JSON décode un Model3D avec ID "a1"
func TestFabricAdapter_StoreModelRecord_Success(t *testing.T) {
	stub := &stubContract{}
	adapter := fabric.NewFabricAdapter(stub)

	asset := newAsset("a1", "ch1")
	if err := adapter.StoreModelRecord(asset); err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastSubmitName != "CreateModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "CreateModel", stub.lastSubmitName)
	}
	if len(stub.lastSubmitArgs) != 1 {
		t.Fatalf("1 argument JSON attendu, obtenu %d", len(stub.lastSubmitArgs))
	}
	// Vérifier que l'argument est du JSON valide contenant l'ID
	var decoded model.Model3D
	if err := json.Unmarshal([]byte(stub.lastSubmitArgs[0]), &decoded); err != nil {
		t.Fatalf("l'argument n'est pas du JSON valide : %v", err)
	}
	if decoded.ID != "a1" {
		t.Errorf("ID attendu %q dans le JSON, obtenu %q", "a1", decoded.ID)
	}
}

// / @brief  Vérifie que StoreModelRecord propage l'erreur retournée par SubmitTransaction
// / @input  stubContract dont submitFn retourne errors.New("fabric: peer indisponible")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricAdapter_StoreModelRecord_SubmitError(t *testing.T) {
	expectedErr := errors.New("fabric: peer indisponible")
	stub := &stubContract{
		submitFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	adapter := fabric.NewFabricAdapter(stub)

	if err := adapter.StoreModelRecord(newAsset("a1", "ch1")); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// FabricAdapter.GetModelRecord appelle EvaluateTransaction("GetModel", id)

// / @brief  Vérifie que GetModelRecord évalue correctement la transaction GetModel et décode le JSON retourné
// / @details Chemin nominal : le payload JSON est désérialisé et le Model3D résultant correspond à l'asset attendu.
// / @input  stubContract dont evaluateFn retourne le JSON de l'asset id="a2" ; appel GetModelRecord("a2", "")
// / @expect retourne nil err ; lastEvaluateName == "GetModel" ; lastEvaluateArgs[0] == "a2" ; got.ID == "a2"
func TestFabricAdapter_GetModelRecord_Success(t *testing.T) {
	want := newAsset("a2", "ch1")
	payload, _ := json.Marshal(want)
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return payload, nil },
	}
	adapter := fabric.NewFabricAdapter(stub)

	got, err := adapter.GetModelRecord("a2", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "GetModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "GetModel", stub.lastEvaluateName)
	}
	if len(stub.lastEvaluateArgs) < 1 || stub.lastEvaluateArgs[0] != "a2" {
		t.Errorf("argument ID attendu %q, obtenu %v", "a2", stub.lastEvaluateArgs)
	}
	if got.ID != want.ID {
		t.Errorf("ID attendu %q, obtenu %q", want.ID, got.ID)
	}
}

// / @brief  Vérifie que GetModelRecord propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: timeout")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricAdapter_GetModelRecord_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: timeout")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	adapter := fabric.NewFabricAdapter(stub)

	if _, err := adapter.GetModelRecord("a2", ""); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// / @brief  Vérifie que GetModelRecord retourne un Model3D vide sans erreur lorsque le JSON est invalide
// / @details FabricAdapter absorbe silencieusement l'erreur d'unmarshal (comportement legacy).
// / @input  stubContract dont evaluateFn retourne []byte("{invalid}") ; appel GetModelRecord("a2", "")
// / @expect retourne nil err ; got.ID == "" (Model3D vide)
func TestFabricAdapter_GetModelRecord_InvalidJSON_ReturnsEmptyModel(t *testing.T) {
	// json.Unmarshal est appelé sans vérification d'erreur dans FabricAdapter :
	// un payload invalide retourne un Model3D vide plutôt qu'une erreur.
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("{invalid}"), nil },
	}
	adapter := fabric.NewFabricAdapter(stub)

	got, err := adapter.GetModelRecord("a2", "")
	if err != nil {
		t.Fatalf("aucune erreur attendue (unmarshal silencieux), obtenu %v", err)
	}
	if got.ID != "" {
		t.Errorf("ID vide attendu sur JSON invalide, obtenu %q", got.ID)
	}
}

// FabricAdapter.ListModelRecords appelle EvaluateTransaction("ListModels", channelID)

// / @brief  Vérifie que ListModelRecords évalue ListModels avec le bon channelID et décode la liste retournée
// / @details Chemin nominal : le payload JSON d'un tableau de deux assets est désérialisé correctement.
// / @input  stubContract dont evaluateFn retourne le JSON de [a1, a2] ; appel ListModelRecords("ch1")
// / @expect retourne nil err ; lastEvaluateName == "ListModels" ; lastEvaluateArgs[0] == "ch1" ; len(got) == 2
func TestFabricAdapter_ListModelRecords_Success(t *testing.T) {
	list := []*model.Model3D{newAsset("a1", "ch1"), newAsset("a2", "ch1")}
	payload, _ := json.Marshal(list)
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return payload, nil },
	}
	adapter := fabric.NewFabricAdapter(stub)

	got, err := adapter.ListModelRecords("ch1")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "ListModels" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "ListModels", stub.lastEvaluateName)
	}
	if len(stub.lastEvaluateArgs) < 1 || stub.lastEvaluateArgs[0] != "ch1" {
		t.Errorf("argument channelID attendu %q, obtenu %v", "ch1", stub.lastEvaluateArgs)
	}
	if len(got) != 2 {
		t.Errorf("2 assets attendus, obtenu %d", len(got))
	}
}

// / @brief  Vérifie que ListModelRecords propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: canal introuvable")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricAdapter_ListModelRecords_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: canal introuvable")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	adapter := fabric.NewFabricAdapter(stub)

	if _, err := adapter.ListModelRecords("ch1"); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// FabricAdapter.VerifyIntegrity appelle EvaluateTransaction("VerifyModel", id, hash)

// / @brief  Vérifie que VerifyIntegrity retourne true lorsque le chaincode confirme l'intégrité
// / @input  stubContract dont evaluateFn retourne []byte("true") ; appel VerifyIntegrity("a1", "sha256:abc", "")
// / @expect retourne (true, nil) ; lastEvaluateName == "VerifyModel" ; args == ["a1", "sha256:abc"]
func TestFabricAdapter_VerifyIntegrity_True(t *testing.T) {
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("true"), nil },
	}
	adapter := fabric.NewFabricAdapter(stub)

	ok, err := adapter.VerifyIntegrity("a1", "sha256:abc", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "VerifyModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "VerifyModel", stub.lastEvaluateName)
	}
	if len(stub.lastEvaluateArgs) < 2 || stub.lastEvaluateArgs[0] != "a1" || stub.lastEvaluateArgs[1] != "sha256:abc" {
		t.Errorf("arguments (id, hash) incorrects : %v", stub.lastEvaluateArgs)
	}
	if !ok {
		t.Error("true attendu, obtenu false")
	}
}

// / @brief  Vérifie que VerifyIntegrity retourne false lorsque le chaincode infirme l'intégrité
// / @input  stubContract dont evaluateFn retourne []byte("false") ; appel VerifyIntegrity("a1", "sha256:bad", "")
// / @expect retourne (false, nil)
func TestFabricAdapter_VerifyIntegrity_False(t *testing.T) {
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("false"), nil },
	}
	adapter := fabric.NewFabricAdapter(stub)

	ok, err := adapter.VerifyIntegrity("a1", "sha256:bad", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if ok {
		t.Error("false attendu, obtenu true")
	}
}

// / @brief  Vérifie que VerifyIntegrity propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: endorsement failed")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricAdapter_VerifyIntegrity_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: endorsement failed")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	adapter := fabric.NewFabricAdapter(stub)

	if _, err := adapter.VerifyIntegrity("a1", "sha256:abc", ""); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// ── FabricBlockchain ──────────────────────────────────────────────────────────
// FabricBlockchain.StoreModelRecord appelle ContractFor(channelID).SubmitTransaction("StoreModel", <json>)
// Note : la fonction chaincode est "StoreModel" (≠ "CreateModel" utilisé par FabricAdapter).

// / @brief  Vérifie que StoreModelRecord route vers le bon canal et soumet StoreModel avec le JSON de l'asset
// / @details Chemin nominal : ContractFor est appelé avec channelID de l'asset ; le JSON soumis est valide.
// / @input  stubGateway wrappant un stubContract vide ; asset Model3D id="b1", channelID="chan-alpha"
// / @expect retourne nil ; lastSubmitName == "StoreModel" ; lastContractForChan == "chan-alpha" ; JSON décodé avec ID "b1"
func TestFabricBlockchain_StoreModelRecord_Success(t *testing.T) {
	stub := &stubContract{}
	gw := &stubGateway{contract: stub}
	bc := fabric.NewFabricBlockchain(gw)

	asset := newAsset("b1", "chan-alpha")
	if err := bc.StoreModelRecord(asset); err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastSubmitName != "StoreModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "StoreModel", stub.lastSubmitName)
	}
	if gw.lastContractForChan != "chan-alpha" {
		t.Errorf("canal attendu %q dans ContractFor, obtenu %q", "chan-alpha", gw.lastContractForChan)
	}
	// Vérifier que l'argument est le JSON de l'asset
	if len(stub.lastSubmitArgs) != 1 {
		t.Fatalf("1 argument JSON attendu, obtenu %d", len(stub.lastSubmitArgs))
	}
	var decoded model.Model3D
	if err := json.Unmarshal([]byte(stub.lastSubmitArgs[0]), &decoded); err != nil {
		t.Fatalf("l'argument n'est pas du JSON valide : %v", err)
	}
	if decoded.ID != "b1" {
		t.Errorf("ID attendu %q dans le JSON, obtenu %q", "b1", decoded.ID)
	}
}

// / @brief  Vérifie que StoreModelRecord propage l'erreur retournée par SubmitTransaction
// / @input  stubContract dont submitFn retourne errors.New("fabric: submit rejected")
// / @expect retourne une erreur non nil wrappant expectedErr via errors.Is
func TestFabricBlockchain_StoreModelRecord_SubmitError(t *testing.T) {
	expectedErr := errors.New("fabric: submit rejected")
	stub := &stubContract{
		submitFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	err := bc.StoreModelRecord(newAsset("b1", "ch1"))
	if err == nil {
		t.Fatal("une erreur était attendue")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// FabricBlockchain.GetModelRecord appelle Contract().EvaluateTransaction("GetModel", id)

// / @brief  Vérifie que GetModelRecord évalue GetModel et décode correctement le Model3D retourné
// / @details Chemin nominal : ID et Hash de l'asset sont conservés après désérialisation.
// / @input  stubContract dont evaluateFn retourne le JSON de l'asset id="b2" ; appel GetModelRecord("b2", "")
// / @expect retourne nil err ; lastEvaluateName == "GetModel" ; got.ID == "b2" et got.Hash correspondent
func TestFabricBlockchain_GetModelRecord_Success(t *testing.T) {
	want := newAsset("b2", "ch1")
	payload, _ := json.Marshal(want)
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return payload, nil },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	got, err := bc.GetModelRecord("b2", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "GetModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "GetModel", stub.lastEvaluateName)
	}
	if len(stub.lastEvaluateArgs) < 1 || stub.lastEvaluateArgs[0] != "b2" {
		t.Errorf("argument ID attendu %q, obtenu %v", "b2", stub.lastEvaluateArgs)
	}
	if got.ID != want.ID {
		t.Errorf("ID attendu %q, obtenu %q", want.ID, got.ID)
	}
	if got.Hash != want.Hash {
		t.Errorf("Hash attendu %q, obtenu %q", want.Hash, got.Hash)
	}
}

// / @brief  Vérifie que GetModelRecord propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: peer timeout")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricBlockchain_GetModelRecord_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: peer timeout")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	if _, err := bc.GetModelRecord("b2", ""); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// / @brief  Vérifie que GetModelRecord retourne une erreur lorsque le JSON retourné est invalide
// / @details Contrairement à FabricAdapter, FabricBlockchain propage l'erreur d'unmarshal.
// / @input  stubContract dont evaluateFn retourne []byte("{bad json}") ; appel GetModelRecord("b2", "")
// / @expect retourne une erreur non nil (erreur d'unmarshal JSON)
func TestFabricBlockchain_GetModelRecord_InvalidJSON_ReturnsError(t *testing.T) {
	// FabricBlockchain.GetModelRecord retourne l'erreur d'unmarshal (contrairement à FabricAdapter).
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("{bad json}"), nil },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	if _, err := bc.GetModelRecord("b2", ""); err == nil {
		t.Error("une erreur d'unmarshal était attendue")
	}
}

// FabricBlockchain.ListModelRecords appelle ContractFor(channelID).EvaluateTransaction("ListModels", channelID)

// / @brief  Vérifie que ListModelRecords route vers le bon canal, évalue ListModels et décode la liste
// / @details Chemin nominal : ContractFor et EvaluateTransaction reçoivent tous deux channelID="ch2".
// / @input  stubGateway + stubContract dont evaluateFn retourne le JSON de [b1, b2] ; appel ListModelRecords("ch2")
// / @expect retourne nil err ; lastEvaluateName == "ListModels" ; lastContractForChan == "ch2" ; len(got) == 2
func TestFabricBlockchain_ListModelRecords_Success(t *testing.T) {
	list := []*model.Model3D{newAsset("b1", "ch2"), newAsset("b2", "ch2")}
	payload, _ := json.Marshal(list)
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return payload, nil },
	}
	gw := &stubGateway{contract: stub}
	bc := fabric.NewFabricBlockchain(gw)

	got, err := bc.ListModelRecords("ch2")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "ListModels" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "ListModels", stub.lastEvaluateName)
	}
	// Le canal et l'argument de query doivent correspondre à channelID
	if gw.lastContractForChan != "ch2" {
		t.Errorf("canal attendu %q dans ContractFor, obtenu %q", "ch2", gw.lastContractForChan)
	}
	if len(stub.lastEvaluateArgs) < 1 || stub.lastEvaluateArgs[0] != "ch2" {
		t.Errorf("argument channelID attendu %q dans EvaluateTransaction, obtenu %v", "ch2", stub.lastEvaluateArgs)
	}
	if len(got) != 2 {
		t.Errorf("2 assets attendus, obtenu %d", len(got))
	}
}

// / @brief  Vérifie que ListModelRecords propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: canal ch3 inconnu")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricBlockchain_ListModelRecords_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: canal ch3 inconnu")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	if _, err := bc.ListModelRecords("ch3"); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// FabricBlockchain.VerifyIntegrity appelle Contract().EvaluateTransaction("VerifyModel", id, hash)

// / @brief  Vérifie que VerifyIntegrity retourne true lorsque le chaincode confirme l'intégrité
// / @input  stubContract dont evaluateFn retourne []byte("true") ; appel VerifyIntegrity("b1", "sha256:xyz", "")
// / @expect retourne (true, nil) ; lastEvaluateName == "VerifyModel" ; args == ["b1", "sha256:xyz"]
func TestFabricBlockchain_VerifyIntegrity_True(t *testing.T) {
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("true"), nil },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	ok, err := bc.VerifyIntegrity("b1", "sha256:xyz", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if stub.lastEvaluateName != "VerifyModel" {
		t.Errorf("fonction chaincode attendue %q, obtenu %q", "VerifyModel", stub.lastEvaluateName)
	}
	if len(stub.lastEvaluateArgs) < 2 || stub.lastEvaluateArgs[0] != "b1" || stub.lastEvaluateArgs[1] != "sha256:xyz" {
		t.Errorf("arguments (id, hash) incorrects : %v", stub.lastEvaluateArgs)
	}
	if !ok {
		t.Error("true attendu, obtenu false")
	}
}

// / @brief  Vérifie que VerifyIntegrity retourne false lorsque le chaincode infirme l'intégrité
// / @input  stubContract dont evaluateFn retourne []byte("false") ; appel VerifyIntegrity("b1", "sha256:wrong", "")
// / @expect retourne (false, nil)
func TestFabricBlockchain_VerifyIntegrity_False(t *testing.T) {
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return []byte("false"), nil },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	ok, err := bc.VerifyIntegrity("b1", "sha256:wrong", "")
	if err != nil {
		t.Fatalf("inattendu : err = %v", err)
	}
	if ok {
		t.Error("false attendu, obtenu true")
	}
}

// / @brief  Vérifie que VerifyIntegrity propage l'erreur retournée par EvaluateTransaction
// / @input  stubContract dont evaluateFn retourne errors.New("fabric: endorsement policy failure")
// / @expect retourne l'erreur du chaincode via errors.Is
func TestFabricBlockchain_VerifyIntegrity_EvaluateError(t *testing.T) {
	expectedErr := errors.New("fabric: endorsement policy failure")
	stub := &stubContract{
		evaluateFn: func(_ string, _ ...string) ([]byte, error) { return nil, expectedErr },
	}
	bc := fabric.NewFabricBlockchain(&stubGateway{contract: stub})

	if _, err := bc.VerifyIntegrity("b1", "sha256:xyz", ""); !errors.Is(err, expectedErr) {
		t.Errorf("erreur attendue %v, obtenu %v", expectedErr, err)
	}
}

// ── Vérification de conformité aux ports ─────────────────────────────────────
// Ces assertions de compilation garantissent que les deux adapters implémentent
// bien model.BlockchainPort. Si ce n'est plus le cas, la compilation échoue.

var _ model.BlockchainPort = (*fabric.FabricAdapter)(nil)
var _ model.BlockchainPort = (*fabric.FabricBlockchain)(nil)
