// adapters/out/fabric/network_bootstrap.go
// Implémente network.NetworkBootstrapper : crée un réseau Hyperledger Fabric
// from scratch (UCADM02 flux nominal) sur la machine où myr s'exécute — CA,
// identités, configtx/genesis, conteneurs Docker (orderer + peers + CouchDB),
// rejoint du canal. Réutilise le client CA déjà utilisé par PeerProvisioner
// (peerCAClient, adapters/out/fabric/peer_provisioner.go) pour l'enregistrement
// et l'enrôlement des identités.
package fabric

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"myr-core/domain/network"
)

// FabricNetworkBootstrapper implémente network.NetworkBootstrapper.
type FabricNetworkBootstrapper struct{}

func NewFabricNetworkBootstrapper() *FabricNetworkBootstrapper {
	return &FabricNetworkBootstrapper{}
}

const (
	bootstrapCAAdminUser = "admin"
	bootstrapCAAdminPwd  = "adminpw"

	caHostPort           = 7054
	ordererHostPort      = 7050
	ordererAdminHostPort = 7053
	ordererOpHostPort    = 9443
	peerBasePort         = 7051
	peerPortStep         = 10
	couchdbBasePort      = 5984
)

// Bootstrap crée un réseau blockchain from scratch : démarre la CA, provisionne
// les identités (admin d'org, orderer, peers), génère configtx/genesis, démarre
// les conteneurs Docker, fait rejoindre le canal à l'orderer et aux peers.
func (b *FabricNetworkBootstrapper) Bootstrap(req network.CreateNetworkRequest) (result *network.BootstrapResult, err error) {
	ctx := context.Background()

	networkDirID := fmt.Sprintf("net-%d", time.Now().UnixNano())
	base := filepath.Join(fabricBase(), "networks", networkDirID)
	// Bootstrap est tout-ou-rien : les noms de conteneurs Docker sont déterministes
	// par org (container_name: ca.{{.CAName}}, voir dockercompose_template.go), pas
	// par networkDirID — une tentative avortée qui laisse ses conteneurs/répertoire
	// derrière elle bloque donc *toute* tentative suivante pour la même org avec un
	// conflit "container name already in use". Un échec à n'importe quelle étape
	// nettoie donc ce que cette tentative a démarré.
	defer func() {
		if err != nil {
			cleanupFailedBootstrap(ctx, base)
		}
	}()
	cryptoDir := filepath.Join(base, "crypto")
	artifactsDir := filepath.Join(base, "channel-artifacts")
	for _, d := range []string{cryptoDir, artifactsDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, fmt.Errorf("création %s : %w", d, err)
		}
	}

	orgShort := strings.ToLower(strings.TrimSuffix(req.OrgMSPID, "MSP"))
	if orgShort == "" {
		orgShort = "org1"
	}
	caName := "ca-" + orgShort
	ordererMSPID := "OrdererMSP"
	ordererName := "orderer." + req.Domain
	networkDockerName := "myr-" + networkDirID

	peers := make([]dcPeerParams, req.NumPeers)
	for i := 0; i < req.NumPeers; i++ {
		peers[i] = dcPeerParams{
			Name:          fmt.Sprintf("peer%d.%s.%s", i, orgShort, req.Domain),
			Port:          peerBasePort + i*peerPortStep,
			ChaincodePort: peerBasePort + 1 + i*peerPortStep,
			OperationPort: 9444 + i,
			CouchDBName:   fmt.Sprintf("couchdb%d", i),
			CouchDBPort:   couchdbBasePort + i,
		}
	}

	// ── 1. Générer et démarrer le conteneur CA ────────────────────────────────
	composePath := filepath.Join(base, "docker-compose.yaml")
	composeYAML, err := renderDockerCompose(dockerComposeParams{
		NetworkName: networkDockerName,
		CryptoDir:   cryptoDir,
		CAName:      caName,
		CAAdminID:   bootstrapCAAdminUser,
		CAAdminPwd:  bootstrapCAAdminPwd,
		CAPort:      caHostPort,
		OrgMSPID:    req.OrgMSPID,
		Orderers: []dcOrdererParams{
			{Name: ordererName, Port: ordererHostPort, AdminPort: ordererAdminHostPort, OpPort: ordererOpHostPort},
		},
		Peers: peers,
	})
	if err != nil {
		return nil, fmt.Errorf("génération docker-compose.yaml : %w", err)
	}
	if err := os.WriteFile(composePath, []byte(composeYAML), 0o644); err != nil {
		return nil, fmt.Errorf("écriture docker-compose.yaml : %w", err)
	}

	if _, err := runDocker(ctx, base, "compose", "-f", "docker-compose.yaml", "up", "-d", "ca."+caName); err != nil {
		return nil, fmt.Errorf("démarrage CA : %w", err)
	}
	caEndpoint := fmt.Sprintf("https://localhost:%d", caHostPort)
	if err := waitCAReady(caEndpoint, 90*time.Second); err != nil {
		return nil, fmt.Errorf("CA non prête après démarrage : %w", err)
	}

	// ── 2. Enrôler l'admin bootstrap de la CA (auth basique admin:adminpw) ────
	// /cainfo répond dès que le serveur HTTP écoute, mais /enroll peut encore
	// échouer pendant un très court instant (initialisation interne) — quelques
	// tentatives supplémentaires absorbent cette marge sans être excessives.
	bootstrapClient := &peerCAClient{caURL: caEndpoint, caName: caName, http: buildPeerHTTPClient(nil)}
	var adminCert, adminKey, caCert string
	err = retry(5, 3*time.Second, func() error {
		var enrollErr error
		adminCert, adminKey, caCert, enrollErr = bootstrapClient.enroll(ctx, bootstrapCAAdminUser, bootstrapCAAdminPwd, "", nil)
		return enrollErr
	})
	if err != nil {
		return nil, fmt.Errorf("enrôlement admin CA : %w", err)
	}
	caAdminDir := filepath.Join(cryptoDir, "ca-admin")
	if err := writeFiles(caAdminDir, map[string]string{
		"cert.pem":    adminCert,
		"key.pem":     adminKey,
		"ca-cert.pem": caCert,
	}); err != nil {
		return nil, err
	}

	// Client CA authentifié avec l'admin fraîchement enrôlé — utilisé pour
	// enregistrer les identités suivantes (org admin, orderer, peers).
	authClient := &peerCAClient{
		caURL:     caEndpoint,
		caName:    caName,
		adminCert: []byte(adminCert),
		adminKey:  []byte(adminKey),
		http:      buildPeerHTTPClient(nil),
	}

	// ── 3. Provisionner l'admin de l'organisation applicative ─────────────────
	orgAdminID := "orgadmin." + orgShort
	orgMSPDir, orgAdminCert, err := provisionAdminMSP(ctx, authClient, orgAdminID, filepath.Join(cryptoDir, "org-msp"), caCert)
	if err != nil {
		return nil, fmt.Errorf("provisioning admin org : %w", err)
	}

	// ── 4. Provisionner l'admin de l'organisation orderer ─────────────────────
	ordererAdminID := "ordereradmin"
	ordererMSPRootDir, ordererAdminCert, err := provisionAdminMSP(ctx, authClient, ordererAdminID, filepath.Join(cryptoDir, "orderer-msp"), caCert)
	if err != nil {
		return nil, fmt.Errorf("provisioning admin orderer : %w", err)
	}

	// ── 5. Provisionner l'identité de l'orderer (signature + TLS) ─────────────
	ordererCryptoDir := filepath.Join(cryptoDir, "orderers", ordererName)
	if err := provisionNodeIdentity(ctx, authClient, ordererName, "orderer", ordererCryptoDir, []string{ordererName, "localhost"}, ordererAdminCert); err != nil {
		return nil, fmt.Errorf("provisioning identité orderer : %w", err)
	}

	// ── 6. Provisionner chaque peer (signature + TLS) ─────────────────────────
	for _, p := range peers {
		peerCryptoDir := filepath.Join(cryptoDir, "peers", p.Name)
		if err := provisionNodeIdentity(ctx, authClient, p.Name, "peer", peerCryptoDir, []string{p.Name, "localhost"}, orgAdminCert); err != nil {
			return nil, fmt.Errorf("provisioning identité %s : %w", p.Name, err)
		}
	}

	// ── 7. Générer configtx.yaml et le genesis block du canal ────────────────
	profileName := "MyrGenesis"
	configtxYAML, err := renderConfigtx(configtxParams{
		OrgMSPID:       req.OrgMSPID,
		OrgName:        req.OrgName,
		OrgMSPDir:      orgMSPDir,
		AnchorPeerHost: peers[0].Name,
		AnchorPeerPort: peers[0].Port,
		OrdererMSPID:   ordererMSPID,
		OrdererMSPDir:  ordererMSPRootDir,
		OrdererHost:    ordererName,
		OrdererPort:    ordererHostPort,
		OrdererTLSCert: filepath.Join(ordererCryptoDir, "tls", "server.crt"),
		ProfileName:    profileName,
	})
	if err != nil {
		return nil, fmt.Errorf("génération configtx.yaml : %w", err)
	}
	if err := os.WriteFile(filepath.Join(base, "configtx.yaml"), []byte(configtxYAML), 0o644); err != nil {
		return nil, fmt.Errorf("écriture configtx.yaml : %w", err)
	}

	genesisPath := filepath.Join(artifactsDir, req.ChannelName+".block")
	if _, err := runBin(ctx, "configtxgen", []string{
		"-profile", profileName,
		"-outputBlock", genesisPath,
		"-channelID", req.ChannelName,
	}, base, []string{"FABRIC_CFG_PATH=" + base}); err != nil {
		return nil, fmt.Errorf("génération genesis block : %w", err)
	}

	// ── 8. Démarrer l'orderer et les peers ────────────────────────────────────
	if _, err := runDocker(ctx, base, "compose", "-f", "docker-compose.yaml", "up", "-d"); err != nil {
		return nil, fmt.Errorf("démarrage des nœuds : %w", err)
	}
	if err := waitTCP(fmt.Sprintf("localhost:%d", ordererAdminHostPort), 60*time.Second); err != nil {
		return nil, fmt.Errorf("orderer non joignable après démarrage : %w", err)
	}
	for _, p := range peers {
		if err := waitTCP(fmt.Sprintf("localhost:%d", p.Port), 60*time.Second); err != nil {
			return nil, fmt.Errorf("peer %s non joignable après démarrage : %w", p.Name, err)
		}
	}

	// ── 9. L'orderer rejoint le canal (osnadmin, channel participation API) ──
	// L'API d'administration de l'orderer exige le mTLS dès que Admin.TLS.Enabled
	// est actif (Fabric refuse de démarrer sinon : "ClientAuthRequired must be
	// true if TLS.Enabled is true") — osnadmin doit donc présenter un certificat
	// client signé par la même CA ; le certificat TLS propre de l'orderer convient.
	// Les conteneurs viennent de démarrer : quelques tentatives absorbent la
	// marge d'initialisation du process orderer.
	if err := retry(5, 3*time.Second, func() error {
		res, err := runBin(ctx, "osnadmin", []string{
			"channel", "join",
			"--channelID", req.ChannelName,
			"--config-block", genesisPath,
			"--orderer-address", fmt.Sprintf("localhost:%d", ordererAdminHostPort),
			"--ca-file", filepath.Join(ordererCryptoDir, "tls", "ca.crt"),
			"--client-cert", filepath.Join(ordererCryptoDir, "tls", "server.crt"),
			"--client-key", filepath.Join(ordererCryptoDir, "tls", "server.key"),
		}, base, nil)
		if err == nil {
			err = checkOsnadminStatus(res)
		}
		if isAlreadyJoinedErr(err) {
			return nil
		}
		return err
	}); err != nil {
		return nil, fmt.Errorf("orderer channel join : %w", err)
	}

	// ── 10. Chaque peer rejoint le canal ───────────────────────────────────────
	// La proposition JoinChain doit être signée par une identité satisfaisant
	// la politique locale "Admins" du peer (vérifiée contre son admincerts) —
	// donc par l'admin de l'org, PAS par l'identité propre du peer (qui ne
	// figure pas dans admincerts et se ferait rejeter : "This identity is not
	// an admin"). CORE_PEER_ADDRESS cible tout de même le peer précis à rejoindre.
	for _, p := range peers {
		peerTLSDir := filepath.Join(cryptoDir, "peers", p.Name, "tls")
		cfgPathEnv, err := ensureCoreYAML(base)
		if err != nil {
			return nil, fmt.Errorf("écriture core.yaml : %w", err)
		}
		env := []string{
			cfgPathEnv,
			"CORE_PEER_MSPCONFIGPATH=" + orgMSPDir,
			"CORE_PEER_LOCALMSPID=" + req.OrgMSPID,
			"CORE_PEER_ADDRESS=" + fmt.Sprintf("localhost:%d", p.Port),
			"CORE_PEER_TLS_ENABLED=true",
			"CORE_PEER_TLS_ROOTCERT_FILE=" + filepath.Join(peerTLSDir, "ca.crt"),
		}
		if err := retry(5, 3*time.Second, func() error {
			_, err := runBin(ctx, "peer", []string{"channel", "join", "-b", genesisPath}, base, env)
			if isAlreadyJoinedErr(err) {
				return nil
			}
			return err
		}); err != nil {
			return nil, fmt.Errorf("peer %s channel join : %w", p.Name, err)
		}
	}

	// ── 11. Construire le profil réseau résultant ─────────────────────────────
	nodes := make([]network.BootstrapNode, 0, len(peers)+1)
	nodes = append(nodes, network.BootstrapNode{Type: "orderer", Addr: fmt.Sprintf("localhost:%d", ordererHostPort)})
	for _, p := range peers {
		nodes = append(nodes, network.BootstrapNode{Type: "peer", Addr: fmt.Sprintf("localhost:%d", p.Port)})
	}

	profile := &network.NetworkProfile{
		ID:                   networkDirID,
		Name:                 req.Name,
		PeerEndpoint:         fmt.Sprintf("localhost:%d", peers[0].Port),
		GatewayPeer:          peers[0].Name,
		MSPID:                req.OrgMSPID,
		CertPath:             filepath.Join(orgMSPDir, "signcerts", "cert.pem"),
		KeyPath:              filepath.Join(orgMSPDir, "keystore", "key.pem"),
		TLSCertPath:          filepath.Join(orgMSPDir, "cacerts", "ca.pem"),
		FabricChannel:        req.ChannelName,
		Channels:             []string{req.ChannelName},
		CAEndpoint:           caEndpoint,
		CAName:               caName,
		OrdererEndpoint:      fmt.Sprintf("localhost:%d", ordererHostPort),
		OrdererTLSCACertPath: filepath.Join(ordererCryptoDir, "tls", "ca.crt"),
		CAAdminCertPath:      filepath.Join(caAdminDir, "cert.pem"),
		CAAdminKeyPath:       filepath.Join(caAdminDir, "key.pem"),
		CreatedAt:            time.Now(),
	}

	return &network.BootstrapResult{Profile: profile, Nodes: nodes}, nil
}

