---
categorie: Composant Ecriture
titre: "Ajout d'un composant Physique"
probabilite: 3
impact: 5
importance: 15
etat: analyse
---

# Ajout d'un composant Physique

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Ajouter un composant physique" as UC1
    usecase "Vérifier le hash (anti-plagiat SHA-256)" as UC2
    usecase "Analyser similarité SCM" as UC3
    usecase "Vérifier compatibilité de licence" as UC4
    usecase "Enregistrer sur la blockchain" as UC5
    usecase "Stocker le fichier 3D (IPFS)" as UC6
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>
UC1 ..> UC4 : <<extend>> (si ParentID)
UC1 ..> UC5 : <<include>>
UC1 ..> UC6 : <<include>>

@enduml
```

## Contexte

L'ajout d'un composant physique est le use case fondamental du domaine D3. Il permet à un **Concepteur** d'enregistrer une création originale (pièce CAO, fichier 3D/STL/STEP/OBJ…) sur la blockchain Fabric, établissant ainsi son droit d'auteur de façon immuable.

L'entité produite est un `Model3D` avec `Hash` renseigné et `WorkspaceInstances` vide — c'est la définition d'un **Composant** dans la terminologie MYR.

La catégorie `base` est la seule catégorie qui ne requiert pas de `ParentID`. C'est aussi la seule catégorie soumise à la vérification anti-plagiat complète (SHA-256 + SCM > 50%). Les 7 autres catégories impliquent toutes un `ParentID` et un fichier parent existant sur le réseau.

**Écart code connu (E4) :** `service.AddFull()` calcule le SHA-256 du fichier soumis mais ne compare pas ce hash avec les assets déjà inscrits sur Fabric. La vérification complète (RM01) est une cible à implémenter.

## Pré-conditions

- Le Concepteur est authentifié avec le rôle `contributor` (JWT valide).
- Un canal Fabric est opérationnel et accessible.
- Le Concepteur dispose d'un fichier 3D/CAO valide (STL, STEP, OBJ, ou format natif).
- Pour une catégorie non-`base` : l'asset parent existe sur le canal et son ID est connu.

## Scénario

**Étape initiale :** Le Concepteur clique sur **New Asset** dans la MenuBar — une Asset UI vide s'ouvre.

### Flux nominal — Composant base nouveau

1. Le Concepteur importe son fichier 3D via l'interface (champ `file` ou `stl` — multipart/form-data).
2. Le Concepteur renseigne les métadonnées : nom, description, licence, tags, catégorie (`base`).
3. Il soumet le formulaire → `POST /api/components` (multipart/form-data).
4. Le REST Handler valide les champs : `name` non vide (max 256), `owner_id` valide, `channel_id` valide.
5. Le handler appelle `service.AddFull(AddRequest{...})`.
6. Le service calcule le SHA-256 du fichier : `sha256:<hex>`.
7. **[Cible RM01]** Le service interroge Fabric (`ListModelRecords`) et compare le hash avec tous les assets existants.
8. Si aucun doublon : le service analyse la similarité SCM (seuil 50%) avec les assets existants.
9. Le service téléverse le fichier vers IPFS (`fileStorage.Upload(filePath)`) → retourne une référence de stockage.
10. Le service construit le `Model3D` avec un UUID généré (`generateID()`), le hash, la référence IPFS.
11. Le service soumet la transaction `StoreModel` sur Fabric (`blockchain.StoreModelRecord(m)`).
12. Fabric valide la transaction et ancre le bloc.
13. L'API retourne `201 Created` avec le `Model3D` JSON (ID, name, hash, blockID…).
14. L'interface affiche le composant créé avec son UUID.

### Flux alternatif — Import depuis un format CAO non natif (STL, STEP, OBJ)

1. Le fichier est dans un format supporté mais non natif.
2. Le système accepte le fichier et calcule son SHA-256 normalement.
3. Les métadonnées géométriques extractibles automatiquement (dimensions, volume) sont pré-renseignées selon le format.
4. Le Concepteur complète les métadonnées non extractibles (description, licence).
5. La transaction est soumise normalement (flux nominal à partir de l'étape 7).

### Flux alternatif — Catégorie dérivée (non-`base`)

1. Le Concepteur sélectionne une catégorie dérivée (`amelioration`, `variation`, `adaptation`, `derivation`, `extension`, `regression`, `decoupage`).
2. Il renseigne le `parent_id` (UUID de l'asset parent sur le canal).
3. Si une `license_id` est fournie : le service vérifie la compatibilité de licence avec le parent (`CheckLicenseCompatibility`). En cas d'incompatibilité : erreur retournée avant soumission Fabric.
4. La vérification anti-plagiat SHA-256 + SCM n'est **pas** déclenchée pour les catégories dérivées.
5. La transaction est soumise avec `ParentID` renseigné.

### Flux erreur — Hash déjà existant (RM01)

1. La comparaison des hashes (étape 7) détecte un doublon exact.
2. Le service retourne une erreur avant toute soumission Fabric.
3. L'API retourne `409 Conflict` : `{ "error": "Composant déjà existant — risque de plagiat. Contacter l'administration." }`.

### Flux erreur — Similarité SCM > 50% (RM01)

1. L'analyse SCM (étape 8) détecte une similarité structurelle supérieure à 50% avec un asset existant.
2. Le service retourne une erreur avant toute soumission Fabric.
3. L'API retourne `409 Conflict` : `{ "error": "Similarité trop élevée avec un composant existant. Contacter l'administration." }`.

### Flux erreur — Incompatibilité de licence (RM03)

1. `CheckLicenseCompatibility(parentLicenseID, req.LicenseID)` retourne `Compatible: false`.
2. Le service retourne l'erreur avant toute soumission Fabric.
3. L'API retourne `422 Unprocessable Entity` : `{ "error": "Incompatibilité de licence : <raison>." }`.

### Flux erreur — Échec endorsement Fabric

1. `blockchain.StoreModelRecord(m)` retourne une erreur Fabric (nœud indisponible, politique non satisfaite).
2. Le fichier uploadé sur IPFS reste (orphelin temporaire — acceptable).
3. L'API retourne `500 Internal Server Error` : `{ "error": "Erreur blockchain : <message>. Aucune donnée enregistrée." }`.

## Post-conditions

- Le `Model3D` est inscrit sur la blockchain Fabric (immuable — RM07).
- Un UUID unique est attribué au composant (généré par le service, non par le client — RM04).
- Le fichier 3D est stocké dans IPFS avec une référence dans `Versions[0].Hash`.
- Le droit d'auteur est enregistré via `OwnerID`.

## Diagramme de séquence

```plantuml
@startuml
participant "Navigateur" as Browser
participant "REST Handler\n(adapters/in/rest/)" as REST
participant "Model Service\n(domain/model/)" as ModelSvc
database "IPFS\n(adapters/out/ipfs/)" as IPFS
database "Fabric\n(adapters/out/fabric/)" as Fabric

