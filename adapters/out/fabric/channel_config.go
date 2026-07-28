// adapters/out/fabric/channel_config.go
// Implémente domain/channel.ChannelConfigPort via des mises à jour réelles de la
// configuration de canal Fabric (UCADM01/03/04) : peer channel fetch/signconfigtx/
// update + configtxlator, exactement comme scripts/create-sandbox-channel.sh le
// fait manuellement — mais piloté depuis Go (os/exec), car le Fabric Gateway SDK v2
// ne permet pas ces opérations de configuration.
package fabric

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"myr-core/adapters/out/localstorage"
	"myr-core/domain/channel"
	"myr-core/domain/network"
)

// FabricChannelConfig implémente channel.ChannelConfigPort.
type FabricChannelConfig struct {
	netSvc    network.NetworkService
	nodeStore *localstorage.JSONNodeStore
}

func NewFabricChannelConfig(netSvc network.NetworkService, nodeStore *localstorage.JSONNodeStore) *FabricChannelConfig {
	return &FabricChannelConfig{netSvc: netSvc, nodeStore: nodeStore}
}

// ── résolution du contexte réseau actif ──────────────────────────────────────

func (f *FabricChannelConfig) resolveActive() (*network.NetworkProfile, string, error) {
	profile, err := f.netSvc.GetActive()
	if err != nil {
		return nil, "", err
	}
	if profile == nil {
		return nil, "", channel.ErrFabricUnavailable
	}
	if profile.OrdererEndpoint == "" {
		return nil, "", fmt.Errorf("orderer non configuré pour le réseau %q — créez-le via 'myr network create' ou renseignez orderer_endpoint", profile.Name)
	}
	base := filepath.Join(fabricBase(), "networks", profile.ID)
	return profile, base, nil
}

func ordererCAFile(profile *network.NetworkProfile) string {
	if profile.OrdererTLSCACertPath != "" {
		return profile.OrdererTLSCACertPath
	}
	return profile.TLSCertPath
}

func (f *FabricChannelConfig) adminMSPDir(base string) string {
	return filepath.Join(base, "crypto", "org-msp")
}

func (f *FabricChannelConfig) peerEnv(profile *network.NetworkProfile, base string) ([]string, error) {
	cfgPathEnv, err := ensureCoreYAML(base)
	if err != nil {
		return nil, fmt.Errorf("écriture core.yaml : %w", err)
	}
	return []string{
		cfgPathEnv,
		"CORE_PEER_MSPCONFIGPATH=" + f.adminMSPDir(base),
		"CORE_PEER_LOCALMSPID=" + profile.MSPID,
		"CORE_PEER_ADDRESS=" + profile.PeerEndpoint,
		"CORE_PEER_TLS_ENABLED=true",
		"CORE_PEER_TLS_ROOTCERT_FILE=" + profile.TLSCertPath,
	}, nil
}

// ── AddOrganisation (UCADM01) ────────────────────────────────────────────────

func (f *FabricChannelConfig) AddOrganisation(channelID string, org channel.Organization) error {
	profile, base, err := f.resolveActive()
	if err != nil {
		return err
	}
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp(base, "addorg-*")
	if err != nil {
		return fmt.Errorf("répertoire temporaire : %w", err)
	}
	defer os.RemoveAll(tmpDir)

	orgMSPDir := filepath.Join(tmpDir, "neworg-msp")
	if err := writeNewOrgMSP(orgMSPDir, org); err != nil {
		return fmt.Errorf("MSP de la nouvelle organisation : %w", err)
	}
	orgConfigtx := renderOrgOnlyConfigtx(org.MSPID, orgMSPDir)
	if err := os.WriteFile(filepath.Join(tmpDir, "configtx.yaml"), []byte(orgConfigtx), 0o644); err != nil {
		return err
	}

	printRes, err := runBin(ctx, "configtxgen", []string{"-printOrg", org.MSPID}, tmpDir, []string{"FABRIC_CFG_PATH=" + tmpDir})
	if err != nil {
		return fmt.Errorf("génération de la définition d'organisation : %w", err)
	}
	var orgGroup interface{}
	if err := json.Unmarshal([]byte(printRes.Stdout), &orgGroup); err != nil {
		return fmt.Errorf("parse de la définition d'organisation : %w", err)
	}

	_, err = f.submitConfigUpdate(ctx, profile, base, tmpDir, channelID, func(cfg map[string]interface{}) (bool, error) {
		appGroups, err := groupsAt(cfg, "channel_group", "groups", "Application", "groups")
		if err != nil {
			return false, err
		}
		if _, exists := appGroups[org.MSPID]; exists {
			return false, channel.ErrAlreadyMember
		}
		appGroups[org.MSPID] = orgGroup
		return true, nil
	})
	return err
}