// cleanupFailedBootstrap arrête et supprime les conteneurs Docker démarrés par une
// tentative de Bootstrap avortée, puis supprime son répertoire de travail. Sans ce
// nettoyage, les conteneurs (noms déterministes par org — container_name dans
// dockercompose_template.go, pas de suffixe par networkDirID) restent en place et
// toute tentative suivante échoue avec "Conflict: container name already in use".
// `docker compose down` sur un fichier dont les services n'ont jamais démarré (ex:
// échec avant l'étape 1) ne fait rien et ne renvoie pas d'erreur — l'appel est donc
// sûr à n'importe quel stade de Bootstrap.
func cleanupFailedBootstrap(ctx context.Context, base string) {
	composePath := filepath.Join(base, "docker-compose.yaml")
	if _, statErr := os.Stat(composePath); statErr == nil {
		if _, err := runDocker(ctx, base, "compose", "-f", "docker-compose.yaml", "down", "-v", "--remove-orphans"); err != nil {
			fmt.Fprintf(os.Stderr, "myr: nettoyage après échec du bootstrap réseau (%s) : %v\n", base, err)
		}
	}
	// La CA (image hyperledger/fabric-ca, voir dockercompose_template.go) écrit
	// dans son volume monté ({{.CryptoDir}}/ca) en tant que root — le process myr
	// tourne lui en tant qu'utilisateur non-root et ne peut donc pas supprimer ces
	// fichiers directement depuis l'hôte (os.RemoveAll échouerait avec
	// "permission denied"). Un conteneur jetable, root côté conteneur, est
	// autorisé sur le bind-mount quel que soit le propriétaire des fichiers créés
	// par un conteneur précédent : on lui délègue la suppression du contenu avant
	// de retirer le répertoire de base lui-même (celui-ci appartient à myr, donc
	// supprimable normalement une fois vide).
	if _, err := runDocker(ctx, base, "run", "--rm", "-v", base+":/data", "alpine", "sh", "-c", "rm -rf /data/*"); err != nil {
		fmt.Fprintf(os.Stderr, "myr: nettoyage après échec du bootstrap réseau (%s) : suppression du contenu via conteneur jetable : %v\n", base, err)
	}
	if err := os.RemoveAll(base); err != nil {
		fmt.Fprintf(os.Stderr, "myr: nettoyage après échec du bootstrap réseau : suppression %s : %v\n", base, err)
	}
}