Browser -> REST : POST /api/components\n(multipart: name, file, category, license_id, parent_id, tags)
REST -> REST : ParseMultipartForm(32 MB)
REST -> REST : validateFields(name, owner_id, channel_id)

alt Champ invalide (name vide, owner_id malformé…)
    REST --> Browser : 400 Bad Request { error }
else Champs valides
    REST -> ModelSvc : AddFull(AddRequest{FilePath, Name, Category, ParentID, LicenseID, ...})

    alt ParentID && LicenseID renseignés
        ModelSvc -> Fabric : GetModelRecord(ParentID)
        Fabric --> ModelSvc : parentModel
        ModelSvc -> ModelSvc : CheckLicenseCompatibility(parent.LicenseID, req.LicenseID)
        alt Incompatible
            ModelSvc --> REST : ErrLicenseIncompatibility
            REST --> Browser : 422 Incompatibilité de licence
        end
    end

    ModelSvc -> ModelSvc : hashFile(FilePath) → sha256:<hex>

    note over ModelSvc : [Cible RM01 — non implémenté]\nComparer hash avec Fabric ListModelRecords
    ModelSvc -> Fabric : ListModelRecords(channelID)
    Fabric --> ModelSvc : existingAssets[]

    alt Hash doublon détecté
        ModelSvc --> REST : ErrHashDuplicate
        REST --> Browser : 409 Composant déjà existant
    else
        ModelSvc -> ModelSvc : analyseSCM(file, existingAssets)
        alt Similarité > 50%
            ModelSvc --> REST : ErrSimilarityTooHigh
            REST --> Browser : 409 Similarité trop élevée
        else
            ModelSvc -> IPFS : Upload(FilePath)
            IPFS --> ModelSvc : storageRef
            ModelSvc -> ModelSvc : buildModel3D(UUID, hash, storageRef)
            ModelSvc -> Fabric : StoreModelRecord(model3D)
            alt Échec Fabric
                Fabric --> ModelSvc : ErrEndorsement
                ModelSvc --> REST : ErrBlockchain
                REST --> Browser : 500 Erreur blockchain
            else Succès
                Fabric --> ModelSvc : blockID
                ModelSvc --> REST : Model3D{ID, Name, Hash, BlockID}
                REST --> Browser : 201 Created { model3D }
            end
        end
    end
