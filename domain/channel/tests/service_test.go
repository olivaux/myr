package channel_test

import (
	"errors"
	"strings"
	"testing"

	"myr/domain/channel"
)

// --- stub in-memory repo ---

type memRepo struct {
	m map[string]*channel.Channel
}

func newMemRepo() *memRepo {
	return &memRepo{m: make(map[string]*channel.Channel)}
}

func (r *memRepo) add(ch *channel.Channel) {
	r.m[ch.ID] = ch
}

func (r *memRepo) FindByID(id string) (*channel.Channel, error) {
	return r.m[id], nil
}

func (r *memRepo) FindAll() ([]*channel.Channel, error) {
	list := make([]*channel.Channel, 0, len(r.m))
	for _, ch := range r.m {
		list = append(list, ch)
	}
	return list, nil
}

// --- tests ---

// / @brief  Récupération d'un canal existant par son ID
// / @input  memRepo avec un canal "greenchannel"
// / @expect retourne le canal avec l'ID correct, sans erreur
func TestGet(t *testing.T) {
	repo := newMemRepo()
	repo.add(&channel.Channel{ID: "greenchannel", Name: "greenchannel"})
	svc := channel.NewService(repo)

	got, err := svc.Get("greenchannel")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "greenchannel" {
		t.Errorf("ID: got %q, want %q", got.ID, "greenchannel")
	}
}