// provisionAdminMSP enregistre et enrôle une identité admin (Type "client"),
// puis écrit sa structure MSP sur disque (signcerts, keystore, cacerts et,
// pour compatibilité avec la politique 'MSPID.admin' sans NodeOUs, admincerts).
// Retourne le répertoire MSP écrit.
// provisionAdminMSP retourne (mspDir, certPEM de l'admin) — certPEM est réutilisé
// par provisionNodeIdentity pour déclarer les nœuds de l'org comme admincerts
// (sans NodeOUs, Fabric exige qu'au moins un admin soit déclaré dans chaque MSP
// locale : "administrators must be declared when no admin ou classification is set").
func provisionAdminMSP(ctx context.Context, c *peerCAClient, id, mspDir, caCert string) (string, string, error) {
	secret, err := c.registerIdentity(ctx, id, "client", "")
	if err != nil {
		return "", "", fmt.Errorf("register %s : %w", id, err)
	}
	cert, key, _, err := c.enroll(ctx, id, secret, "", nil)
	if err != nil {
		return "", "", fmt.Errorf("enroll %s : %w", id, err)
	}
	if err := writeFiles(mspDir, map[string]string{
		filepath.Join("signcerts", "cert.pem"):   cert,
		filepath.Join("keystore", "key.pem"):     key,
		filepath.Join("cacerts", "ca.pem"):       caCert,
		filepath.Join("tlscacerts", "tlsca.pem"): caCert,
		filepath.Join("admincerts", "cert.pem"):  cert,
		"config.yaml":                            nodeOUsConfigYAML,
	}); err != nil {
		return "", "", err
	}
	return mspDir, cert, nil
}

