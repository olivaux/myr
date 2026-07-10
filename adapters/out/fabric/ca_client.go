// adapters/out/fabric/ca_client.go
// Implémente domain/identity.CAPort via l'API REST de la Fabric CA.
package fabric

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"

	"myr/domain/identity"
)

// CAClient implémente identity.CAPort.
type CAClient struct {
	caURL     string // ex: "https://XXX.XXX.XXX.XXX:YYYY"
	caName    string // ex: "ca-org1"
	adminCert []byte // PEM du certificat admin
	adminKey  []byte // PEM de la clé privée admin
	tlsPool   *x509.CertPool
	http      *http.Client
}

// NewCAClient construit un CAClient depuis le Config du réseau actif.
// Retourne nil (sans erreur) si CAEndpoint n'est pas configuré.
func NewCAClient(cfg Config) (*CAClient, error) {
	if cfg.CAEndpoint == "" {
		return nil, nil // CA non configuré — pas d'erreur fatale
	}

	// Charger le certificat TLS pour vérifier le serveur CA
	pool := x509.NewCertPool()
	if cfg.TLSCertPath != "" {
		tlsPEM, err := os.ReadFile(cfg.TLSCertPath)
		if err != nil {
			return nil, fmt.Errorf("ca_client: TLS cert : %w", err)
		}
		pool.AppendCertsFromPEM(tlsPEM)
	} else {
		// Pas de cert TLS configuré — accepter n'importe quel cert (dev)
		pool = nil
	}

	// Utiliser les credentials CA admin dédiés si configurés,
	// sinon fallback sur le cert/key peer (utile en dev).
	adminCertPath := cfg.CAAdminCertPath
	if adminCertPath == "" {
		adminCertPath = cfg.CertPath
	}
	adminKeyPath := cfg.CAAdminKeyPath
	if adminKeyPath == "" {
		adminKeyPath = cfg.KeyPath
	}

	var adminCert, adminKey []byte
	if adminCertPath != "" {
		var err error
		adminCert, err = os.ReadFile(adminCertPath)
		if err != nil {
			return nil, fmt.Errorf("ca_client: cert admin : %w", err)
		}
	}
	if adminKeyPath != "" {
		var err error
		adminKey, err = os.ReadFile(adminKeyPath)
		if err != nil {
			return nil, fmt.Errorf("ca_client: clé admin : %w", err)
		}
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: pool == nil}
	if pool != nil {
		tlsCfg.RootCAs = pool
	}

	return &CAClient{
		caURL:     cfg.CAEndpoint,
		caName:    cfg.CAName,
		adminCert: adminCert,
		adminKey:  adminKey,
		tlsPool:   pool,
		http: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}, nil
}

// decodeCAEnrollCert normalise un champ certificat renvoyé par l'API Fabric CA
// (/api/v1/enroll et /api/v1/reenroll — champs Cert et ServerInfo.CAChain).
// Ces champs sont du base64(PEM) — PAS du base64(DER) — un piège classique :
// re-envelopper le contenu décodé dans un nouveau bloc PEM (en le traitant à
// tort comme du DER) produit un certificat corrompu ("x509: malformed
// certificate" côté serveur lors d'une prochaine utilisation comme token
// d'authentification). On décode une seule fois et on retourne tel quel si
// le résultat est déjà du PEM ; seul un vrai DER brut est ré-enveloppé.
func decodeCAEnrollCert(raw string) string {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw // déjà en PEM tel quel
	}
	if bytes.Contains(decoded, []byte("-----BEGIN")) {
		return string(decoded)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: decoded}))
}

func roleOrDefault(role string) string {
	switch role {
	case identity.RoleContributor, identity.RoleAuditor, identity.RoleAdmin:
		return role
	default:
		return identity.RoleReader
	}
}

// ─── Register ────────────────────────────────────────────────────────────────

