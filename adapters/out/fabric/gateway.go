// adapters/out/fabric/gateway.go
package fabric

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GatewayClient encapsule la connexion gRPC + Gateway Fabric.
// Il est partagé par tous les adapters Fabric.
type GatewayClient struct {
	gw   *client.Gateway
	conn *grpc.ClientConn
	cfg  Config
}

// NewGatewayClient ouvre la connexion vers le peer Fabric et retourne un GatewayClient.
func NewGatewayClient(cfg Config) (*GatewayClient, error) {
	// Validation préalable — retourne des erreurs lisibles avant d'essayer d'ouvrir des fichiers
	switch {
	case cfg.PeerEndpoint == "":
		return nil, fmt.Errorf("peer_endpoint non configuré")
	case cfg.TLSCertPath == "":
		return nil, fmt.Errorf("tls_cert_path non configuré (certificat TLS du peer)")
	case cfg.CertPath == "":
		return nil, fmt.Errorf("cert_path non configuré (certificat client X.509)")
	case cfg.KeyPath == "":
		return nil, fmt.Errorf("key_path non configuré (clé privée client)")
	case cfg.MSPID == "":
		return nil, fmt.Errorf("msp_id non configuré")
	case cfg.FabricChannel == "":
		return nil, fmt.Errorf("fabric_channel non configuré")
	case cfg.ChaincodeName == "":
		return nil, fmt.Errorf("chaincode_name non configuré")
	}

	// 1. Connexion gRPC (TLS)
	conn, err := newGRPCConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("fabric gateway: connexion gRPC : %w", err)
	}

	// 2. Identité cliente X.509
	id, err := newIdentity(cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("fabric gateway: identité : %w", err)
	}

	// 3. Fonction de signature (clé privée)
	sign, err := newSign(cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("fabric gateway: signature : %w", err)
	}

	// 4. Gateway
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("fabric gateway: Connect : %w", err)
	}

	return &GatewayClient{gw: gw, conn: conn, cfg: cfg}, nil
}

// Close libère les ressources.
func (g *GatewayClient) Close() {
	g.gw.Close()
	g.conn.Close()
}

// Contract retourne le contrat chaincode sur le canal par défaut.
func (g *GatewayClient) Contract() ContractCaller {
	return g.ContractFor(g.cfg.FabricChannel)
}

// ContractFor retourne le contrat chaincode sur un canal spécifique.
// Si channelName est vide, utilise le canal par défaut.
func (g *GatewayClient) ContractFor(channelName string) ContractCaller {
	ch := channelName
	if ch == "" {
		ch = g.cfg.FabricChannel
	}
	return g.gw.GetNetwork(ch).GetContract(g.cfg.ChaincodeName)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newGRPCConnection(cfg Config) (*grpc.ClientConn, error) {
	tlsPEM, err := os.ReadFile(cfg.TLSCertPath)
	if err != nil {
		return nil, fmt.Errorf("lecture TLS cert : %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(tlsPEM) {
		return nil, fmt.Errorf("certificat TLS peer invalide")
	}
	tlsCfg := &tls.Config{
		RootCAs:    pool,
		ServerName: cfg.GatewayPeer,
	}
	return grpc.NewClient(cfg.PeerEndpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
	)
}

func newIdentity(cfg Config) (*identity.X509Identity, error) {
	certPEM, err := os.ReadFile(cfg.CertPath)
	if err != nil {
		return nil, fmt.Errorf("lecture certificat client : %w", err)
	}
	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse certificat client : %w", err)
	}
	return identity.NewX509Identity(cfg.MSPID, cert)
}

func newSign(cfg Config) (identity.Sign, error) {
	keyPEM, err := os.ReadFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("lecture clé privée : %w", err)
	}
	pk, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse clé privée : %w", err)
	}
	return identity.NewPrivateKeySign(pk)
}
