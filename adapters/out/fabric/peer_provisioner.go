// adapters/out/fabric/peer_provisioner.go
// Implémente network.PeerProvisioner via l'API REST Fabric CA.
// Enregistre une identité de type "peer" et génère ses certificats MSP + TLS.
package fabric

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"myr-core/domain/network"
)

// FabricPeerProvisioner implémente network.PeerProvisioner.
type FabricPeerProvisioner struct{}

func NewFabricPeerProvisioner() *FabricPeerProvisioner {
	return &FabricPeerProvisioner{}
}

// RegisterAndProvision enregistre le peer dans la CA et obtient ses certificats.
func (p *FabricPeerProvisioner) RegisterAndProvision(
	prof *network.NetworkProfile,
	req network.AddPeerRequest,
) (*network.PeerCredentials, error) {
	ctx := context.Background()

	// ── Charger les credentials admin depuis le profil réseau ─────────────────
	var adminCert, adminKey, tlsCACert []byte
	var err error

	if prof.CertPath != "" {
		if adminCert, err = os.ReadFile(prof.CertPath); err != nil {
			return nil, fmt.Errorf("lecture cert admin : %w", err)
		}
	}
	if prof.KeyPath != "" {
		if adminKey, err = os.ReadFile(prof.KeyPath); err != nil {
			return nil, fmt.Errorf("lecture clé admin : %w", err)
		}
	}
	if prof.TLSCertPath != "" {
		if tlsCACert, err = os.ReadFile(prof.TLSCertPath); err != nil {
			return nil, fmt.Errorf("lecture cert TLS CA : %w", err)
		}
	}

	hostname := req.Hostname
	if hostname == "" {
		hostname = req.PeerID
	}

	pc := &peerCAClient{
		caURL:     prof.CAEndpoint,
		caName:    prof.CAName,
		adminCert: adminCert,
		adminKey:  adminKey,
		http:      buildPeerHTTPClient(tlsCACert),
	}

	// ── 1. Enregistrer l'identité peer dans la CA ─────────────────────────────
	secret, err := pc.registerPeer(ctx, req.PeerID, req.Secret)
	if err != nil {
		return nil, fmt.Errorf("CA register : %w", err)
	}

	// ── 2. Certificats de signature (MSP) ─────────────────────────────────────
	signCert, signKey, caCert, err := pc.enroll(ctx, req.PeerID, secret, "", nil)
	if err != nil {
		return nil, fmt.Errorf("CA enroll signing : %w", err)
	}

	// ── 3. Certificats TLS (profil "tls" avec SANs) ───────────────────────────
	tlsCert, tlsKey, _, err := pc.enroll(ctx, req.PeerID, secret, "tls", []string{hostname, "localhost"})
	if err != nil {
		return nil, fmt.Errorf("CA enroll TLS : %w", err)
	}

	creds := &network.PeerCredentials{
		PeerID:   req.PeerID,
		SignCert: signCert,
		SignKey:  signKey,
		CACert:   caCert,
		TLSCert:  tlsCert,
		TLSKey:   tlsKey,
	}
	creds.Files = PeerMSPFiles(creds)
	creds.StartupInstructions = PeerStartupInstructions(req.PeerID)
	return creds, nil
}

// PeerMSPFiles construit la mise en page de fichiers attendue par un peer
// Hyperledger Fabric (structure MSP + TLS) à partir des matériaux générés.
func PeerMSPFiles(creds *network.PeerCredentials) map[string]string {
	return map[string]string{
		filepath.Join("msp", "signcerts", "cert.pem"): creds.SignCert,
		filepath.Join("msp", "keystore", "key.pem"):   creds.SignKey,
		filepath.Join("msp", "cacerts", "ca.pem"):     creds.CACert,
		filepath.Join("tls", "server.crt"):            creds.TLSCert,
		filepath.Join("tls", "server.key"):            creds.TLSKey,
	}
}

// PeerStartupInstructions décrit comment démarrer un peer Hyperledger Fabric
// avec les fichiers générés (le chemin absolu de sortie est substitué par le CLI).
func PeerStartupInstructions(peerID string) string {
	return fmt.Sprintf(`Étapes suivantes :

1. Démarrez le nœud avec ces certificats :

   CORE_PEER_ID=%s \
   CORE_PEER_MSPCONFIGPATH=<out>/msp \
   CORE_PEER_TLS_ENABLED=true \
   CORE_PEER_TLS_CERT_FILE=<out>/tls/server.crt \
   CORE_PEER_TLS_KEY_FILE=<out>/tls/server.key \
   CORE_PEER_TLS_ROOTCERT_FILE=<out>/msp/cacerts/ca.pem \
   peer node start

2. Une fois le nœud démarré, rejoignez un canal :

   # Récupérer le bloc genèse (depuis un peer existant) :
   peer channel fetch 0 genesis.block -c <channel> --orderer <orderer:port>

   # Rejoindre le canal :
   CORE_PEER_ADDRESS=%s:7051 peer channel join -b genesis.block

3. Vérifier que le nœud a bien rejoint :

   peer channel list`, peerID, peerID)
}

// ── peerCAClient — client CA dédié aux opérations peer ───────────────────────

type peerCAClient struct {
	caURL     string
	caName    string
	adminCert []byte
	adminKey  []byte
	http      *http.Client
}

// registerPeer enregistre une identité de type "peer" auprès de la CA.
// Si secret est vide, la CA en génère un automatiquement.
func (c *peerCAClient) registerPeer(ctx context.Context, peerID, secret string) (string, error) {
	return c.registerIdentity(ctx, peerID, "peer", secret)
}

