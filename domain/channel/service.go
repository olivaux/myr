// domain/channel/service.go
package channel

import (
	"fmt"
	"net"
	"regexp"
)

// mspIDPattern valide le format Fabric d'un MSP ID : [a-zA-Z0-9_.-]{1,128}
var mspIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_.\-]{1,128}$`)

// Service expose les canaux Fabric en lecture et les opérations d'administration
// réseau (UCADM01/03/04). La création et la liste des canaux restent gérées par
// l'admin réseau ; les opérations d'écriture délèguent à ChannelConfigPort.
type Service struct {
	repo       ChannelRepo
	fabricConf ChannelConfigPort // nil = mode sans Fabric (DC-CLI-03)
}

func NewService(repo ChannelRepo) *Service {
	return &Service{repo: repo}
}

// WithFabricConfig injecte l'adapter Fabric utilisé par AddOrganisation/AddNode/RemoveNode.
// Sans cet appel, ces méthodes retournent ErrFabricUnavailable.
func (s *Service) WithFabricConfig(p ChannelConfigPort) *Service {
	s.fabricConf = p
	return s
}

func (s *Service) Get(id string) (*Channel, error) {
	ch, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return ch, nil
}

func (s *Service) List() ([]*Channel, error) {
	return s.repo.FindAll()
}

// AddOrganisation ajoute une organisation au canal Fabric (UCADM01).
// Valide le format du MSPID avant soumission (RM07 : valider AVANT transaction immuable).
func (s *Service) AddOrganisation(channelID string, org Organization) error {
	if !mspIDPattern.MatchString(org.MSPID) {
		return fmt.Errorf("%w: %q", ErrInvalidMSPID, org.MSPID)
	}
	if s.fabricConf == nil {
		return ErrFabricUnavailable
	}
	return s.fabricConf.AddOrganisation(channelID, org)
}

// AddNode ajoute un peer ou orderer au canal Fabric (UCADM03).
// Valide le format host:port de l'adresse avant soumission.
func (s *Service) AddNode(channelID string, nodeType NodeType, addr, orgMSP string, certs NodeCerts) error {
	if nodeType != NodeTypePeer && nodeType != NodeTypeOrderer {
		return fmt.Errorf("type de nœud invalide : %q (attendu peer ou orderer)", nodeType)
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("adresse de nœud invalide %q : utilisez le format host:port", addr)
	}
	if s.fabricConf == nil {
		return ErrFabricUnavailable
	}
	return s.fabricConf.AddNode(channelID, nodeType, addr, orgMSP, certs)
}

// RemoveNode retire administrativement un nœud du canal Fabric (UCADM04).
// Le seuil minimum de 3 nœuds actifs (RM27) et l'appartenance au canal sont
// vérifiés côté ChannelConfigPort, qui a la visibilité sur la configuration Fabric réelle.
// #incoherence — RM27 n'est donc vérifiée que par l'adapter Fabric, jamais ici
// dans le domaine, contrairement à l'ajout de nœud (domain/network/service.go)
// — voir specs/roadmap_dev.md § Écarts — revue de code, E6.
func (s *Service) RemoveNode(channelID, addr string) error {
	if s.fabricConf == nil {
		return ErrFabricUnavailable
	}
	return s.fabricConf.RemoveNode(channelID, addr)
}