// writeNewOrgMSP écrit la MSP minimale d'une nouvelle organisation (cacerts + NodeOUs)
// à partir du certificat racine fourni par l'admin (myr org add --cert).
func writeNewOrgMSP(dir string, org channel.Organization) error {
	files := map[string]string{
		filepath.Join("cacerts", "ca.pem"): org.RootCert,
		"config.yaml":                      nodeOUsConfigYAML,
	}
	if org.TLSCert != "" {
		files[filepath.Join("tlscacerts", "tlsca.pem")] = org.TLSCert
	}
	return writeFiles(dir, files)
}

func renderOrgOnlyConfigtx(mspID, mspDir string) string {
	return fmt.Sprintf(`Organizations:
  - &NewOrg
    Name: %s
    ID: %s
    MSPDir: %s
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('%s.admin', '%s.peer', '%s.client')"
      Writers:
        Type: Signature
        Rule: "OR('%s.admin', '%s.client')"
      Admins:
        Type: Signature
        Rule: "OR('%s.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('%s.peer')"
`, mspID, mspID, mspDir, mspID, mspID, mspID, mspID, mspID, mspID, mspID)
}

// groupsAt navigue dans une config JSON décodée (map imbriquée) et retourne la
// map trouvée au bout du chemin, en créant les niveaux "groups"/"values" manquants.
func groupsAt(cfg map[string]interface{}, path ...string) (map[string]interface{}, error) {
	cur := cfg
	for i, key := range path {
		v, ok := cur[key]
		if !ok {
			// autorise la création d'un niveau "groups"/"values" vide s'il manque
			if key == "groups" || key == "values" {
				m := map[string]interface{}{}
				cur[key] = m
				cur = m
				continue
			}
			return nil, fmt.Errorf("champ de configuration manquant : %s", strings.Join(path[:i+1], "."))
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("champ de configuration invalide : %s", strings.Join(path[:i+1], "."))
		}
		cur = m
	}
	return cur, nil
}

// ── AddNode (UCADM03) ─────────────────────────────────────────────────────────

func (f *FabricChannelConfig) AddNode(channelID string, nodeType channel.NodeType, addr, orgMSP string, certs channel.NodeCerts) error {
	profile, base, err := f.resolveActive()
	if err != nil {
		return err
	}
	if err := checkTCPReachable(addr); err != nil {
		return channel.ErrNodeUnreachable
	}
	ctx := context.Background()

	switch nodeType {
	case channel.NodeTypePeer:
		if err := f.setAnchorPeer(ctx, profile, base, channelID, orgMSP, addr); err != nil {
			return err
		}
	case channel.NodeTypeOrderer:
		if err := f.addConsenter(ctx, profile, base, channelID, addr, certs); err != nil {
			return err
		}
	default:
		return fmt.Errorf("type de nœud invalide : %s", nodeType)
	}

	return f.nodeStore.Add(&localstorage.ProvisionedNode{
		ChannelID: channelID,
		Type:      string(nodeType),
		Addr:      addr,
		OrgMSP:    orgMSP,
		Active:    true,
		CreatedAt: time.Now(),
	})
}