type caRegisterReq struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Affiliation string   `json:"affiliation"`
	Attrs       []caAttr `json:"attrs"`
	CAName      string   `json:"caname,omitempty"`
}
type caAttr struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	ECert bool   `json:"ecert"`
}
type caRegisterResp struct {
	Success bool `json:"success"`
	Result  struct {
		Secret string `json:"secret"`
	} `json:"result"`
	Errors []struct{ Message string } `json:"errors"`
}

func (c *CAClient) Register(ctx context.Context, req identity.RegisterRequest) (string, error) {
	body := caRegisterReq{
		ID:     req.Name,
		Type:   "client",
		CAName: c.caName,
		Attrs: []caAttr{
			{Name: "Myr.status", Value: identity.StatusPending, ECert: true},
			{Name: "Myr.role", Value: roleOrDefault(req.Role), ECert: true},
			{Name: "Myr.channels", Value: "green,red,blue", ECert: true},
			{Name: "Myr.displayName", Value: req.DisplayName, ECert: false},
			{Name: "Myr.legalName", Value: req.LegalName, ECert: false},
			{Name: "Myr.email", Value: req.Email, ECert: false},
			{Name: "Myr.country", Value: req.Country, ECert: false},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	token, err := c.adminToken(data, "POST", "/api/v1/register")
	if err != nil {
		return "", fmt.Errorf("token admin : %w", err)
	}

	resp, err := c.post(ctx, "/api/v1/register", data, token)
	if err != nil {
		return "", err
	}
	var r caRegisterResp
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}
	if !r.Success {
		if len(r.Errors) > 0 {
			return "", fmt.Errorf("CA register : %s", r.Errors[0].Message)
		}
		return "", fmt.Errorf("CA register : échec inconnu")
	}
	return r.Result.Secret, nil
}

// ─── Enroll ──────────────────────────────────────────────────────────────────

type caEnrollReq struct {
	CertificateRequest string `json:"certificate_request"`
	Profile            string `json:"profile"`
	CAName             string `json:"caname,omitempty"`
}
type caEnrollResp struct {
	Success bool `json:"success"`
	Result  struct {
		Cert       string `json:"Cert"`
		ServerInfo struct {
			CAChain string `json:"CAChain"`
		} `json:"ServerInfo"`
	} `json:"result"`
	Errors []struct{ Message string } `json:"errors"`
}

func (c *CAClient) Enroll(ctx context.Context, name, secret string) (certPEM, keyPEM, caCertPEM string, err error) {
	// 1. Générer une paire de clés ECDSA P-256
	privKey, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		return "", "", "", fmt.Errorf("génération clé : %w", e)
	}

	// 2. Créer un CSR
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: name},
	}
	csrDER, e := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if e != nil {
		return "", "", "", fmt.Errorf("création CSR : %w", e)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	// 3. Préparer la requête
	body := caEnrollReq{
		CertificateRequest: string(csrPEM),
		CAName:             c.caName,
	}
	data, e := json.Marshal(body)
	if e != nil {
		return "", "", "", e
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(name + ":" + secret))
	resp, e := c.post(ctx, "/api/v1/enroll", data, "Basic "+basicAuth)
	if e != nil {
		return "", "", "", e
	}

	var r caEnrollResp
	if e := json.Unmarshal(resp, &r); e != nil {
		return "", "", "", e
	}
	if !r.Success {
		if len(r.Errors) > 0 {
			return "", "", "", fmt.Errorf("CA enroll : %s", r.Errors[0].Message)
		}
		return "", "", "", fmt.Errorf("CA enroll : échec inconnu")
	}

	// 4. Décoder le certificat (base64(PEM), cf. decodeCAEnrollCert)
	certPEM = decodeCAEnrollCert(r.Result.Cert)

	// 5. Encoder la clé privée en PEM
	keyDER, e := x509.MarshalECPrivateKey(privKey)
	if e != nil {
		return "", "", "", fmt.Errorf("encodage clé : %w", e)
	}
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// 6. Décoder la chaîne CA
	caCertPEM = decodeCAEnrollCert(r.Result.ServerInfo.CAChain)

	return certPEM, string(keyPEMBytes), caCertPEM, nil
}

