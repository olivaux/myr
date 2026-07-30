// chaincode/model/entity.go
package model

import "time"

// Category définit le type d'évolution d'un asset selon la nomenclature MYR.
type Category string

const (
	CategoryBase         Category = "base"
	CategoryAmelioration Category = "amelioration"
	CategoryVariation    Category = "variation"
	CategoryAdaptation   Category = "adaptation"
	CategoryDerivation   Category = "derivation"
	CategoryExtension    Category = "extension"
	CategoryRegression   Category = "regression"
)

// ModuleStatus décrit l'état du cycle de vie d'un composant utilisé comme module.
type ModuleStatus string

const (
	ModuleDraft     ModuleStatus = "draft"
	ModuleSubmitted ModuleStatus = "submitted"
)

// IfaceDirection indique le sens de l'interface sur un asset.
type IfaceDirection string

const (
	IfaceIn   IfaceDirection = "in"
	IfaceOut  IfaceDirection = "out"
	IfaceBidi IfaceDirection = "bidir"
)

// Model3D miroir exact de domain/model/entity.go (myr-core) : mêmes champs,
// mêmes tags JSON (ou absence de tag) dans le même ordre. C'est le contrat de
// sérialisation avec adapters/out/fabric/blockchain.go (StoreModelRecord marshal
// le Model3D domaine tel quel, GetModelRecord/ListModelRecords désérialisent la
// réponse directement dedans) : tout écart de tag ici reproduit la perte de
// données silencieuse décrite dans specs/3-Conception/Chaincode.md §2.
type Model3D struct {
	ID          string
	Name        string
	Description string
	Category    Category
	ParentID    string
	BlockID     string
	Hash        string
	ChannelID   string
	OwnerID     string
	LicenseID   string `json:"license_id,omitempty"`
	Tags        []string
	Links       []string
	Versions    []Version
	CreatedAt   time.Time

	Status     ModuleStatus     `json:"status,omitempty"`
	Interfaces []AssetInterface `json:"interfaces,omitempty"`

	Assemblies         []string            `json:"assemblies"`
	WorkspaceInstances []WorkspaceInstance `json:"workspace_instances,omitempty"`
	ModuleVersions     []ModuleVersion     `json:"module_versions,omitempty"`
}

type Version struct {
	Number    int
	Hash      string
	CreatedAt time.Time
}

// AssetInterface décrit un point de connexion physique ou virtuel d'un asset.
type AssetInterface struct {
	ID        string         `json:"id"`
	AssetID   string         `json:"asset_id"`
	Name      string         `json:"name,omitempty"`
	Category  string         `json:"category"`
	Type      string         `json:"type"`
	Direction IfaceDirection `json:"direction"`
	ValueMin  float64        `json:"value_min"`
	ValueMax  float64        `json:"value_max"`
	IsRange   bool           `json:"is_range"`
	Unit      string         `json:"unit,omitempty"`
	Virtual   bool           `json:"virtual,omitempty"`
}

// ModuleVersion est un snapshot immuable d'un module, ancré sur la blockchain.
type ModuleVersion struct {
	Number     int       `json:"number"`
	Assemblies []string  `json:"assemblies"`
	Hash       string    `json:"hash"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	BlockID    string    `json:"block_id,omitempty"`
}

// WorkspaceInstance représente un slot de référence d'un composant dans un module.
type WorkspaceInstance struct {
	ID      string  `json:"id"`
	AssetID string  `json:"asset_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}