// registerIdentity enregistre une identité d'un type Fabric CA donné
// ("client", "peer", "orderer") auprès de la CA. Utilisé pour le bootstrap
// d'un réseau from scratch (admin d'org, orderer, peers) autant que pour
// l'ajout d'un peer à un réseau existant.
// Si secret est vide, la CA en génère un automatiquement.
func (c *peerCAClient) registerIdentity(ctx context.Context, id, idType, secret string) (string, error) {
	type peerRegisterReq struct {
		ID          string   `json:"id"`
		Type        string   `json:"type"`
		Secret      string   `json:"secret,omitempty"`
		Affiliation string   `json:"affiliation"`
		Attrs       []caAttr `json:"attrs"`
		CAName      string   `json:"caname,omitempty"`
	}
	body := peerRegisterReq{
		ID:     id,
		Type:   idType,
		Secret: secret,
		CAName: c.caName,
		Attrs:  []caAttr{},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	token, err := c.adminToken(data, "/api/v1/register")
	if err != nil {
		return "", err
	}
	resp, err := c.postJSON(ctx, "/api/v1/register", data, token)
	if err != nil {
		return "", err
	}
	var r caRegisterResp
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}
	if !r.Success {
		msg := "échec inconnu"
		if len(r.Errors) > 0 {
			msg = r.Errors[0].Message
		}
		return "", fmt.Errorf("%s", msg)
	}
	return r.Result.Secret, nil
}

// enroll génère une paire de clés ECDSA, crée un CSR et obtient un certificat de la CA.
//   - profile="" → certificats de signature (MSP)
//   - profile="tls" → certificats TLS avec SANs (hosts)
func (c *peerCAClient) enroll(ctx context.Context, name, secret, profile string, hosts []string) (certPEM, keyPEM, caCertPEM string, err error) {
	privKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		return "", "", "", fmt.Errorf("génération clé : %w", e)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: name},
	}
	if len(hosts) > 0 {
		csrTemplate.DNSNames, csrTemplate.IPAddresses = splitSANHosts(hosts)
	}
	csrDER, e := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if e != nil {
		return "", "", "", fmt.Errorf("création CSR : %w", e)
	}
	csrPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	type enrollCSR struct {
		Hosts []string `json:"hosts,omitempty"`
	}
	type enrollReq struct {
		CertificateRequest string     `json:"certificate_request"`
		Profile            string     `json:"profile,omitempty"`
		CAName             string     `json:"caname,omitempty"`
		CSR                *enrollCSR `json:"csr,omitempty"`
	}
	reqBody := enrollReq{
		CertificateRequest: string(csrPEMBytes),
		Profile:            profile,
		CAName:             c.caName,
	}
	if len(hosts) > 0 {
		reqBody.CSR = &enrollCSR{Hosts: hosts}
	}
	data, e := json.Marshal(reqBody)
	if e != nil {
		return "", "", "", e
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(name + ":" + secret))
	resp, e := c.postJSON(ctx, "/api/v1/enroll", data, "Basic "+basicAuth)
	if e != nil {
		return "", "", "", e
	}
	var r caEnrollResp
	if e := json.Unmarshal(resp, &r); e != nil {
		return "", "", "", e
	}
	if !r.Success {
		msg := "échec inconnu"
		if len(r.Errors) > 0 {
			msg = r.Errors[0].Message
		}
		return "", "", "", fmt.Errorf("%s", msg)
	}

	// Décoder le certificat — base64(PEM), cf. decodeCAEnrollCert.
	certPEM = decodeCAEnrollCert(r.Result.Cert)

	keyDER, e := x509.MarshalECPrivateKey(privKey)
	if e != nil {
		return "", "", "", fmt.Errorf("encodage clé : %w", e)
	}
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	caCert := decodeCAEnrollCert(r.Result.ServerInfo.CAChain)

	return certPEM, string(keyBytes), caCert, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (c *peerCAClient) postJSON(ctx context.Context, path string, body []byte, auth string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.caURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CA %s : %w", path, err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// adminToken calcule le token Fabric CA admin (cf. fabricCAToken, ca_client.go,
// pour l'algorithme exact vérifié sur le code source upstream de fabric-ca).
func (c *peerCAClient) adminToken(body []byte, endpoint string) (string, error) {
	if len(c.adminCert) == 0 || len(c.adminKey) == 0 {
		return "", fmt.Errorf("credentials admin manquants — configurez CertPath et KeyPath dans le profil réseau")
	}
	block, _ := pem.Decode(c.adminKey)
	if block == nil {
		return "", fmt.Errorf("clé admin PEM invalide")
	}
	privKey, err := parseECKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse clé admin : %w", err)
	}
	return fabricCAToken(c.adminCert, privKey, "POST", endpoint, body)
}

// parseECKey tente SEC1 puis PKCS8.
func parseECKey(der []byte) (*ecdsa.PrivateKey, error) {
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("la clé n'est pas ECDSA")
	}
	return ecKey, nil
}

// splitSANHosts distingue IPs et noms DNS pour les SANs.
func splitSANHosts(hosts []string) (dnsNames []string, ips []net.IP) {
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			ips = append(ips, ip)
		} else {
			dnsNames = append(dnsNames, h)
		}
	}
	return
}

// buildPeerHTTPClient construit un client HTTP avec vérification TLS optionnelle.
func buildPeerHTTPClient(tlsCACertPEM []byte) *http.Client {
	tlsCfg := &tls.Config{}
	if len(tlsCACertPEM) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(tlsCACertPEM)
		tlsCfg.RootCAs = pool
	} else {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec // dev sans cert TLS
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}
}