// setAnchorPeer déclare addr comme anchor peer de orgMSP sur le canal (config update réel).
// Best-effort : si l'org possède déjà un anchor peer identique, ne fait rien.
func (f *FabricChannelConfig) setAnchorPeer(ctx context.Context, profile *network.NetworkProfile, base, channelID, orgMSP, addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("adresse invalide %q : %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port invalide %q : %w", addr, err)
	}

	tmpDir, err := os.MkdirTemp(base, "addnode-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	_, err = f.submitConfigUpdate(ctx, profile, base, tmpDir, channelID, func(cfg map[string]interface{}) (bool, error) {
		orgValues, err := groupsAt(cfg, "channel_group", "groups", "Application", "groups", orgMSP, "values")
		if err != nil {
			return false, fmt.Errorf("organisation %q introuvable dans la configuration du canal : %w", orgMSP, err)
		}
		anchorEntry, _ := orgValues["AnchorPeers"].(map[string]interface{})
		if anchorEntry == nil {
			anchorEntry = map[string]interface{}{"mod_policy": "Admins", "version": "0"}
		}
		value, _ := anchorEntry["value"].(map[string]interface{})
		if value == nil {
			value = map[string]interface{}{}
		}
		list, _ := value["anchor_peers"].([]interface{})
		for _, a := range list {
			am, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			if am["host"] == host && fmt.Sprintf("%v", am["port"]) == fmt.Sprintf("%v", port) {
				return false, nil // déjà anchor peer — pas de changement
			}
		}
		list = append(list, map[string]interface{}{"host": host, "port": port})
		value["anchor_peers"] = list
		anchorEntry["value"] = value
		orgValues["AnchorPeers"] = anchorEntry
		return true, nil
	})
	return err
}

// addConsenter ajoute un orderer au groupe de consensus etcdraft (config update réel).
func (f *FabricChannelConfig) addConsenter(ctx context.Context, profile *network.NetworkProfile, base, channelID, addr string, certs channel.NodeCerts) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("adresse invalide %q : %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("port invalide %q : %w", addr, err)
	}
	if certs.TLSCert == "" {
		return fmt.Errorf("le certificat TLS du nouvel orderer est requis (--cert)")
	}
	tlsDER, err := pemToBase64DER(certs.TLSCert)
	if err != nil {
		return fmt.Errorf("certificat TLS invalide : %w", err)
	}

	tmpDir, err := os.MkdirTemp(base, "addnode-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	_, err = f.submitConfigUpdate(ctx, profile, base, tmpDir, channelID, func(cfg map[string]interface{}) (bool, error) {
		consensusType, err := groupsAt(cfg, "channel_group", "groups", "Orderer", "values")
		if err != nil {
			return false, err
		}
		ct, ok := consensusType["ConsensusType"].(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("ConsensusType introuvable dans la configuration de l'orderer")
		}
		value, _ := ct["value"].(map[string]interface{})
		metadata, _ := value["metadata"].(map[string]interface{})
		consenters, _ := metadata["consenters"].([]interface{})
		for _, c := range consenters {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cm["host"] == host && fmt.Sprintf("%v", cm["port"]) == fmt.Sprintf("%v", port) {
				return false, channel.ErrAlreadyMember
			}
		}
		consenters = append(consenters, map[string]interface{}{
			"host":            host,
			"port":            port,
			"client_tls_cert": tlsDER,
			"server_tls_cert": tlsDER,
		})
		metadata["consenters"] = consenters
		value["metadata"] = metadata
		ct["value"] = value
		consensusType["ConsensusType"] = ct
		return true, nil
	})
	return err
}

// ── RemoveNode (UCADM04) ──────────────────────────────────────────────────────

