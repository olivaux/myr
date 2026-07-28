// domain/model/ports.go — interfaces secondaires (out-ports) du domaine model
package model

// BlockchainPort est le contrat que tout adapter blockchain doit respecter.
// Intentionnellement minimal pour rester agnostique de la technologie sous-jacente.
type BlockchainPort interface {
	StoreModelRecord(m *Model3D) error
	// GetModelRecord récupère un modèle par son ID sur le canal indiqué.
	// channelID="" utilise le canal par défaut configuré dans l'adapter.
	GetModelRecord(id, channelID string) (*Model3D, error)
	ListModelRecords(channelID string) ([]*Model3D, error)
	// VerifyIntegrity vérifie l'intégrité d'un modèle sur le canal indiqué.
	// channelID="" utilise le canal par défaut configuré dans l'adapter.
	VerifyIntegrity(id, hash, channelID string) (bool, error)
}

// FileStoragePort est le contrat que tout adapter de stockage de fichiers doit respecter.
type FileStoragePort interface {
	Upload(filePath string) (hash string, err error)
	Download(hash string, destPath string) error
	Delete(hash string) error
}

// ConnectionStore gère les connexions d'assemblage entre assets.
// Implémenté par les adapters qui supportent le graphe (ex: JSONBlockchain).
// Non requis par FabricBlockchain.
type ConnectionStore interface {
	SaveConnection(c *Connection) error
	UpdateConnection(c *Connection) error
	RemoveConnection(id string) error
	ListConnections() ([]*Connection, error)
}

// ThumbnailStore persiste les miniatures STL (data URL base64).
// Séparé de Model3D pour ne pas alourdir les réponses blockchain.
type ThumbnailStore interface {
	SaveThumbnail(assetID, dataURL string) error
	GetThumbnail(assetID string) (string, error)
}

// OGImageFetcher récupère l'image représentative (balise og:image) d'une page web
// et la retourne encodée en data URL base64. Permet de redériver la miniature d'un
// asset depuis sa source durable (Model3D.Links) quand aucun fichier 3D n'est fourni —
// voir Service.RegenerateThumbnail.
type OGImageFetcher interface {
	FetchOGImage(pageURL string) (dataURL string, err error)
}

// DraftStore persiste les Model3D créés en brouillon (AddRequest.Draft=true) avant
// leur soumission à la blockchain — cycle brouillon → soumission généralisé à tout
// asset (composant ou module). Non requis par Fabric ou NoOpBlockchain — activé
// uniquement quand des composants sont créés en brouillon.
type DraftStore interface {
	SaveDraft(m *Model3D) error
	GetDraft(id string) (*Model3D, error)
	RemoveDraft(id string) error
	ListDrafts(channelID string) ([]*Model3D, error)
}

// InterfaceStore gère les interfaces physiques des assets et le vocabulaire de référence.
// Non requis par Fabric ou NoOpBlockchain — activé uniquement en mode GUI.
type InterfaceStore interface {
	// Interfaces sur un asset
	SaveInterface(iface *AssetInterface) error
	RemoveInterface(id string) error
	ListInterfacesForAsset(assetID string) ([]*AssetInterface, error)
	GetInterface(id string) (*AssetInterface, error)

	// Vocabulaire extensible (catégories, types, unités)
	GetRefs() (*InterfaceRefs, error)
	AddRefCategory(cat string) error
	AddRefType(cat, typeName string) error
	AddRefUnit(cat, unit string) error
}
