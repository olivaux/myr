// domain/model/port_in.go — port d'entrée : interface exposée aux adapters entrants
package model

// ModelService est le port d'entrée du domaine model.
// Il est implémenté par Service et consommé par les adapters (GUI, CLI, TUI, API).
type ModelService interface {
	// Compatibilité CLI / TUI
	Add(filePath, name, channelID, ownerID string, tags []string) (*Model3D, error)
	// Get récupère un modèle par son ID sur le canal indiqué.
	// channelID="" utilise le canal actif de la session ou le canal par défaut.
	Get(id, channelID string) (*Model3D, error)
	List(channelID string) ([]*Model3D, error)
	// Verify vérifie l'intégrité d'un modèle sur le canal indiqué.
	// channelID="" utilise le canal actif de la session ou le canal par défaut.
	Verify(id, channelID string) (bool, error)

	// Mode GUI — création enrichie
	AddFull(req AddRequest) (*Model3D, error)
	// Submit engage sur la blockchain un composant créé en brouillon (AddRequest.Draft=true) :
	// une seule transaction committe les métadonnées et les interfaces locales
	// (Model3D.Interfaces), puis Status passe à submitted. Généralise SubmitModule
	// à tout Model3D, sans la vérification d'assemblage (RM17, propre aux modules).
	Submit(assetID string) (*Model3D, error)
	Remove(id string) error

	// Connexions d'assemblage (optionnel selon adapter)
	AddConnection(from, to, label string) (*Connection, error)
	AddAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*Connection, error)
	RemoveConnection(id string) error
	ListConnections() ([]*Connection, error)
	GetChildren(parentID string) ([]*Model3D, error)

	// Miniatures STL
	SaveThumbnail(assetID, dataURL string) error
	GetThumbnail(assetID string) (string, error)
	// RegenerateThumbnail redérive la miniature depuis la source durable de l'asset
	// (og:image du premier lien externe) — voir Service.RegenerateThumbnail.
	RegenerateThumbnail(assetID string) (string, error)

	// Interfaces physiques (entrées/sorties d'un asset)
	AddInterface(iface *AssetInterface) error
	UpdateInterface(iface *AssetInterface) error
	RemoveInterface(id string) error
	ListInterfacesForAsset(assetID string) ([]*AssetInterface, error)
	GetInterface(id string) (*AssetInterface, error)
	EnsureVirtualSlot(assetID string)

	// Vocabulaire de référence (catégories, types, unités)
	GetRefs() (*InterfaceRefs, error)
	AddRefCategory(cat string) error
	AddRefType(cat, typeName string) error
	AddRefUnit(cat, unit string) error

	// Modification d'assets et d'interfaces existants
	UpdateAsset(req UpdateRequest) (*Model3D, error)

	// Modules — assemblages nommés et versionnés
	CreateModule(req ModuleRequest) (*Model3D, error)
	GetModule(id string) (*Model3D, error)
	ListModules(channelID string) ([]*Model3D, error)
	AddAssemblyToModule(moduleID, connID string) error
	RemoveAssemblyFromModule(moduleID, connID string) error
	SubmitModule(moduleID, note string) (*Model3D, error)
	RemoveModule(id string) error

	// Interfaces exposées d'un module (interfaces non connectées en interne)
	GetModuleInterfaces(moduleID string) ([]*AssetInterface, error)

	// Composition — gestion des instances d'un module
	AddAssetToWorkspace(moduleID, assetID string) (*Model3D, error)
	RemoveAssetFromWorkspace(moduleID, instanceID string) (*Model3D, error)
	UpdateInstancePosition(moduleID, instanceID string, x, y float64) (*Model3D, error)

	// Licences — catalogue statique + règles de compatibilité
	ListLicenses() []*License
	GetLicense(id string) (*License, error)
	CheckLicenseCompatibility(parentLicenseID, proposedLicenseID string) *LicenseCheck
	CheckModuleLicenseCompatibility(componentLicenseIDs []string, proposedProductLicenseID string) *LicenseCheck

	// Liaison interface virtuelle → interface physique
	// popupValues porte les champs saisis par le client (CLI/API).
	ConnectVirtualToPhysical(virtualIfaceID, physicalIfaceID string, popupValues AssetInterface, fromInstanceID, toInstanceID string) (*Connection, error)
}