// #incoherence — le seuil RM27 (min. 3 nœuds actifs) est vérifié ici, dans
// l'adapter, et non dans domain/channel (contrairement à l'ajout de nœud,
// vérifié côté domaine dans domain/network/service.go) ; cet adapter importe
// aussi directement adapters/out/localstorage pour lire ce compte — voir
// specs/roadmap_dev.md § Écarts — revue de code, E6.
func (f *FabricChannelConfig) RemoveNode(channelID, addr string) error {
	profile, base, err := f.resolveActive()
	if err != nil {
		return err
	}
	node, err := f.nodeStore.Find(channelID, addr)
	if err != nil {
		return err
	}
	if node == nil {
		return channel.ErrNodeNotMember
	}
	count, err := f.nodeStore.ActiveCount(channelID)
	if err != nil {
		return err
	}
	if count <= 3 {
		return channel.ErrMinNodesRequired
	}

	ctx := context.Background()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("adresse invalide %q : %w", addr, err)
	}
	port, _ := strconv.Atoi(portStr)

	tmpDir, err := os.MkdirTemp(base, "removenode-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	switch node.Type {
	case "peer":
		_, err = f.submitConfigUpdate(ctx, profile, base, tmpDir, channelID, func(cfg map[string]interface{}) (bool, error) {
			orgValues, err := groupsAt(cfg, "channel_group", "groups", "Application", "groups", node.OrgMSP, "values")
			if err != nil {
				return false, nil // organisation déjà absente — rien à faire côté config
			}
			anchorEntry, _ := orgValues["AnchorPeers"].(map[string]interface{})
			if anchorEntry == nil {
				return false, nil
			}
			value, _ := anchorEntry["value"].(map[string]interface{})
			list, _ := value["anchor_peers"].([]interface{})
			filtered := list[:0]
			changed := false
			for _, a := range list {
				am, ok := a.(map[string]interface{})
				if ok && am["host"] == host && fmt.Sprintf("%v", am["port"]) == fmt.Sprintf("%v", port) {
					changed = true
					continue
				}
				filtered = append(filtered, a)
			}
			value["anchor_peers"] = filtered
			anchorEntry["value"] = value
			orgValues["AnchorPeers"] = anchorEntry
			return changed, nil
		})
	case "orderer":
		_, err = f.submitConfigUpdate(ctx, profile, base, tmpDir, channelID, func(cfg map[string]interface{}) (bool, error) {
			consensusType, err := groupsAt(cfg, "channel_group", "groups", "Orderer", "values")
			if err != nil {
				return false, err
			}
			ct, _ := consensusType["ConsensusType"].(map[string]interface{})
			value, _ := ct["value"].(map[string]interface{})
			metadata, _ := value["metadata"].(map[string]interface{})
			consenters, _ := metadata["consenters"].([]interface{})
			filtered := consenters[:0]
			changed := false
			for _, c := range consenters {
				cm, ok := c.(map[string]interface{})
				if ok && cm["host"] == host && fmt.Sprintf("%v", cm["port"]) == fmt.Sprintf("%v", port) {
					changed = true
					continue
				}
				filtered = append(filtered, c)
			}
			metadata["consenters"] = filtered
			value["metadata"] = metadata
			ct["value"] = value
			consensusType["ConsensusType"] = ct
			return changed, nil
		})
	}
	if err != nil {
		return err
	}
	return f.nodeStore.Deactivate(channelID, addr)
}

// ── soumission générique d'une mise à jour de configuration de canal ─────────

