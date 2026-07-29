// domain/model/entity.go
package model

import (
	"crypto/rand"
	"fmt"
	"time"
)

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

// Model3D est l'entité unifiée représentant un composant ou un module.
// Un composant contient une ressource (Hash non vide).
// Un module contient des sous-composants/modules (WorkspaceInstances non vide).
// Les deux partagent les mêmes paramètres — la distinction se fait par le contenu.
type Model3D struct {
	ID          string
	Name        string
	Description string
	Category    Category
	ParentID    string // ID du composant parent (lignée)
	BlockID     string // référence bloc blockchain
	Hash        string // hash du fichier ressource (composant simple)
	ChannelID   string
	OwnerID     string
	LicenseID   string `json:"license_id,omitempty"`
	Tags        []string
	Links       []string // URLs boutique / sources web
	Versions    []Version
	CreatedAt   time.Time

	// Status (draft/submitted) s'applique à tout asset, composant ou module
	// (cycle brouillon → soumission généralisé, voir Service.Submit/SubmitModule).
	Status ModuleStatus `json:"status,omitempty"`
	// Interfaces embarque l'état courant des AssetInterface de l'asset au moment
	// de sa soumission (Service.Submit/SubmitModule) — source de vérité une fois
	// sur la blockchain ; avant soumission, les interfaces vivent uniquement dans
	// InterfaceStore (brouillon local).
	Interfaces []AssetInterface `json:"interfaces,omitempty"`

	// Champs module — peuplés quand le composant est un assemblage
	Assemblies         []string            `json:"assemblies"` // non-nil uniquement pour un module (CreateModule) — voir IsModule()
	WorkspaceInstances []WorkspaceInstance `json:"workspace_instances,omitempty"`
	ModuleVersions     []ModuleVersion     `json:"module_versions,omitempty"`
}

// IsModule retourne true si ce Model3D est un module (créé via CreateModule).
// Le discriminant est la présence du slice Assemblies (non-nil dès la création
// d'un module, jamais initialisé pour un composant) — pas Status, qui s'applique
// désormais aussi aux composants en brouillon (draft) et ne distingue donc plus rien.
func (m *Model3D) IsModule() bool {
	return m.Assemblies != nil
}

type Version struct {
	Number    int
	Hash      string
	CreatedAt time.Time
}

// Connection représente un lien d'assemblage entre deux assets.
// Si FromIfaceID/ToIfaceID sont renseignés, le lien est un assemblage précis d'interfaces physiques.
// Sans ces champs, c'est un lien simple (compatibilité ascendante).
type Connection struct {
	ID              string
	From            string // ID asset source
	To              string // ID asset cible
	Label           string // ex: "composant", "assemblage"
	FromIfaceID     string // interface physique source (optionnel)
	ToIfaceID       string // interface physique cible (optionnel)
	FromInstanceID  string // ID instance workspace source (optionnel)
	ToInstanceID    string // ID instance workspace cible (optionnel)
	FastenerAssetID string // ID de l'asset d'accroche (vis, câble, clip…) — vide si connexion directe
	Incompatible    bool   `json:"incompatible,omitempty"` // true si les interfaces ne sont plus compatibles
}

// UpdateRequest regroupe les champs modifiables d'un asset existant.
// Seuls les champs non vides sont pris en compte (patch partiel).
type UpdateRequest struct {
	ID          string
	Name        string
	Description string
	LicenseID   string
	Tags        []string
	Links       []string
}

// AddRequest regroupe tous les paramètres de création d'un asset.
type AddRequest struct {
	FilePath    string
	Name        string
	Description string
	Category    Category
	ChannelID   string
	OwnerID     string
	ParentID    string
	LicenseID   string // ID dans le catalogue de licences (optionnel)
	Tags        []string
	Links       []string // URLs boutique / sources web
}

// ── Interfaces physiques ─────────────────────────────────────────────────────

// IfaceDirection indique le sens de l'interface sur un asset.
type IfaceDirection string

const (
	IfaceIn   IfaceDirection = "in"
	IfaceOut  IfaceDirection = "out"
	IfaceBidi IfaceDirection = "bidir"
)

