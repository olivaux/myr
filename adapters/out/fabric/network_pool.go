// adapters/out/fabric/network_pool.go — pool de connexions Fabric multi-réseau
package fabric

import (
	"fmt"
	"log"
	"sync"

	"myr/domain/model"
	"myr/domain/network"
)

// NetworkPool gère un ensemble de GatewayClient ouverts, un par réseau Fabric.
// Thread-safe — peut être utilisé depuis plusieurs goroutines simultanées.
type NetworkPool struct {
	mu       sync.RWMutex
	gateways map[string]*GatewayClient // networkID → gateway
}

// NewNetworkPool crée un pool vide.
func NewNetworkPool() *NetworkPool {
	return &NetworkPool{gateways: make(map[string]*GatewayClient)}
}

// Add ajoute (ou remplace) un gateway dans le pool.
func (p *NetworkPool) Add(networkID string, gw *GatewayClient) {
	p.mu.Lock()
	if old := p.gateways[networkID]; old != nil {
		old.Close()
	}
	p.gateways[networkID] = gw
	p.mu.Unlock()
}

// Get retourne le gateway associé au networkID, ou nil s'il est absent.
func (p *NetworkPool) Get(networkID string) *GatewayClient {
	p.mu.RLock()
	gw := p.gateways[networkID]
	p.mu.RUnlock()
	return gw
}

// Remove ferme et supprime un gateway du pool.
func (p *NetworkPool) Remove(networkID string) {
	p.mu.Lock()
	if gw := p.gateways[networkID]; gw != nil {
		gw.Close()
	}
	delete(p.gateways, networkID)
	p.mu.Unlock()
}

// IDs retourne les identifiants de tous les réseaux connectés.
func (p *NetworkPool) IDs() []string {
	p.mu.RLock()
	ids := make([]string, 0, len(p.gateways))
	for id := range p.gateways {
		ids = append(ids, id)
	}
	p.mu.RUnlock()
	return ids
}

// Close ferme toutes les connexions et vide le pool.
func (p *NetworkPool) Close() {
	p.mu.Lock()
	for _, gw := range p.gateways {
		gw.Close()
	}
	p.gateways = make(map[string]*GatewayClient)
	p.mu.Unlock()
}

// BlockchainFor retourne un model.BlockchainPort connecté au réseau spécifié.
// Si le réseau est absent du pool, retourne un blockchain hors-ligne qui log l'erreur.
// Cette méthode implémente l'interface rest.blockchainRouter.
func (p *NetworkPool) BlockchainFor(networkID string) model.BlockchainPort {
	gw := p.Get(networkID)
	if gw == nil {
		return &offlineBlockchain{networkID: networkID}
	}
	return NewFabricBlockchain(gw)
}

// LoadFromProfiles tente d'ouvrir une connexion pour chaque profil réseau actif.
// Les échecs sont loggés mais non fatals — le pool démarre avec les réseaux disponibles.
func (p *NetworkPool) LoadFromProfiles(profiles []*network.NetworkProfile) {
	for _, np := range profiles {
		cfg := ConfigFromProfile(np)
		gw, err := NewGatewayClient(cfg)
		if err != nil {
			log.Printf("réseau %s (%s) : connexion impossible : %v", np.Name, np.ID, err)
			continue
		}
		p.Add(np.ID, gw)
		log.Printf("réseau %s (%s) : connecté → %s", np.Name, np.ID, np.PeerEndpoint)
	}
}

// ── offlineBlockchain ─────────────────────────────────────────────────────────

// offlineBlockchain est retourné quand un réseau demandé n'est pas dans le pool.
type offlineBlockchain struct {
	networkID string
}

func (o *offlineBlockchain) err() error {
	return fmt.Errorf("réseau %q non disponible dans le pool de connexions", o.networkID)
}

func (o *offlineBlockchain) StoreModelRecord(*model.Model3D) error               { return o.err() }
func (o *offlineBlockchain) GetModelRecord(_, _ string) (*model.Model3D, error)  { return nil, o.err() }
func (o *offlineBlockchain) ListModelRecords(_ string) ([]*model.Model3D, error) { return nil, o.err() }
func (o *offlineBlockchain) VerifyIntegrity(_, _, _ string) (bool, error)        { return false, o.err() }