// / @brief  Récupération d'un canal inexistant retourne ErrNotFound
// / @input  memRepo vide, ID "inexistant"
// / @expect retourne channel.ErrNotFound
func TestGet_NotFound(t *testing.T) {
	svc := channel.NewService(newMemRepo())

	_, err := svc.Get("inexistant")
	if !errors.Is(err, channel.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// / @brief  Listage de tous les canaux disponibles
// / @input  memRepo avec deux canaux "a" et "b"
// / @expect retourne une liste de 2 éléments sans erreur
func TestList(t *testing.T) {
	repo := newMemRepo()
	repo.add(&channel.Channel{ID: "a", Name: "a"})
	repo.add(&channel.Channel{ID: "b", Name: "b"})
	svc := channel.NewService(repo)

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d canaux, want 2", len(list))
	}
}

// / @brief  Listage retourne une liste vide si aucun canal n'existe
// / @input  memRepo vide
// / @expect retourne une liste vide sans erreur
func TestList_Empty(t *testing.T) {
	svc := channel.NewService(newMemRepo())

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

// --- stub ChannelConfigPort (Pattern A : struct concrète avec état en mémoire) ---

type addNodeCall struct {
	channelID string
	nodeType  channel.NodeType
	addr      string
	orgMSP    string
	certs     channel.NodeCerts
}

type removeNodeCall struct {
	channelID string
	addr      string
}

type mockChannelConfig struct {
	addOrgErr     error
	addNodeErr    error
	removeNodeErr error

	addedOrgs    []channel.Organization
	addedOrgChan []string
	addedNodes   []addNodeCall
	removedNodes []removeNodeCall
}

func newMockChannelConfig() *mockChannelConfig {
	return &mockChannelConfig{}
}

func (m *mockChannelConfig) AddOrganisation(channelID string, org channel.Organization) error {
	m.addedOrgChan = append(m.addedOrgChan, channelID)
	m.addedOrgs = append(m.addedOrgs, org)
	return m.addOrgErr
}

func (m *mockChannelConfig) AddNode(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error {
	m.addedNodes = append(m.addedNodes, addNodeCall{channelID, nodeType, addr, orgMSP, certs})
	return m.addNodeErr
}

func (m *mockChannelConfig) RemoveNode(channelID, addr string) error {
	m.removedNodes = append(m.removedNodes, removeNodeCall{channelID, addr})
	return m.removeNodeErr
}

// --- AddOrganisation (UCADM01) ---

// / @brief  Ajout nominal d'une organisation valide à un canal Fabric
// / @input  fabricConf injecté sans erreur, MSPID "Org2MSP" valide
// / @expect aucune erreur, l'organisation est transmise au ChannelConfigPort avec le bon channelID
func TestAddOrganisation_NominalCase(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	org := channel.Organization{MSPID: "Org2MSP", Name: "Org2", Role: "member", RootCert: "PEM"}
	if err := svc.AddOrganisation("sandbox", org); err != nil {
		t.Fatalf("AddOrganisation: %v", err)
	}
	if len(mock.addedOrgs) != 1 || mock.addedOrgs[0].MSPID != "Org2MSP" {
		t.Errorf("organisation non transmise au ChannelConfigPort : %+v", mock.addedOrgs)
	}
	if len(mock.addedOrgChan) != 1 || mock.addedOrgChan[0] != "sandbox" {
		t.Errorf("channelID non transmis : %+v", mock.addedOrgChan)
	}
}

// / @brief  Rejet d'un MSP ID contenant des caractères non autorisés
// / @input  fabricConf injecté, MSPID "Org#2 MSP" (espace et caractère interdit)
// / @expect retourne ErrInvalidMSPID, le ChannelConfigPort n'est jamais appelé (RM07 : valider avant soumission)
func TestAddOrganisation_InvalidMSPID_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	org := channel.Organization{MSPID: "Org#2 MSP", Name: "Org2"}
	err := svc.AddOrganisation("sandbox", org)
	if !errors.Is(err, channel.ErrInvalidMSPID) {
		t.Errorf("expected ErrInvalidMSPID, got %v", err)
	}
	if len(mock.addedOrgs) != 0 {
		t.Error("le ChannelConfigPort ne doit pas être appelé si le MSPID est invalide")
	}
}

// / @brief  Rejet d'un MSP ID dépassant 128 caractères (limite RM Fabric)
// / @input  MSPID de 129 caractères
// / @expect retourne ErrInvalidMSPID
func TestAddOrganisation_RM_MSPIDTooLong_Rejected(t *testing.T) {
	svc := channel.NewService(newMemRepo()).WithFabricConfig(newMockChannelConfig())

	long := strings.Repeat("a", 129)
	err := svc.AddOrganisation("sandbox", channel.Organization{MSPID: long, Name: "Org2"})
	if !errors.Is(err, channel.ErrInvalidMSPID) {
		t.Errorf("expected ErrInvalidMSPID, got %v", err)
	}
}

// / @brief  Rejet d'un MSP ID vide
// / @input  MSPID ""
// / @expect retourne ErrInvalidMSPID
func TestAddOrganisation_RM_MSPIDEmpty_Rejected(t *testing.T) {
	svc := channel.NewService(newMemRepo()).WithFabricConfig(newMockChannelConfig())

	err := svc.AddOrganisation("sandbox", channel.Organization{MSPID: "", Name: "Org2"})
	if !errors.Is(err, channel.ErrInvalidMSPID) {
		t.Errorf("expected ErrInvalidMSPID, got %v", err)
	}
}

// / @brief  AddOrganisation refuse d'opérer si aucun adapter Fabric n'est configuré
// / @input  Service sans WithFabricConfig (DC-CLI-03 : mode sans Fabric)
// / @expect retourne ErrFabricUnavailable
func TestAddOrganisation_FabricUnavailable_Rejected(t *testing.T) {
	svc := channel.NewService(newMemRepo())

	err := svc.AddOrganisation("sandbox", channel.Organization{MSPID: "Org2MSP", Name: "Org2"})
	if !errors.Is(err, channel.ErrFabricUnavailable) {
		t.Errorf("expected ErrFabricUnavailable, got %v", err)
	}
}

// / @brief  AddOrganisation propage ErrAlreadyMember renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrAlreadyMember
// / @expect l'erreur ErrAlreadyMember remonte inchangée
func TestAddOrganisation_AlreadyMember_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.addOrgErr = channel.ErrAlreadyMember
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddOrganisation("sandbox", channel.Organization{MSPID: "Org2MSP", Name: "Org2"})
	if !errors.Is(err, channel.ErrAlreadyMember) {
		t.Errorf("expected ErrAlreadyMember, got %v", err)
	}
}

// / @brief  AddOrganisation propage ErrEndorsementPolicy renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrEndorsementPolicy
// / @expect l'erreur ErrEndorsementPolicy remonte inchangée
func TestAddOrganisation_EndorsementPolicy_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.addOrgErr = channel.ErrEndorsementPolicy
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddOrganisation("sandbox", channel.Organization{MSPID: "Org2MSP", Name: "Org2"})
	if !errors.Is(err, channel.ErrEndorsementPolicy) {
		t.Errorf("expected ErrEndorsementPolicy, got %v", err)
	}
}