// provisionNodeIdentity enregistre et enrôle un peer ou un orderer (signature + TLS)
// et écrit sa structure msp/ + tls/ sur disque sous nodeDir. adminCertPEM est
// déclaré comme admincerts de la MSP locale du nœud (cf. commentaire ci-dessus).
func provisionNodeIdentity(ctx context.Context, c *peerCAClient, id, idType, nodeDir string, tlsHosts []string, adminCertPEM string) error {
	secret, err := c.registerIdentity(ctx, id, idType, "")
	if err != nil {
		return fmt.Errorf("register %s : %w", id, err)
	}
	signCert, signKey, caCert, err := c.enroll(ctx, id, secret, "", nil)
	if err != nil {
		return fmt.Errorf("enroll signature %s : %w", id, err)
	}
	tlsCert, tlsKey, tlsCACert, err := c.enroll(ctx, id, secret, "tls", tlsHosts)
	if err != nil {
		return fmt.Errorf("enroll TLS %s : %w", id, err)
	}
	if tlsCACert == "" {
		tlsCACert = caCert
	}
	return writeFiles(nodeDir, map[string]string{
		filepath.Join("msp", "signcerts", "cert.pem"):   signCert,
		filepath.Join("msp", "keystore", "key.pem"):     signKey,
		filepath.Join("msp", "cacerts", "ca.pem"):       caCert,
		filepath.Join("msp", "tlscacerts", "tlsca.pem"): caCert,
		filepath.Join("msp", "admincerts", "cert.pem"):  adminCertPEM,
		filepath.Join("msp", "config.yaml"):             nodeOUsConfigYAML,
		filepath.Join("tls", "server.crt"):              tlsCert,
		filepath.Join("tls", "server.key"):              tlsKey,
		filepath.Join("tls", "ca.crt"):                  tlsCACert,
	})
}

// writeFiles écrit chaque contenu de files (clé = chemin relatif à dir) sur disque,
// en créant les répertoires intermédiaires nécessaires.
func writeFiles(dir string, files map[string]string) error {
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			return fmt.Errorf("mkdir %s : %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			return fmt.Errorf("écriture %s : %w", full, err)
		}
	}
	return nil
}

// waitTCP attend qu'une adresse TCP accepte des connexions, jusqu'à timeout.
func waitTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout en attendant %s : %w", addr, lastErr)
}
