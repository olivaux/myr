// domain/network/service.go
package network

import (
	"fmt"
	"net"
	"time"
)

type Service struct {
	repo         Repo
	tester       ConnectionTester    // nil = fallback TCP
	provisioner  PeerProvisioner     // nil = fonctionnalité désactivée
	bootstrapper NetworkBootstrapper // nil = "network create" indisponible
	poolSync     PoolSync            // nil = pas de resynchronisation (redémarrage requis)
}

func NewService(repo Repo, tester ConnectionTester) *Service {
	return &Service{repo: repo, tester: tester}
}

// WithPoolSync injecte la resynchronisation du pool de connexions blockchain :
// tout profil créé, activé, modifié ou supprimé après le démarrage du
// processus est immédiatement répercuté dans le pool, sans redémarrage.
func (s *Service) WithPoolSync(p PoolSync) *Service {
	s.poolSync = p
	return s
}

// syncPool notifie le pool si un PoolSync est injecté — no-op sinon.
func (s *Service) syncPool(n *NetworkProfile) {
	if s.poolSync != nil {
		s.poolSync.Sync(n)
	}
}

// WithProvisioner injecte le provisioner peer (CA adapter).
// Doit être appelé avant AddPeer.
func (s *Service) WithProvisioner(p PeerProvisioner) *Service {
	s.provisioner = p
	return s
}

// WithBootstrapper injecte le bootstrapper de réseau from-scratch (CA + Fabric).
// Doit être appelé avant Create.
func (s *Service) WithBootstrapper(b NetworkBootstrapper) *Service {
	s.bootstrapper = b
	return s
}

// Add crée un nouveau profil réseau.
func (s *Service) Add(name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, channels []string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*NetworkProfile, error) {
	if name == "" {
		return nil, fmt.Errorf("le nom est requis")
	}
	if peerEndpoint == "" {
		return nil, fmt.Errorf("l'adresse du peer est requise")
	}
	if _, _, err := net.SplitHostPort(peerEndpoint); err != nil {
		return nil, fmt.Errorf("adresse du peer invalide %q : utilisez le format host:port ou [ipv6]:port", peerEndpoint)
	}
	id := fmt.Sprintf("net-%d", time.Now().UnixNano())
	n := &NetworkProfile{
		ID:                id,
		Name:              name,
		PeerEndpoint:      peerEndpoint,
		GatewayPeer:       gatewayPeer,
		MSPID:             mspID,
		CertPath:          certPath,
		KeyPath:           keyPath,
		TLSCertPath:       tlsCertPath,
		FabricChannel:     fabricChannel,
		ChaincodeName:     chaincodeName,
		CAEndpoint:        caEndpoint,
		CAName:            caName,
		Channels:          channels,
		AllowAutoGuest:    allowAutoGuest,
		AllowAutoRegister: allowAutoRegister,
		AutoRegisterRole:  autoRegisterRole,
		IsProduction:      isProduction,
		CreatedAt:         time.Now(),
	}
	if err := s.repo.Save(n); err != nil {
		return nil, fmt.Errorf("add network: %w", err)
	}
	s.syncPool(n)
	return n, nil
}

// List retourne tous les profils réseau.
func (s *Service) List() ([]*NetworkProfile, error) {
	return s.repo.FindAll()
}

// GetActive retourne le profil actif, ou nil si aucun.
func (s *Service) GetActive() (*NetworkProfile, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, n := range all {
		if n.Active {
			return n, nil
		}
	}
	return nil, nil
}

// Activate marque le profil id comme actif et désactive les autres.
func (s *Service) Activate(id string) error {
	all, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	found := false
	for _, n := range all {
		wasActive := n.Active
		n.Active = (n.ID == id)
		if n.ID == id {
			found = true
		}
		if wasActive != n.Active {
			if err := s.repo.Save(n); err != nil {
				return err
			}
		}
	}
	if !found {
		return fmt.Errorf("réseau introuvable : %s", id)
	}
	// Le réseau activé peut ne jamais être entré dans le pool (créé après le
	// démarrage du processus, ou connexion précédemment en échec) — resynchronise.
	for _, n := range all {
		if n.ID == id {
			s.syncPool(n)
			break
		}
	}
	return nil
}

