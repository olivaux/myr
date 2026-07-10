// domain/network/parser.go
package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// connectionProfile couvre deux formats :
//  1. Format natif myr (flat JSON, tous les champs en clair)
//  2. Format standard Hyperledger Fabric (organizations / peers / channels)
type connectionProfile struct {
	// ── Format natif myr ──────────────────────────────────────────────────
	Name         string `json:"name"`
	PeerEndpoint string `json:"peerEndpoint"`
	GatewayPeer  string `json:"gatewayPeer"`
	MspID        string `json:"mspId"`
	CertPath     string `json:"certPath"`
	KeyPath      string `json:"keyPath"`
	TLSCertPath  string `json:"tlsCertPath"`
	Channel      string `json:"channel"`
	Chaincode    string `json:"chaincode"`

	// ── Format standard Fabric ────────────────────────────────────────────
	Organizations map[string]struct {
		MSPID  string   `json:"mspid"`
		Peers  []string `json:"peers"`
		Signed struct {
			Path string `json:"path"`
		} `json:"signedCert"`
		AdminKey struct {
			Path string `json:"path"`
		} `json:"adminPrivateKey"`
	} `json:"organizations"`

	Peers map[string]struct {
		URL        string `json:"url"`
		TLSCACerts struct {
			Path string `json:"path"`
			Pem  string `json:"pem"` // PEM inline (format standard Fabric)
		} `json:"tlsCACerts"`
		GRPCOptions struct {
			SSLOverride string `json:"ssl-target-name-override"`
		} `json:"grpcOptions"`
	} `json:"peers"`

	Channels map[string]json.RawMessage `json:"channels"`

	// Autorités de certification (format standard Fabric)
	CertificateAuthorities map[string]struct {
		URL        string `json:"url"`
		CAName     string `json:"caName"`
		TLSCACerts struct {
			Pem string `json:"pem"`
		} `json:"tlsCACerts"`
	} `json:"certificateAuthorities"`
}

// ParseConnectionProfile lit un fichier JSON de profil de connexion Fabric
// et retourne un NetworkProfile prêt à être passé à NetworkService.Add().
// Deux formats sont supportés : natif myr et standard Hyperledger Fabric.
func ParseConnectionProfile(filePath string) (*NetworkProfile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier : %w", err)
	}

	var cp connectionProfile
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parsing JSON : %w", err)
	}

	np := &NetworkProfile{
		Name:          cp.Name,
		PeerEndpoint:  cp.PeerEndpoint,
		GatewayPeer:   cp.GatewayPeer,
		MSPID:         cp.MspID,
		CertPath:      cp.CertPath,
		KeyPath:       cp.KeyPath,
		TLSCertPath:   cp.TLSCertPath,
		FabricChannel: cp.Channel,
		ChaincodeName: cp.Chaincode,
	}

	// ── Complète depuis le format standard Fabric ─────────────────────────

	// Peer endpoint + gateway + TLS cert
	if np.PeerEndpoint == "" {
		for peerName, peer := range cp.Peers {
			url := strings.TrimPrefix(peer.URL, "grpcs://")
			url = strings.TrimPrefix(url, "grpc://")
			np.PeerEndpoint = url

			if np.GatewayPeer == "" {
				if peer.GRPCOptions.SSLOverride != "" {
					np.GatewayPeer = peer.GRPCOptions.SSLOverride
				} else {
					np.GatewayPeer = peerName
				}
			}
			if np.TLSCertPath == "" {
				if peer.TLSCACerts.Path != "" {
					np.TLSCertPath = peer.TLSCACerts.Path
				} else if peer.TLSCACerts.Pem != "" {
					// PEM inline → sauvegarde dans ./data/certs/
					certDir := "./data/certs"
					os.MkdirAll(certDir, 0755)
					certFile := filepath.Join(certDir, np.Name+"-tls.pem")
					pem := strings.ReplaceAll(peer.TLSCACerts.Pem, `\n`, "\n")
					if err := os.WriteFile(certFile, []byte(pem), 0644); err == nil {
						np.TLSCertPath = certFile
					}
				}
			}
			break // premier peer uniquement
		}
	}

	// MSP ID + certificat client + clé privée
	if np.MSPID == "" {
		for _, org := range cp.Organizations {
			np.MSPID = org.MSPID
			if np.CertPath == "" {
				np.CertPath = org.Signed.Path
			}
			if np.KeyPath == "" {
				np.KeyPath = org.AdminKey.Path
			}
			break // première organisation uniquement
		}
	}

	// CA endpoint + caName — depuis certificateAuthorities
	if np.CAEndpoint == "" {
		for caKey, ca := range cp.CertificateAuthorities {
			np.CAEndpoint = ca.URL
			if ca.CAName != "" {
				np.CAName = ca.CAName
			} else {
				np.CAName = caKey
			}
			break // première CA uniquement
		}
	}

	// Canaux — on conserve tous les noms; le premier sert de canal par défaut
	for channelName := range cp.Channels {
		np.Channels = append(np.Channels, channelName)
	}
	if np.FabricChannel == "" && len(np.Channels) > 0 {
		np.FabricChannel = np.Channels[0]
	}

	if np.Name == "" {
		return nil, fmt.Errorf("le fichier ne contient pas de champ « name »")
	}
	if np.PeerEndpoint == "" {
		return nil, fmt.Errorf("impossible de déterminer le peer endpoint depuis ce fichier")
	}

	return np, nil
}