// --- AddNode (UCADM03) ---

// / @brief  Ajout nominal d'un peer à un canal Fabric
// / @input  fabricConf injecté sans erreur, type peer, adresse host:port valide
// / @expect aucune erreur, le nœud est transmis au ChannelConfigPort avec les bons paramètres
func TestAddNode_Peer_NominalCase(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{TLSCert: "PEM"})
	if err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if len(mock.addedNodes) != 1 {
		t.Fatalf("expected 1 appel AddNode, got %d", len(mock.addedNodes))
	}
	call := mock.addedNodes[0]
	if call.nodeType != channel.NodeTypePeer || call.addr != "203.0.113.10:7051" || call.orgMSP != "Org2MSP" {
		t.Errorf("paramètres incorrects transmis au ChannelConfigPort : %+v", call)
	}
}

// / @brief  Ajout nominal d'un orderer à un canal Fabric
// / @input  fabricConf injecté sans erreur, type orderer, adresse host:port valide
// / @expect aucune erreur, le nœud orderer est transmis au ChannelConfigPort
func TestAddNode_Orderer_NominalCase(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypeOrderer, "orderer.example.com:7050", "OrdererMSP", channel.NodeCerts{})
	if err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if len(mock.addedNodes) != 1 || mock.addedNodes[0].nodeType != channel.NodeTypeOrderer {
		t.Errorf("nœud orderer non transmis correctement : %+v", mock.addedNodes)
	}
}

// / @brief  Rejet d'une adresse de nœud qui n'est pas au format host:port
// / @input  addr "not-a-host-port"
// / @expect retourne une erreur de validation, le ChannelConfigPort n'est jamais appelé
func TestAddNode_InvalidAddr_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "not-a-host-port", "Org2MSP", channel.NodeCerts{})
	if err == nil {
		t.Error("want error for invalid host:port address")
	}
	if len(mock.addedNodes) != 0 {
		t.Error("le ChannelConfigPort ne doit pas être appelé si l'adresse est invalide")
	}
}

// / @brief  Rejet d'un type de nœud autre que peer ou orderer
// / @input  nodeType "relay" (invalide)
// / @expect retourne une erreur de validation, le ChannelConfigPort n'est jamais appelé
func TestAddNode_InvalidType_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeType("relay"), "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{})
	if err == nil {
		t.Error("want error for invalid node type")
	}
	if len(mock.addedNodes) != 0 {
		t.Error("le ChannelConfigPort ne doit pas être appelé si le type est invalide")
	}
}

// / @brief  AddNode refuse d'opérer si aucun adapter Fabric n'est configuré
// / @input  Service sans WithFabricConfig
// / @expect retourne ErrFabricUnavailable
func TestAddNode_FabricUnavailable_Rejected(t *testing.T) {
	svc := channel.NewService(newMemRepo())

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{})
	if !errors.Is(err, channel.ErrFabricUnavailable) {
		t.Errorf("expected ErrFabricUnavailable, got %v", err)
	}
}

// / @brief  AddNode propage ErrNodeUnreachable renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrNodeUnreachable
// / @expect l'erreur ErrNodeUnreachable remonte inchangée
func TestAddNode_NodeUnreachable_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.addNodeErr = channel.ErrNodeUnreachable
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{})
	if !errors.Is(err, channel.ErrNodeUnreachable) {
		t.Errorf("expected ErrNodeUnreachable, got %v", err)
	}
}