// Update modifie les champs d'un profil réseau existant.
func (s *Service) Update(id, name, peerEndpoint, gatewayPeer, mspID, certPath, keyPath, tlsCertPath, fabricChannel, chaincodeName, caEndpoint, caName string, allowAutoGuest, allowAutoRegister bool, autoRegisterRole string, isProduction bool) (*NetworkProfile, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, n := range all {
		if n.ID == id {
			n.Name = name
			n.PeerEndpoint = peerEndpoint
			n.GatewayPeer = gatewayPeer
			n.MSPID = mspID
			n.CertPath = certPath
			n.KeyPath = keyPath
			n.TLSCertPath = tlsCertPath
			n.FabricChannel = fabricChannel
			n.ChaincodeName = chaincodeName
			n.CAEndpoint = caEndpoint
			n.CAName = caName
			n.AllowAutoGuest = allowAutoGuest
			n.AllowAutoRegister = allowAutoRegister
			n.AutoRegisterRole = autoRegisterRole
			n.IsProduction = isProduction
			if err := s.repo.Save(n); err != nil {
				return nil, fmt.Errorf("update network: %w", err)
			}
			// Les identifiants de connexion (endpoint, certs, chaincode...) ont pu
			// changer : reconnecte plutôt que de garder une connexion pool obsolète.
			s.syncPool(n)
			return n, nil
		}
	}
	return nil, fmt.Errorf("réseau introuvable : %s", id)
}

// Delete supprime un profil réseau (le désactive automatiquement s'il est actif).
func (s *Service) Delete(id string) error {
	all, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, n := range all {
		if n.ID == id {
			if err := s.repo.Delete(id); err != nil {
				return err
			}
			if s.poolSync != nil {
				s.poolSync.Remove(id)
			}
			return nil
		}
	}
	return fmt.Errorf("réseau introuvable : %s", id)
}

// AddPeer enregistre un nouveau peer dans la CA du réseau et retourne ses certificats.
func (s *Service) AddPeer(networkID string, req AddPeerRequest) (*PeerCredentials, error) {
	if s.provisioner == nil {
		return nil, fmt.Errorf("provisioning non disponible : aucun adapter CA injecté")
	}
	if req.PeerID == "" {
		return nil, fmt.Errorf("peer-id requis")
	}

	net, err := s.findNetwork(networkID)
	if err != nil {
		return nil, err
	}
	if net.CAEndpoint == "" {
		return nil, fmt.Errorf("ce réseau n'a pas de CA configurée (ca_endpoint manquant)")
	}
	return s.provisioner.RegisterAndProvision(net, req)
}

// Create construit un réseau blockchain from scratch (UCADM02) et enregistre le
// profil résultant. Valide les champs requis puis délègue au bootstrapper injecté.
func (s *Service) Create(req CreateNetworkRequest) (*NetworkProfile, error) {
	if s.bootstrapper == nil {
		return nil, fmt.Errorf("création de réseau indisponible : aucun bootstrapper Fabric injecté")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("le nom du réseau est requis")
	}
	if req.OrgMSPID == "" {
		return nil, fmt.Errorf("l'identifiant de l'organisation (--org-id) est requis")
	}
	if req.ChannelName == "" {
		return nil, fmt.Errorf("le nom du canal (--channel) est requis")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("le domaine (--domain) est requis")
	}
	if req.NumPeers <= 0 {
		req.NumPeers = 2
	}
	if req.NumPeers+1 < 3 {
		return nil, fmt.Errorf("le réseau doit compter au moins 3 nœuds actifs [RM27] — augmentez --peers")
	}

	result, err := s.bootstrapper.Bootstrap(req)
	if err != nil {
		return nil, fmt.Errorf("création du réseau : %w", err)
	}
	if err := s.repo.Save(result.Profile); err != nil {
		return nil, fmt.Errorf("enregistrement du profil réseau : %w", err)
	}
	s.syncPool(result.Profile)
	return result.Profile, nil
}

// findNetwork retourne le profil networkID, ou le profil actif si networkID est vide.
func (s *Service) findNetwork(networkID string) (*NetworkProfile, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, n := range all {
		if networkID == "" && n.Active {
			return n, nil
		}
		if networkID != "" && n.ID == networkID {
			return n, nil
		}
	}
	if networkID == "" {
		return nil, fmt.Errorf("aucun réseau actif — activez-en un avec 'myr network activate <id>'")
	}
	return nil, fmt.Errorf("réseau introuvable : %s", networkID)
}

// TestConnection teste la connexion vers le peer — via le tester Fabric injecté
// si disponible, sinon par simple dial TCP.
func (s *Service) TestConnection(id string) error {
	all, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, n := range all {
		if n.ID == id {
			if s.tester != nil {
				return s.tester.Test(n)
			}
			conn, dialErr := net.DialTimeout("tcp", n.PeerEndpoint, 5*time.Second)
			if dialErr != nil {
				return fmt.Errorf("impossible de joindre %s : %w", n.PeerEndpoint, dialErr)
			}
			conn.Close()
			return nil
		}
	}
	return fmt.Errorf("réseau introuvable : %s", id)
}
