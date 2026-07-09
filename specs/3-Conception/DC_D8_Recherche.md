# DC — D8 : Recherche

> Phase 3 — Arrington | Use cases : UCREC01–05 | Domaine : `domain/model`

---

## 1. Objectif

Ce document modélise la couche de conception manquante pour D8 — les diagrammes de séquence détaillés existent déjà au niveau analyse (`specs/2-Analyse/UCREC-Recherche/UCREC02.md` à `UCREC05.md`, qui vont jusqu'au pseudo-code de handler) : ce document ne les duplique pas, il en extrait la vue domaine (ports, entités, algorithmes transverses) et consigne les décisions/écarts au même format que `DC_D2_Administration.md`/`DC_D7_Payment.md`.

Aucune de ces fonctionnalités n'a de méthode `ModelService` ni d'endpoint REST dédié aujourd'hui — `List(channelID)` et `Get(id)` couvrent UCREC01 (déjà exposé, voir `DC_CLI_Model.md` §5), le reste (UCREC02–05) est absent.

---

## 2. UCREC02 — Composants compatibles

**Contrat de service proposé** (nouvelle méthode `ModelService`, réutilise `ifacesCompatible()` — voir `Architecture_Composition.md` §5) :

```go
// domain/model — port in, méthode à ajouter
FindCompatibleAssets(sourceID, channelID string, categoryFilter []string) ([]*CompatibleAsset, error)

type CompatibleAsset struct {
    Asset               *Model3D
    CompatibleInterfaces []InterfacePair // {SourceIfaceID, CandidateIfaceID}
}
```

Algorithme : cf. `specs/2-Analyse/UCREC-Recherche/UCREC02.md` § Diagramme de séquence (déjà détaillé au niveau analyse — O(n × m), n = composants du canal, m = interfaces par composant).

**Écart requis en amont :** dépend de l'écart E2 (`Tag` manquant sur `AssetInterface`, `Architecture_Composition.md` §5) pour appliquer les 5 critères complets de RM11.

---

## 3. UCREC03 — Arbre de versions (généalogie `ParentID`)

`GetChildren(parentID)` est déjà implémenté (`service.go:~251`). Il manque la remontée ascendante et l'assemblage en arbre.

**Contrat de service proposé :**

```go
// domain/model — port in, méthode à ajouter
GetLineage(assetID, channelID string) (*LineageTree, error)

type LineageTree struct {
    Root     *Model3D
    Children []*LineageNode
}
type LineageNode struct {
    Asset          *Model3D
    DerivationType Category // RM02 : amelioration, variation, adaptation, derivation, extension, regression, decoupage
    Children       []*LineageNode
}
```

Implémentation : remontée via `ParentID` jusqu'à `ParentID == ""` (racine), puis descente récursive via `GetChildren()` existant à chaque nœud. Pas de nouvel algorithme — composition de méthodes déjà présentes dans le service.

> Le rendu graphique de l'arbre relève du dépôt GUI externe (`specs/2-Analyse/UCREC-Recherche/UCREC03.md`) — seul le contrat de données ci-dessus fait partie de `myr`.

---

## 4. UCREC04 — Modules utilisant un composant

**Contrat de service proposé :**

```go
// domain/model — port in, méthode à ajouter
FindModulesUsingComponent(componentID, channelID string, requesterID string, includeOwnDrafts bool) ([]*Model3D, error)
```

Algorithme (cf. `specs/2-Analyse/UCREC-Recherche/UCREC04.md`) : `ListModules(channelID)` puis filtre sur `WorkspaceInstances[].AssetID == componentID`, avec application de la visibilité RM16 (un module `draft` n'est visible que pour son propriétaire — comparaison `OwnerID == requesterID`, actuellement non implémentée nulle part dans le code, écart transverse déjà noté pour UCA06/`Securite.md`).

**Question ouverte pour le PO (non tranchée ici, cf. `specs/2-Analyse/UCREC-Recherche/UCREC04.md` § Notes d'implémentation) :** la recherche doit-elle descendre dans les sous-modules (un module contenant un sous-module qui contient lui-même le composant cible) ? Le contrat ci-dessus ne couvre que la profondeur directe — une recherche récursive nécessiterait un paramètre supplémentaire (`recursive bool`) une fois la décision prise.

---

## 5. UCREC05 — Export BOM (Bill of Materials)

**Contrat de service proposé :**

```go
// domain/model — port in, méthode à ajouter
ResolveBOM(moduleID, channelID string) (*BOM, error)

type BOM struct {
    ModuleID string
    Lines    []BOMLine
    Warnings []string // ex: "fichier IPFS indisponible pour <assetID>"
}
type BOMLine struct {
    AssetID   string
    Name      string
    Category  string
    Quantity  int    // agrégation des instances multiples — RM15
    Hash      string // Model3D.Versions[last].Hash
    IPFSURL   string // vide si fichier indisponible
}
```

Résolution récursive via `WorkspaceInstances` (composant feuille → ligne BOM ; sous-module → récursion), agrégation par `AssetID` (RM15), respect ENF03 (≤ 5s / 100 composants — cache par `assetID` pour éviter les appels Fabric redondants, cf. `specs/2-Analyse/UCREC-Recherche/UCREC05.md` § Notes d'implémentation). Le formatage de sortie (CSV/XML/PDF) est une préoccupation d'adapter `in/rest/` (content negotiation), pas du domaine — `ResolveBOM` retourne une structure de données, jamais un flux de fichier déjà formaté.

**Questions ouvertes pour le PO (non tranchées ici, cf. `specs/2-Analyse/UCREC-Recherche/UCREC05.md`) :**
- Inclusion des sous-modules comme lignes agrégées, ou uniquement les composants feuilles ?
- Inclusion des prix (UCPI04/05, `DC_D7_Payment.md`) dans la BOM ?
- Priorité du format PDF pour la v1 (CSV recommandé en premier par l'analyse) ?

---

## 6. Décisions de conception

| ID | Décision | Raison |
|----|---------|--------|
| DC-D8-01 | Filtrage et agrégation côté service domaine (`ModelService`), jamais dans le chaincode Fabric | `WorkspaceInstances` est un champ complexe difficile à requêter en chaincode v1 (cf. UCREC02/04) — cohérent avec le principe « le domaine est la seule source de vérité fonctionnelle ». Un index secondaire côté chaincode reste une optimisation future si la volumétrie l'exige. |
| DC-D8-02 | `ResolveBOM` retourne une structure de données, le formatage (CSV/XML/PDF) est un problème d'adapter | Garantit la parité CLI/REST : un même contrat de service peut être rendu en JSON (REST), texte tabwriter (CLI), ou fichier détaché (export), sans dupliquer la logique de résolution. |
| DC-D8-03 | `GetLineage` compose `GetChildren` existant plutôt que réimplémenter la traversée | Centraliser les fonctions — évite un second algorithme de parcours d'arbre `ParentID`. |

---

## 7. Écarts code → specs

| ID | Écart | Fichier à corriger | Impact |
|----|-------|-------------------|--------|
| E-D8-01 | Aucune méthode `FindCompatibleAssets` dans `ModelService` | `domain/model/port_in.go`, `service.go` | UCREC02 non implémentable — `ifacesCompatible()` existe mais n'est pas exposée en méthode publique |
| E-D8-02 | Aucune méthode `GetLineage` — seule la descente (`GetChildren`) existe | `domain/model/port_in.go`, `service.go` | UCREC03 partiellement bloqué (remontée manquante) |
| E-D8-03 | Aucune méthode `FindModulesUsingComponent` | `domain/model/port_in.go`, `service.go` | UCREC04 non implémentable |
| E-D8-04 | Aucune méthode `ResolveBOM` | `domain/model/port_in.go`, `service.go` | UCREC05 non implémentable |
| E-D8-05 | Aucune comparaison `OwnerID == requesterID` pour filtrer les `draft` d'autrui | `adapters/in/rest/handlers.go`, `adapters/in/cli/` | UCREC04 (visibilité RM16) — écart de sécurité transverse, déjà signalé pour UCA06/`Securite.md` |
| — (rappel) | Dépendance sur écart E2 (`Tag` absent) | `domain/model/entity.go` | UCREC02 : compatibilité limitée à 4/5 critères tant que E2 n'est pas corrigé |

---

## 8. CLI et REST

Aucune commande CLI ni route REST n'existe pour ces quatre méthodes — une fois les méthodes `ModelService` ajoutées (§6/§7), les commandes cibles suivent la même convention que `DC_CLI_Model.md` :

| Méthode `ModelService` | Commande CLI cible | Route REST cible | Use case |
|---|---|---|---|
| `FindCompatibleAssets` | `myr model compatible <id>` | `GET /api/components/:id/compatible` | UCREC02 |
| `GetLineage` | `myr model lineage <id>` | `GET /api/components/:id/versions` | UCREC03 |
| `FindModulesUsingComponent` | `myr module find-users <componentID>` | `GET /api/modules?uses_component=:id` | UCREC04 |
| `ResolveBOM` | `myr module bom-export <id> --format csv` | `GET /api/modules/:id/bom?format=csv` | UCREC05 |

Ce tableau complète — sans le remplacer — `DC_CLI_Model.md` §6 point 2/3, qui signalait déjà l'absence de méthode de filtre serveur pour la recherche.