end

@enduml
```

## Règles métier déclenchées

| Règle | Description |
|-------|-------------|
| **RM01** | Anti-plagiat obligatoire pour tout asset `base` : SHA-256 + SCM > 50% → rejet |
| **RM02** | Catégorie obligatoire parmi les 8 types : `base`, `amelioration`, `variation`, `adaptation`, `derivation`, `extension`, `regression`, `decoupage` |
| **RM03** | Si `ParentID != ""` et `LicenseID != ""` : vérification de compatibilité de licence obligatoire |
| **RM04** | UUID généré par le service, jamais par le client |
| **RM05** | `ParentID` obligatoire pour tout asset non-`base` |
| **RM07** | Validation complète côté serveur avant toute soumission blockchain |

## Exigences non-fonctionnelles

| ID | Exigence |
|----|---------|
| **EF10** | Import de fichiers 3D (STL, STEP, OBJ minimum) |
| **EF11** | Vérification anti-plagiat obligatoire avant enregistrement |
| **ENF12** | Contrôle du rôle `contributor` côté serveur avant toute écriture |
| **ENF30** | En cas d'échec blockchain, l'état local (draft) est conservé intact |

## Notes d'implémentation

**Route existante :** `POST /api/components` → `handler.createAsset()` → `service.AddFull()` → `fabric.StoreModelRecord()` + `ipfs.Upload()`.

**Écart E4 (RM01 incomplet) :** `service.AddFull()` calcule le SHA-256 mais ne compare pas avec les assets existants. À implémenter : appel `blockchain.ListModelRecords(channelID)` suivi d'une comparaison de hashes avant `fileStorage.Upload()`.

**Écart E1 (catégorie `decoupage` absente) :** `domain/model/entity.go` ne définit pas `CategoryDecoupage`. À ajouter : `CategoryDecoupage Category = "decoupage"`. Cette catégorie est la seule qui transforme un composant en module (UCAM05).

**Analyse SCM :** L'algorithme de similarité structurelle (SCM > 50%) est un service domaine indépendant à créer dans `domain/model/` — il n'est pas encore implémenté.

**Multipart fields acceptés :** `name`, `description`, `owner_id`, `channel_id`, `category`, `parent_id`, `license_id`, `tags` (CSV), `links` (JSON array), `file` ou `stl` (fichier binaire), `thumbnail` (data-URL).
