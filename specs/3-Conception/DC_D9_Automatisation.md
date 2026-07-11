# DC — D9 : Automatisation

> Phase 3 — Arrington | Use cases : UCAUT01–04 | Domaines : `payment`, `model`, `role`

---

## 1. Objectif

Ce document conçoit D9, qui n'avait aucune trace dans `specs/3-Conception/` jusqu'ici. Il distingue ce qui est déjà couvert ailleurs (UCAUT01, largement conçu par `DC_D7_Payment.md`/`Chaincode.md`) de ce qui reste à concevoir (UCAUT02–04).

---

## 2. UCAUT01 — Déclenchement des commissions à la livraison

**Ne pas dupliquer :** ce use case est déjà conçu en détail dans `DC_D7_Payment.md` §4 (algorithme de distribution, RM23/RM24) et `Chaincode.md` (fonction `DistributeCommissions`). Renvoi uniquement.

Point spécifique à D9 non couvert ailleurs : le rôle `manufacturer` (déclencheur `POST /api/orders/:id/deliver`) n'existe pas par défaut dans le RBAC dynamique (`domain/role`) — à créer via `myr role create manufacturer --permission order.deliver` (cf. `specs/2-Analyse/UCAUT-Automatisation/UCAUT01.md` § Notes d'implémentation). C'est une opération de configuration RBAC standard (`domain/role`, déjà conçu), pas un nouveau mécanisme.

---

## 3. UCAUT02 — Commande via boutique partenaire (API tierce)

**Contexte :** une boutique partenaire (client API tiers, rôle `developer`, cf. D13/`DC_CLI_Admin.md`) commande un asset au nom d'un consommateur, via l'API REST publique — pas d'interface graphique Myr impliquée (la boutique est un client externe comme un autre).

**Ce use case ne nécessite aucune nouvelle entité de domaine** : il réutilise `Order`/`OrderItem` déjà conçues dans `DC_D7_Payment.md` §3 (UCPI01) et `AssetPrice` (§3, DC-D7-04) pour le prix en temps réel. La différence avec UCPI01 est uniquement l'**identité de l'appelant** (token API `developer` agissant pour le compte d'un consommateur, plutôt qu'une session consommateur directe) — mécanisme déjà couvert par le modèle de session REST existant (`DC_D1_Auth_Identity.md`), pas un nouveau système d'authentification.

**Contrat REST nécessaire (déjà dans le périmètre `DC_D7_Payment.md`, référencé ici) :**
- `GET /api/assets/:id/price` — lecture de `AssetPrice`
- `POST /api/orders` — création d'`Order`, avec un champ `OnBehalfOf` (consommateur final) distinct de l'identité de l'appelant (boutique)

**Décision de conception :**

| ID | Décision | Raison |
|----|---------|--------|
| DC-D9-01 | `Order.ConsumerID` distinct du pseudo authentifié de la requête | Une boutique partenaire (rôle `developer`) agit *pour* un consommateur, jamais *en tant que* lui — la traçabilité de commission (RM24) doit pointer vers le vrai consommateur, pas vers la boutique. |

**Question ouverte pour le PO (non tranchée, cf. `specs/2-Analyse/UCAUT-Automatisation/UCAUT02.md`) :** le consommateur final doit-il être un compte Myr existant (`MyrIdentity`), ou la boutique peut-elle transmettre une identité externe non enregistrée (email/nom) ? Ce point conditionne si `Order.ConsumerID` est une FK stricte vers `MyrIdentity` ou un champ libre.

**Écart requis en amont :** ce use case est bloqué tant que `Order`/`OrderItem`/`AssetPrice` ne sont pas implémentées (voir `DC_D7_Payment.md` §7, écarts déjà trackés) — aucun nouvel écart D9 spécifique.

---

## 4. UCAUT03 — Import depuis un plugin CAO

**Contexte :** un plugin tiers (Fusion 360, FreeCAD, SolidWorks…) détecte les interfaces et métadonnées d'un modèle 3D côté logiciel CAO, puis soumet vers Myr. Le plugin est un **client externe** consommant l'API REST documentée — son ergonomie et sa logique de détection ne font pas partie de ce dépôt.

**Ce que `myr` doit concevoir :** uniquement le contrat d'entrée côté `POST /api/components` — il doit accepter un tableau `Interfaces` pré-rempli à la création, pour que le plugin puisse soumettre en un seul appel les interfaces détectées automatiquement, sans passer par des appels `AddInterface` séparés après coup.

```go
// domain/model — extension d'AddRequest existante
type AddRequest struct {
    // ... champs existants ...
    Interfaces []AssetInterface // optionnel — pré-rempli si le client (plugin CAO) les a déjà détectées
}
```

