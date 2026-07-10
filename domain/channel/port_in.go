// domain/channel/port_in.go — port d'entrée : interface exposée aux adapters entrants
package channel

// ChannelService est le port d'entrée du domaine channel.
// Il expose les canaux Fabric en lecture et les opérations d'administration réseau.
type ChannelService interface {
	Get(id string) (*Channel, error)
	List() ([]*Channel, error)
	// AddOrganisation ajoute une organisation au canal Fabric (UCADM01).
	// Valide le MSPID et délègue à ChannelConfigPort. Retourne ErrFabricUnavailable si non configuré.
	AddOrganisation(channelID string, org Organization) error
	// AddNode ajoute un peer ou orderer au canal Fabric (UCADM03).
	// Retourne ErrFabricUnavailable si non configuré.
	AddNode(channelID string, nodeType NodeType, addr, orgMSP string, certs NodeCerts) error
	// RemoveNode retire administrativement un nœud du canal Fabric (UCADM04).
	// La vérification du seuil minimum (RM27) est assurée par l'adapter Fabric.
	RemoveNode(channelID, addr string) error
}
