// domain/model/license.go — entité Licence, catalogue statique et règles de compatibilité.
// Le catalogue est embarqué dans le domaine : il est disponible identiquement en mode JSON
// local et en chaincode Hyperledger Fabric (aucune dépendance externe).
package model

import "fmt"

// ── Entités ───────────────────────────────────────────────────────────────────

// License décrit une licence de propriété intellectuelle.
type License struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SPDX        string   `json:"spdx"`        // identifiant SPDX normalisé (ex: "CC-BY-4.0")
	Description string   `json:"description"` // résumé humain
	Permissions []string `json:"permissions"` // ex: "commercial-use", "modifications"
	Conditions  []string `json:"conditions"`  // ex: "include-copyright", "same-license"
	Limitations []string `json:"limitations"` // ex: "liability", "trademark-use"
	// CompatibleWith liste les IDs de licences autorisées pour les œuvres dérivées.
	// Vide = aucune dérivation permise.
	CompatibleWith []string `json:"compatible_with"`
}

// LicenseCheck est le résultat d'un contrôle de compatibilité entre deux licences.
type LicenseCheck struct {
	Compatible bool     `json:"compatible"`
	Reason     string   `json:"reason,omitempty"`  // explication humaine
	Allowed    []string `json:"allowed,omitempty"` // IDs compatibles avec la licence source
}

// ── Catalogue ─────────────────────────────────────────────────────────────────
// Vocabulaire des permissions / conditions / limitations :
//   Permissions : commercial-use | modifications | distribution | private-use | patent-use
//   Conditions  : include-copyright | document-changes | same-license | same-license-w |
//                 disclose-source | network-use-disclose
//   Limitations : liability | warranty | trademark-use | patent-use