// submitConfigUpdate récupère la configuration courante du canal, applique patch,
// et — si patch signale un changement — calcule et soumet la mise à jour à
// l'orderer (fetch → decode → patch → compute_update → encode → sign → update).
// patch retourne (changed, err) : changed=false ⇒ aucune transaction n'est soumise.
func (f *FabricChannelConfig) submitConfigUpdate(
	ctx context.Context,
	profile *network.NetworkProfile,
	base, tmpDir, channelID string,
	patch func(cfg map[string]interface{}) (bool, error),
) (bool, error) {
	env, err := f.peerEnv(profile, base)
	if err != nil {
		return false, err
	}
	ordererCA := ordererCAFile(profile)

	blockPB := filepath.Join(tmpDir, "config_block.pb")
	if _, err := runBin(ctx, "peer", []string{
		"channel", "fetch", "config", blockPB,
		"-c", channelID,
		"-o", profile.OrdererEndpoint,
		"--tls", "--cafile", ordererCA,
	}, base, env); err != nil {
		return false, fmt.Errorf("récupération de la configuration du canal : %w", err)
	}

	blockJSON := filepath.Join(tmpDir, "config_block.json")
	if _, err := runBin(ctx, "configtxlator", []string{
		"proto_decode", "--input", blockPB, "--type", "common.Block", "--output", blockJSON,
	}, base, nil); err != nil {
		return false, fmt.Errorf("décodage de la configuration : %w", err)
	}

	blockData, err := os.ReadFile(blockJSON)
	if err != nil {
		return false, err
	}
	var block map[string]interface{}
	if err := json.Unmarshal(blockData, &block); err != nil {
		return false, fmt.Errorf("parse de la configuration : %w", err)
	}
	originalConfig, err := extractConfig(block)
	if err != nil {
		return false, err
	}

	// Copie profonde via un aller-retour JSON pour obtenir la config modifiable.
	rawOriginal, err := json.Marshal(originalConfig)
	if err != nil {
		return false, err
	}
	var modifiedConfig map[string]interface{}
	if err := json.Unmarshal(rawOriginal, &modifiedConfig); err != nil {
		return false, err
	}

	changed, err := patch(modifiedConfig)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	originalPath := filepath.Join(tmpDir, "original.json")
	modifiedPath := filepath.Join(tmpDir, "modified.json")
	if err := writeJSON(originalPath, originalConfig); err != nil {
		return false, err
	}
	if err := writeJSON(modifiedPath, modifiedConfig); err != nil {
		return false, err
	}

	originalPB := filepath.Join(tmpDir, "original.pb")
	modifiedPB := filepath.Join(tmpDir, "modified.pb")
	if _, err := runBin(ctx, "configtxlator", []string{"proto_encode", "--input", originalPath, "--type", "common.Config", "--output", originalPB}, base, nil); err != nil {
		return false, fmt.Errorf("encodage config originale : %w", err)
	}
	if _, err := runBin(ctx, "configtxlator", []string{"proto_encode", "--input", modifiedPath, "--type", "common.Config", "--output", modifiedPB}, base, nil); err != nil {
		return false, fmt.Errorf("encodage config modifiée : %w", err)
	}

	updatePB := filepath.Join(tmpDir, "update.pb")
	if _, err := runBin(ctx, "configtxlator", []string{
		"compute_update", "--channel_id", channelID,
		"--original", originalPB, "--updated", modifiedPB, "--output", updatePB,
	}, base, nil); err != nil {
		return false, fmt.Errorf("calcul de la mise à jour de configuration : %w", err)
	}

	updateJSONPath := filepath.Join(tmpDir, "update.json")
	if _, err := runBin(ctx, "configtxlator", []string{"proto_decode", "--input", updatePB, "--type", "common.ConfigUpdate", "--output", updateJSONPath}, base, nil); err != nil {
		return false, fmt.Errorf("décodage de la mise à jour : %w", err)
	}
	updateData, err := os.ReadFile(updateJSONPath)
	if err != nil {
		return false, err
	}
	var configUpdate interface{}
	if err := json.Unmarshal(updateData, &configUpdate); err != nil {
		return false, err
	}

	envelope := map[string]interface{}{
		"payload": map[string]interface{}{
			"header": map[string]interface{}{
				"channel_header": map[string]interface{}{
					"channel_id": channelID,
					"type":       2,
				},
			},
			"data": map[string]interface{}{
				"config_update": configUpdate,
			},
		},
	}
	envelopePath := filepath.Join(tmpDir, "envelope.json")
	if err := writeJSON(envelopePath, envelope); err != nil {
		return false, err
	}
	envelopePB := filepath.Join(tmpDir, "envelope.pb")
	if _, err := runBin(ctx, "configtxlator", []string{"proto_encode", "--input", envelopePath, "--type", "common.Envelope", "--output", envelopePB}, base, nil); err != nil {
		return false, fmt.Errorf("encodage de l'enveloppe : %w", err)
	}

	if _, err := runBin(ctx, "peer", []string{"channel", "signconfigtx", "-f", envelopePB}, base, env); err != nil {
		return false, channel.ErrEndorsementPolicy
	}
	if _, err := runBin(ctx, "peer", []string{
		"channel", "update", "-f", envelopePB,
		"-c", channelID, "-o", profile.OrdererEndpoint,
		"--tls", "--cafile", ordererCA,
	}, base, env); err != nil {
		return false, fmt.Errorf("soumission de la mise à jour : %w", err)
	}
	return true, nil
}

// extractConfig extrait l'objet common.Config d'un bloc de configuration décodé
// (block.data.data[0].payload.data.config).
func extractConfig(block map[string]interface{}) (map[string]interface{}, error) {
	data, ok := block["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bloc invalide : champ data manquant")
	}
	items, ok := data["data"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("bloc invalide : aucune transaction")
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bloc invalide : transaction illisible")
	}
	payload, ok := item["payload"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bloc invalide : payload manquant")
	}
	payloadData, ok := payload["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bloc invalide : payload.data manquant")
	}
	config, ok := payloadData["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bloc invalide : config manquante")
	}
	return config, nil
}

func writeJSON(path string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// pemToBase64DER convertit un certificat PEM en DER encodé base64 (format attendu
// par les champs client_tls_cert/server_tls_cert de la configuration etcdraft).
func pemToBase64DER(pemStr string) (string, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return "", fmt.Errorf("PEM invalide")
	}
	return base64.StdEncoding.EncodeToString(block.Bytes), nil
}

// checkTCPReachable vérifie qu'une adresse host:port accepte une connexion TCP.
func checkTCPReachable(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