// ─── GetStatus ───────────────────────────────────────────────────────────────

type caIdentityResp struct {
	Success bool `json:"success"`
	Result  struct {
		ID    string   `json:"id"`
		Attrs []caAttr `json:"attrs"`
	} `json:"result"`
	Errors []struct{ Message string } `json:"errors"`
}

func (c *CAClient) GetStatus(ctx context.Context, name string) (string, error) {
	token, err := c.adminToken(nil, "GET", "/api/v1/identities/"+name)
	if err != nil {
		return "", fmt.Errorf("token admin : %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.caURL+"/api/v1/identities/"+name, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("CA getStatus : %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r caIdentityResp
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if !r.Success {
		if len(r.Errors) > 0 {
			return "", fmt.Errorf("CA getStatus : %s", r.Errors[0].Message)
		}
		return "", fmt.Errorf("CA getStatus : échec inconnu")
	}
	for _, attr := range r.Result.Attrs {
		if attr.Name == "Myr.status" {
			return attr.Value, nil
		}
	}
	return identity.StatusPending, nil
}

// ─── UpdateAttributes ────────────────────────────────────────────────────────

type caUpdateReq struct {
	Attrs []caAttr `json:"attrs"`
}
type caUpdateResp struct {
	Success bool                       `json:"success"`
	Result  json.RawMessage            `json:"result"` // forme variable selon succès/erreur — non exploité ici
	Errors  []struct{ Message string } `json:"errors"`
}

// UpdateAttributes modifie les attributs enregistrés d'une identité existante
// (PUT /api/v1/identities/{id}). Le changement ne s'applique qu'aux certificats
// émis après l'appel — un ré-enrôlement est nécessaire pour l'appliquer au
// certificat actif de l'identité (comportement natif de la Fabric CA).
func (c *CAClient) UpdateAttributes(ctx context.Context, name string, attrs map[string]string) error {
	caAttrs := make([]caAttr, 0, len(attrs))
	for k, v := range attrs {
		caAttrs = append(caAttrs, caAttr{Name: k, Value: v, ECert: true})
	}
	body := caUpdateReq{Attrs: caAttrs}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	path := "/api/v1/identities/" + name
	token, err := c.adminToken(data, "PUT", path)
	if err != nil {
		return fmt.Errorf("token admin : %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.caURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("CA updateAttributes : %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var r caUpdateResp
	if err := json.Unmarshal(respBody, &r); err != nil {
		return err
	}
	if !r.Success {
		if len(r.Errors) > 0 {
			return fmt.Errorf("CA updateAttributes : %s", r.Errors[0].Message)
		}
		return fmt.Errorf("CA updateAttributes : échec inconnu")
	}
	return nil
}

// ─── ReEnroll ────────────────────────────────────────────────────────────────

func (c *CAClient) ReEnroll(ctx context.Context, name, certPEM, keyPEM string) (string, error) {
	// Charger la clé privée existante
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return "", fmt.Errorf("re-enroll : clé privée invalide")
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("re-enroll : parse clé : %w", err)
	}

	// Créer un CSR avec la même clé
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: name},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privKey)
	if err != nil {
		return "", fmt.Errorf("re-enroll CSR : %w", err)
	}
	csrPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	body := caEnrollReq{
		CertificateRequest: string(csrPEMBytes),
		CAName:             c.caName,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	// Auth via le certificat actuel
	token, err := c.certToken([]byte(certPEM), privKey, data, "POST", "/api/v1/reenroll")
	if err != nil {
		return "", fmt.Errorf("re-enroll token : %w", err)
	}

	resp, err := c.post(ctx, "/api/v1/reenroll", data, token)
	if err != nil {
		return "", err
	}
	var r caEnrollResp
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}
	if !r.Success {
		if len(r.Errors) > 0 {
			return "", fmt.Errorf("CA re-enroll : %s", r.Errors[0].Message)
		}
		return "", fmt.Errorf("CA re-enroll : échec inconnu")
	}

	return decodeCAEnrollCert(r.Result.Cert), nil
}

// ─── Helpers HTTP ─────────────────────────────────────────────────────────────

func (c *CAClient) post(ctx context.Context, path string, body []byte, auth string) ([]byte, error) {
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

// adminToken calcule le token d'authentification admin (cert + signature ECDSA).
// Format Fabric CA : base64(certPEM) + "." + base64(sign(SHA256(b64body + "." + endpoint)))
func (c *CAClient) adminToken(body []byte, method, endpoint string) (string, error) {
	if len(c.adminCert) == 0 || len(c.adminKey) == 0 {
		return "", fmt.Errorf("%w", identity.ErrNoAdminCredentials)
	}

	block, _ := pem.Decode(c.adminKey)
	if block == nil {
		return "", fmt.Errorf("clé admin : PEM invalide")
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("clé admin : parse : %w", err)
	}

	return c.certToken(c.adminCert, privKey, body, method, endpoint)
}

// certToken calcule un token Fabric CA avec un cert et une clé privée donnés.
func (c *CAClient) certToken(certPEM []byte, privKey *ecdsa.PrivateKey, body []byte, method, uri string) (string, error) {
	return fabricCAToken(certPEM, privKey, method, uri, body)
}

// fabricCAToken calcule le token d'authentification Fabric CA (algorithme exact
// de hyperledger/fabric-ca util.GenECDSAToken — vérifié sur le code source
// upstream, cf. lib/client/credential/x509/credential.go CreateToken) :
//
//	payload = method + "." + base64(uri) + "." + base64(body) + "." + base64(cert)
//	token   = base64(cert) + "." + base64(sign(SHA256(payload)))
//
// uri doit correspondre exactement à req.URL.RequestURI() côté Fabric CA,
// c.à.d. le chemin tel qu'utilisé dans la requête HTTP (ex: "/api/v1/register"),
// sans host ni schéma. La signature doit être normalisée en low-S : le
// vérificateur BCCSP de Fabric rejette toute signature ECDSA dont S dépasse la
// moitié de l'ordre de la courbe (protection anti-malléabilité) — une omission
// silencieuse ici fait échouer ~50% des requêtes de façon aléatoire.
func fabricCAToken(certPEM []byte, privKey *ecdsa.PrivateKey, method, uri string, body []byte) (string, error) {
	b64Cert := base64.StdEncoding.EncodeToString(certPEM)
	b64Body := base64.StdEncoding.EncodeToString(body)
	b64URI := base64.StdEncoding.EncodeToString([]byte(uri))

	payload := method + "." + b64URI + "." + b64Body + "." + b64Cert
	hash := sha256.Sum256([]byte(payload))

	sig, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", err
	}
	sig, err = toLowS(privKey.Curve, sig)
	if err != nil {
		return "", fmt.Errorf("normalisation low-S : %w", err)
	}
	b64Sig := base64.StdEncoding.EncodeToString(sig)
	return b64Cert + "." + b64Sig, nil
}

// toLowS réencode une signature ECDSA ASN.1 avec un S canonique (≤ moitié de
// l'ordre de la courbe), comme l'exige le vérificateur BCCSP de Fabric.
func toLowS(curve elliptic.Curve, sigASN1 []byte) ([]byte, error) {
	var sig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(sigASN1, &sig); err != nil {
		return nil, err
	}
	halfOrder := new(big.Int).Rsh(curve.Params().N, 1)
	if sig.S.Cmp(halfOrder) == 1 {
		sig.S = new(big.Int).Sub(curve.Params().N, sig.S)
	}
	return asn1.Marshal(sig)
}