var licenseCatalog = []*License{
	{
		ID: "proprietary", Name: "Tous droits réservés", SPDX: "LicenseRef-Proprietary",
		Description:    "Aucune utilisation, reproduction ou modification n'est autorisée sans accord explicite du titulaire.",
		Permissions:    []string{"private-use"},
		Conditions:     []string{},
		Limitations:    []string{"commercial-use", "modifications", "distribution", "patent-use", "liability", "warranty"},
		CompatibleWith: []string{"proprietary"},
	},
	{
		ID: "cc0", Name: "Creative Commons Zéro 1.0", SPDX: "CC0-1.0",
		Description: "Renonciation totale aux droits — l'œuvre entre dans le domaine public. Aucune restriction.",
		Permissions: []string{"commercial-use", "modifications", "distribution", "private-use"},
		Conditions:  []string{},
		Limitations: []string{"liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{
			"proprietary", "cc0", "cc-by-4", "cc-by-sa-4", "cc-by-nc-4", "cc-by-nc-sa-4",
			"cc-by-nd-4", "cc-by-nc-nd-4",
			"cern-ohl-p-2", "cern-ohl-w-2", "cern-ohl-s-2", "tapr-ohl",
			"mit", "apache-2", "gpl-3", "lgpl-3",
		},
	},
	{
		ID: "cc-by-4", Name: "Creative Commons Attribution 4.0", SPDX: "CC-BY-4.0",
		Description: "Utilisation libre (y compris commerciale) avec obligation de mentionner l'auteur original.",
		Permissions: []string{"commercial-use", "modifications", "distribution", "private-use"},
		Conditions:  []string{"include-copyright", "document-changes"},
		Limitations: []string{"liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{
			"cc0", "cc-by-4", "cc-by-sa-4", "cc-by-nc-4", "cc-by-nc-sa-4",
			"cc-by-nd-4", "cc-by-nc-nd-4",
			"cern-ohl-p-2", "cern-ohl-w-2", "cern-ohl-s-2", "tapr-ohl",
			"mit", "apache-2", "gpl-3", "lgpl-3",
		},
	},
	{
		ID: "cc-by-sa-4", Name: "Creative Commons Attribution-ShareAlike 4.0", SPDX: "CC-BY-SA-4.0",
		Description:    "Attribution obligatoire + les œuvres dérivées doivent être publiées sous la même licence (copyleft).",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use"},
		Conditions:     []string{"include-copyright", "document-changes", "same-license"},
		Limitations:    []string{"liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{"cc-by-sa-4", "gpl-3"}, // CC déclare CC-BY-SA 4.0 compatible GPL v3
	},
	{
		ID: "cc-by-nc-4", Name: "Creative Commons Attribution-NonCommercial 4.0", SPDX: "CC-BY-NC-4.0",
		Description:    "Attribution obligatoire. Usage commercial interdit. Les dérivés doivent être non-commerciaux.",
		Permissions:    []string{"modifications", "distribution", "private-use"},
		Conditions:     []string{"include-copyright", "document-changes"},
		Limitations:    []string{"commercial-use", "liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{"cc-by-nc-4", "cc-by-nc-sa-4", "cc-by-nc-nd-4"},
	},
	{
		ID: "cc-by-nc-sa-4", Name: "Creative Commons Attribution-NonCommercial-ShareAlike 4.0", SPDX: "CC-BY-NC-SA-4.0",
		Description:    "Non-commercial + attribution + partage à l'identique. Licence la plus restrictive hors NoDerivs.",
		Permissions:    []string{"modifications", "distribution", "private-use"},
		Conditions:     []string{"include-copyright", "document-changes", "same-license"},
		Limitations:    []string{"commercial-use", "liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{"cc-by-nc-sa-4"},
	},
	{
		ID: "cc-by-nd-4", Name: "Creative Commons Attribution-NoDerivatives 4.0", SPDX: "CC-BY-ND-4.0",
		Description:    "Attribution obligatoire. Aucune modification ni œuvre dérivée autorisée.",
		Permissions:    []string{"commercial-use", "distribution", "private-use"},
		Conditions:     []string{"include-copyright"},
		Limitations:    []string{"modifications", "liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{}, // aucune dérivation permise
	},
	{
		ID: "cc-by-nc-nd-4", Name: "Creative Commons Attribution-NonCommercial-NoDerivatives 4.0", SPDX: "CC-BY-NC-ND-4.0",
		Description:    "Attribution obligatoire. Ni usage commercial, ni modification autorisés.",
		Permissions:    []string{"distribution", "private-use"},
		Conditions:     []string{"include-copyright"},
		Limitations:    []string{"commercial-use", "modifications", "liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{}, // aucune dérivation permise
	},
	{
		ID: "cern-ohl-p-2", Name: "CERN Open Hardware Licence Permissive v2", SPDX: "CERN-OHL-P-2.0",
		Description: "Licence matérielle permissive. Utilisation libre y compris commerciale, sans obligation de réciprocité.",
		Permissions: []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:  []string{"include-copyright", "document-changes"},
		Limitations: []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{
			"proprietary", "cc0", "cc-by-4", "cc-by-sa-4", "cc-by-nc-4", "cc-by-nc-sa-4",
			"cc-by-nd-4", "cc-by-nc-nd-4",
			"cern-ohl-p-2", "cern-ohl-w-2", "cern-ohl-s-2", "tapr-ohl",
			"mit", "apache-2", "gpl-3", "lgpl-3",
		},
	},
	{
		ID: "cern-ohl-w-2", Name: "CERN Open Hardware Licence Weakly Reciprocal v2", SPDX: "CERN-OHL-W-2.0",
		Description:    "Licence matérielle copyleft faible. Les modifications des sources couvertes doivent rester CERN-OHL-W.",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:     []string{"include-copyright", "document-changes", "disclose-source", "same-license-w"},
		Limitations:    []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{"cern-ohl-w-2", "cern-ohl-s-2"},
	},
	{
		ID: "cern-ohl-s-2", Name: "CERN Open Hardware Licence Strongly Reciprocal v2", SPDX: "CERN-OHL-S-2.0",
		Description:    "Licence matérielle copyleft fort. L'ensemble du design dérivé doit rester CERN-OHL-S.",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:     []string{"include-copyright", "document-changes", "disclose-source", "same-license"},
		Limitations:    []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{"cern-ohl-s-2"},
	},
	{
		ID: "tapr-ohl", Name: "TAPR Open Hardware License", SPDX: "TAPR-OHL",
		Description:    "Licence matérielle copyleft (précurseur CERN-OHL). Les œuvres dérivées doivent rester TAPR-OHL.",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use"},
		Conditions:     []string{"include-copyright", "document-changes", "disclose-source", "same-license"},
		Limitations:    []string{"liability", "warranty", "trademark-use", "patent-use"},
		CompatibleWith: []string{"tapr-ohl"},
	},
	{
		ID: "mit", Name: "MIT License", SPDX: "MIT",
		Description: "Licence logicielle ultra-permissive. Utilisation libre à condition de conserver la notice de copyright.",
		Permissions: []string{"commercial-use", "modifications", "distribution", "private-use"},
		Conditions:  []string{"include-copyright"},
		Limitations: []string{"liability", "warranty"},
		CompatibleWith: []string{
			"proprietary", "cc0", "cc-by-4", "cc-by-sa-4", "cc-by-nc-4", "cc-by-nc-sa-4",
			"cc-by-nd-4", "cc-by-nc-nd-4",
			"cern-ohl-p-2", "cern-ohl-w-2", "cern-ohl-s-2", "tapr-ohl",
			"mit", "apache-2", "gpl-3", "lgpl-3",
		},
	},
	{
		ID: "apache-2", Name: "Apache License 2.0", SPDX: "Apache-2.0",
		Description: "Licence logicielle permissive avec clause de brevet. Compatib. GPL v3 mais non GPL v2.",
		Permissions: []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:  []string{"include-copyright", "document-changes"},
		Limitations: []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{
			"proprietary", "cc0", "cc-by-4", "cc-by-sa-4", "cc-by-nc-4", "cc-by-nc-sa-4",
			"cc-by-nd-4", "cc-by-nc-nd-4",
			"cern-ohl-p-2", "cern-ohl-w-2", "cern-ohl-s-2", "tapr-ohl",
			"mit", "apache-2", "gpl-3", "lgpl-3",
		},
	},
	{
		ID: "gpl-3", Name: "GNU General Public License v3", SPDX: "GPL-3.0-only",
		Description:    "Licence logicielle copyleft fort. Tout logiciel dérivé distribué doit être GPL v3.",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:     []string{"include-copyright", "document-changes", "disclose-source", "same-license", "network-use-disclose"},
		Limitations:    []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{"gpl-3"},
	},
	{
		ID: "lgpl-3", Name: "GNU Lesser General Public License v3", SPDX: "LGPL-3.0-only",
		Description:    "Copyleft faible — les modifications de la bibliothèque restent LGPL, mais les applications liées peuvent être propriétaires.",
		Permissions:    []string{"commercial-use", "modifications", "distribution", "private-use", "patent-use"},
		Conditions:     []string{"include-copyright", "document-changes", "disclose-source", "same-license--library"},
		Limitations:    []string{"liability", "warranty", "trademark-use"},
		CompatibleWith: []string{"lgpl-3", "gpl-3"},
	},
}

// licenseIndex est construit au démarrage pour O(1) lookup.
var licenseIndex map[string]*License

func init() {
	licenseIndex = make(map[string]*License, len(licenseCatalog))
	for _, l := range licenseCatalog {
		licenseIndex[l.ID] = l
	}
}

// ── Fonctions du domaine ──────────────────────────────────────────────────────

// ListLicenses retourne le catalogue complet des licences.
func ListLicenses() []*License {
	out := make([]*License, len(licenseCatalog))
	copy(out, licenseCatalog)
	return out
}

// GetLicense retourne une licence par son ID.
func GetLicense(id string) (*License, error) {
	if l, ok := licenseIndex[id]; ok {
		return l, nil
	}
	return nil, fmt.Errorf("licence %q introuvable", id)
}

// CheckLicenseCompatibility vérifie si proposedID est autorisé pour une œuvre dérivée
// d'un asset dont la licence source est parentID.
// Si parentID est vide (pas de licence définie sur le parent), tout est permis.
func CheckLicenseCompatibility(parentID, proposedID string) *LicenseCheck {
	if parentID == "" {
		return &LicenseCheck{Compatible: true, Reason: "Le parent n'a pas de licence définie — toute licence est acceptable."}
	}
	parent, ok := licenseIndex[parentID]
	if !ok {
		return &LicenseCheck{Compatible: false, Reason: fmt.Sprintf("Licence source %q inconnue.", parentID)}
	}
	if proposedID == "" {
		return &LicenseCheck{
			Compatible: false,
			Reason:     fmt.Sprintf("Aucune licence proposée. La licence source « %s » impose des contraintes sur les dérivés.", parent.Name),
			Allowed:    parent.CompatibleWith,
		}
	}
	proposed, ok := licenseIndex[proposedID]
	if !ok {
		return &LicenseCheck{Compatible: false, Reason: fmt.Sprintf("Licence proposée %q inconnue.", proposedID), Allowed: parent.CompatibleWith}
	}
	for _, id := range parent.CompatibleWith {
		if id == proposedID {
			return &LicenseCheck{Compatible: true, Reason: fmt.Sprintf("« %s » est compatible avec la source « %s ».", proposed.Name, parent.Name)}
		}
	}
	return &LicenseCheck{
		Compatible: false,
		Reason: fmt.Sprintf(
			"« %s » n'est pas compatible avec la source « %s ». Les dérivés doivent utiliser : %v.",
			proposed.Name, parent.Name, parent.CompatibleWith,
		),
		Allowed: parent.CompatibleWith,
	}
}

// CheckProductLicenseCompatibility vérifie qu'une licence de produit est compatible
// avec toutes les licences des composants (intersection des CompatibleWith de chaque composant).
func CheckModuleLicenseCompatibility(componentLicenseIDs []string, proposedProductLicenseID string) *LicenseCheck {
	// Calculer l'intersection des licences autorisées pour tous les composants.
	allowed := map[string]int{}
	total := 0
	for _, cid := range componentLicenseIDs {
		if cid == "" {
			continue // composant sans licence = pas de contrainte
		}
		l, ok := licenseIndex[cid]
		if !ok {
			continue
		}
		total++
		for _, a := range l.CompatibleWith {
			allowed[a]++
		}
	}
	if total == 0 {
		return &LicenseCheck{Compatible: true, Reason: "Aucun composant n'a de licence définie — toute licence est acceptable."}
	}

	// IDs présents dans toutes les listes (count == total)
	intersection := make([]string, 0)
	for id, count := range allowed {
		if count == total {
			intersection = append(intersection, id)
		}
	}
	if proposedProductLicenseID == "" {
		return &LicenseCheck{Compatible: false, Reason: "Aucune licence sélectionnée pour le produit.", Allowed: intersection}
	}
	for _, id := range intersection {
		if id == proposedProductLicenseID {
			l, _ := licenseIndex[proposedProductLicenseID]
			return &LicenseCheck{Compatible: true, Reason: fmt.Sprintf("« %s » est compatible avec tous les composants.", l.Name)}
		}
	}
	proposed, _ := licenseIndex[proposedProductLicenseID]
	name := proposedProductLicenseID
	if proposed != nil {
		name = proposed.Name
	}
	return &LicenseCheck{
		Compatible: false,
		Reason:     fmt.Sprintf("« %s » n'est pas compatible avec l'ensemble des composants.", name),
		Allowed:    intersection,
	}
}