La vérification anti-plagiat (RM01, déclenchée par UC5 du diagramme d'acteurs UCAUT03) suit exactement le contrat déjà conçu en `Architecture_Composition.md` §6 — aucune variante pour ce canal d'entrée.

**Question ouverte pour le PO (non tranchée, cf. `specs/2-Analyse/UCAUT-Automatisation/UCAUT03.md`) :** quels logiciels CAO sont prioritaires pour un plugin de référence (FreeCAD, Fusion 360, SolidWorks…) ? Sans objet pour `myr` lui-même (le plugin vit hors de ce dépôt) mais conditionne l'ordre dans lequel le contrat `Interfaces` ci-dessus sera exercé en pratique.

**Décision de conception :**

| ID | Décision | Raison |
|----|---------|--------|
| DC-D9-02 | Le plugin CAO est un client API comme un autre — aucun endpoint ni format spécifique « plugin » | Un logiciel tiers qui n'appelle `myr` que via l'API publique documentée reste un programme séparé. Créer un endpoint dédié « plugin » dupliquerait `POST /api/components`. |

---

## 5. UCAUT04 — Gestion SCM (versionnement) d'un modèle 3D

**Contexte :** versionnement itératif d'un composant en cours d'édition (diffs, historique, comparaison), distinct des soumissions blockchain (`ModuleVersion`, immuable) — similaire à un contrôle de version pour des modèles 3D, format d'export XML.

**Entité existante réutilisée :** `Model3D.Versions []Version` (`Number`, `Hash`, `CreatedAt` — `Modele_Domaine.md` §2) porte déjà l'historique de fichier. UCAUT04 a besoin d'une couche supplémentaire : le **diff** entre deux versions, pas seulement leur liste.

**Contrat de service proposé (nouveau) :**

```go
// domain/model — port in, méthode à ajouter
SaveVersion(assetID, message string, fileData []byte) (*Version, error)
DiffVersions(assetID string, fromVersion, toVersion int) (*VersionDiff, error)
ExportVersionDiff(assetID string, fromVersion, toVersion int) ([]byte, error) // XML

type VersionDiff struct {
    AssetID       string
    FromVersion   int
    ToVersion     int
    GeometryDelta string // format non spécifié — dépend de l'outil de diff CAO retenu
    InterfaceDiff []InterfaceChange
}
```

**Lien avec RM01 (anti-plagiat) :** l'analyse (`specs/2-Analyse/UCAUT-Automatisation/UCAUT04.md`) note que les deltas XML produits ici pourraient alimenter l'algorithme de similarité structurelle SCM (`Architecture_Composition.md` §6). Ce document ne tranche pas ce couplage — il reste une piste d'implémentation future, pas une dépendance bloquante : `DiffVersions`/`PlagiarismChecker.CompareStructural` restent deux contrats indépendants tant que l'algorithme SCM n'est pas choisi.

**Question ouverte pour le PO (non tranchée, cf. analyse) :** l'algorithme de diff géométrique (`GeometryDelta`) est-il une librairie Go existante, un outil CAO externe appelé en sous-processus, ou hors périmètre v1 (seul le diff des métadonnées/interfaces serait alors implémenté) ? #remarque une piste possible (voir `roadmap_dev.md` § Compléments — Post-V1) : stocker nativement un format paramétrique (arbre de construction — schémas, extrusions, opérations successives) plutôt que le fichier 3D final, ce qui rendrait `GeometryDelta` diffable/fusionnable nativement au lieu de dépendre d'un outil de diff CAO externe — non tranché, décision PO.

---

## 6. Écarts code → specs

| ID | Écart | Fichier à corriger | Impact |
|----|-------|-------------------|--------|
| E-D9-01 | Rôle `manufacturer` absent des rôles RBAC par défaut | `domain/role/` (config des rôles intégrés) ou procédure admin (`myr role create`) | UCAUT01 — aucun manufacturer ne peut confirmer de livraison tant que le rôle n'est pas créé |
| E-D9-02 | `AddRequest` n'accepte pas de tableau `Interfaces` pré-rempli | `domain/model/entity.go`, `service.go` (`AddFull`) | UCAUT03 — le plugin CAO devrait sinon enchaîner `AddFull` + N appels `AddInterface`, perdant l'atomicité de soumission (RM07) |
| E-D9-03 | Aucune méthode `SaveVersion`/`DiffVersions`/`ExportVersionDiff` | `domain/model/port_in.go`, `service.go` | UCAUT04 non implémentable — seul `Versions []Version` (liste plate) existe |
| — (rappel) | `Order`/`OrderItem`/`AssetPrice` absentes | `domain/payment/` | UCAUT01 (rappel `DC_D7_Payment.md` §7) et UCAUT02 en dépendent directement |

---

## 7. CLI et REST cibles

| Méthode `ModelService`/`PaymentService` | Commande CLI cible | Route REST cible | Use case |
|---|---|---|---|
| `AddFull` (avec `Interfaces` pré-remplies) | `myr model add --interfaces-file <json>` | `POST /api/components` (`Interfaces` dans le body) | UCAUT03 |
| `SaveVersion` | `myr model version save <id> --message <msg>` | `POST /api/components/:id/versions` | UCAUT04 |
| `DiffVersions` / `ExportVersionDiff` | `myr model version diff <id> --from <n> --to <n>` | `GET /api/components/:id/versions/diff?from=&to=` | UCAUT04 |
| `CreateOrder` (déjà cible `DC_D7_Payment.md`) | — (boutique tierce, pas de CLI dédié) | `POST /api/orders` | UCAUT02 |