// AssetInterface décrit un point de connexion physique d'un asset.
// Exemples : alimentation 5 V (ELEC/DIN/out), vis M3 (MECA/Vis M3/in), sortie eau (HYD/Push-fit 6mm/out).
type AssetInterface struct {
	ID        string         `json:"id"`
	AssetID   string         `json:"asset_id"`
	Name      string         `json:"name,omitempty"`    // label optionnel
	Category  string         `json:"category"`          // "ELEC", "MECA", "HYD" ou custom
	Type      string         `json:"type"`              // ex: "DIN", "Vis M3"
	Direction IfaceDirection `json:"direction"`         // "in", "out", "bidir"
	ValueMin  float64        `json:"value_min"`         // valeur unique ou borne basse
	ValueMax  float64        `json:"value_max"`         // borne haute (si IsRange)
	IsRange   bool           `json:"is_range"`          // true si plage de valeurs
	Unit      string         `json:"unit,omitempty"`    // ex: "V", "mm", "bar"
	Virtual   bool           `json:"virtual,omitempty"` // true = point de connexion non encore matérialisé
}

// InterfaceRefs contient le vocabulaire extensible des catégories, types et unités.
// Persisté dans le store, modifiable par l'utilisateur via l'API.
type InterfaceRefs struct {
	Categories []string            `json:"categories"`
	Types      map[string][]string `json:"types"` // catégorie → liste de types
	Units      map[string][]string `json:"units"` // catégorie → liste d'unités
}

// DefaultInterfaceRefs retourne le vocabulaire initial.
func DefaultInterfaceRefs() *InterfaceRefs {
	return &InterfaceRefs{
		Categories: []string{"ELEC", "MECA", "HYD"},
		Types: map[string][]string{
			"ELEC": {"DIN", "USB-C", "USB-A", "Jack 3.5", "XLR", "RJ45", "Header 2.54mm", "MIPI DSI", "SPI", "I²C", "UART", "CAN"},
			"MECA": {"Vis M2", "Vis M3", "Vis M4", "Vis M5", "Vis M6", "Vis M8", "Rainure T", "Dovetail", "Press-fit", "Clip snap", "Charnière"},
			"HYD":  {"BSP 1/4\"", "BSP 1/2\"", "NPT 1/4\"", "Push-fit 6mm", "Push-fit 8mm", "Push-fit 10mm", "Raccord rapide"},
		},
		Units: map[string][]string{
			"ELEC": {"V", "mV", "A", "mA", "W", "Hz", "kHz", "MHz", "Ω", "kΩ", "bit/s", "Mbit/s"},
			"MECA": {"mm", "cm", "m", "N", "Nm", "kg", "g", "°", "mm/s", "m/s", "rpm", "Hz"},
			"HYD":  {"bar", "mbar", "MPa", "L/min", "mL/s", "L/h", "°C", "m³/h"},
		},
	}
}

// ── Assemblage (module) ───────────────────────────────────────────────────────

// ModuleStatus décrit l'état du cycle de vie d'un composant utilisé comme module.
type ModuleStatus string

const (
	ModuleDraft     ModuleStatus = "draft"     // en cours de composition
	ModuleSubmitted ModuleStatus = "submitted" // soumis à la blockchain
)

// ModuleVersion est un snapshot immuable d'un module, ancré sur la blockchain.
type ModuleVersion struct {
	Number     int       `json:"number"`
	Assemblies []string  `json:"assemblies"` // snapshot des IDs de connexions
	Hash       string    `json:"hash"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	BlockID    string    `json:"block_id,omitempty"`
}

// WorkspaceInstance représente un slot de référence d'un composant dans un module.
// Le même composant peut apparaître plusieurs fois via des instances indépendantes.
type WorkspaceInstance struct {
	ID      string  `json:"id"`
	AssetID string  `json:"asset_id"` // ID du composant ou module référencé
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

// ModuleRequest regroupe les paramètres de création d'un module (composant-assemblage).
type ModuleRequest struct {
	Name        string
	Description string
	OwnerID     string
	ChannelID   string
	LicenseID   string
}

// ── interne ───────────────────────────────────────────────────────────────────

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