// / @brief  AddNode propage ErrSyncTimeout renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrSyncTimeout
// / @expect l'erreur ErrSyncTimeout remonte inchangée
func TestAddNode_SyncTimeout_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.addNodeErr = channel.ErrSyncTimeout
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{})
	if !errors.Is(err, channel.ErrSyncTimeout) {
		t.Errorf("expected ErrSyncTimeout, got %v", err)
	}
}

// / @brief  AddNode propage ErrEndorsementPolicy renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrEndorsementPolicy
// / @expect l'erreur ErrEndorsementPolicy remonte inchangée
func TestAddNode_EndorsementPolicy_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.addNodeErr = channel.ErrEndorsementPolicy
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.AddNode("sandbox", channel.NodeTypePeer, "203.0.113.10:7051", "Org2MSP", channel.NodeCerts{})
	if !errors.Is(err, channel.ErrEndorsementPolicy) {
		t.Errorf("expected ErrEndorsementPolicy, got %v", err)
	}
}

// --- RemoveNode (UCADM04) ---

// / @brief  Retrait nominal d'un nœud membre du canal
// / @input  fabricConf injecté sans erreur, addr membre du canal
// / @expect aucune erreur, le retrait est transmis au ChannelConfigPort avec le bon channelID/addr
func TestRemoveNode_NominalCase(t *testing.T) {
	mock := newMockChannelConfig()
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	if err := svc.RemoveNode("sandbox", "203.0.113.10:7051"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}
	if len(mock.removedNodes) != 1 || mock.removedNodes[0].addr != "203.0.113.10:7051" {
		t.Errorf("retrait non transmis correctement : %+v", mock.removedNodes)
	}
}

// / @brief  RemoveNode refuse d'opérer si aucun adapter Fabric n'est configuré
// / @input  Service sans WithFabricConfig
// / @expect retourne ErrFabricUnavailable
func TestRemoveNode_FabricUnavailable_Rejected(t *testing.T) {
	svc := channel.NewService(newMemRepo())

	err := svc.RemoveNode("sandbox", "203.0.113.10:7051")
	if !errors.Is(err, channel.ErrFabricUnavailable) {
		t.Errorf("expected ErrFabricUnavailable, got %v", err)
	}
}

// / @brief  RemoveNode propage ErrNodeNotMember renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrNodeNotMember
// / @expect l'erreur ErrNodeNotMember remonte inchangée
func TestRemoveNode_NodeNotMember_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.removeNodeErr = channel.ErrNodeNotMember
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.RemoveNode("sandbox", "203.0.113.10:7051")
	if !errors.Is(err, channel.ErrNodeNotMember) {
		t.Errorf("expected ErrNodeNotMember, got %v", err)
	}
}

// / @brief  RemoveNode propage ErrMinNodesRequired (RM27 : seuil minimum 3 nœuds actifs)
// / @input  fabricConf configuré pour retourner ErrMinNodesRequired
// / @expect l'erreur ErrMinNodesRequired remonte inchangée, le retrait est refusé
func TestRemoveNode_RM27_MinNodesRequired_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.removeNodeErr = channel.ErrMinNodesRequired
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.RemoveNode("sandbox", "203.0.113.10:7051")
	if !errors.Is(err, channel.ErrMinNodesRequired) {
		t.Errorf("expected ErrMinNodesRequired, got %v", err)
	}
}

// / @brief  RemoveNode propage ErrEndorsementPolicy renvoyé par le ChannelConfigPort
// / @input  fabricConf configuré pour retourner ErrEndorsementPolicy
// / @expect l'erreur ErrEndorsementPolicy remonte inchangée
func TestRemoveNode_EndorsementPolicy_Rejected(t *testing.T) {
	mock := newMockChannelConfig()
	mock.removeNodeErr = channel.ErrEndorsementPolicy
	svc := channel.NewService(newMemRepo()).WithFabricConfig(mock)

	err := svc.RemoveNode("sandbox", "203.0.113.10:7051")
	if !errors.Is(err, channel.ErrEndorsementPolicy) {
		t.Errorf("expected ErrEndorsementPolicy, got %v", err)
	}
}
