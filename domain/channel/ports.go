// domain/channel/ports.go — out-ports du domaine channel
package channel

// ChannelRepo est le contrat de lecture des canaux disponibles.
type ChannelRepo interface {
	FindByID(id string) (*Channel, error)
	FindAll() ([]*Channel, error)
}

// ChannelConfigPort délègue les opérations de configuration de canal vers l'infrastructure Fabric.
// Si nil, le service retourne ErrFabricUnavailable (DC-CLI-03 : mode sans Fabric autorisé).
type ChannelConfigPort interface {
	AddOrganisation(channelID string, org Organization) error
	AddNode(channelID string, nodeType NodeType, addr, orgMSP string, certs NodeCerts) error
	RemoveNode(channelID, addr string) error
}
