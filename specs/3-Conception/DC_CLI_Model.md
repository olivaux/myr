# DC — CLI Modèle : Référence des commandes composant / interfaces / module

> Phase 3 — Arrington | Use cases : UCCE01–06, UCAM01–03/05/07/08, UCMOD01–06, UCCL01, UCREC01–05 | Outil : `myr` (`bin/myr-linux`)

---

## 1. Objectif

Ce document définit le **contrat d'interface CLI** couvrant les capacités du domaine `model` (composants, interfaces, liaisons, instances, modules). Il joue pour ces use cases le rôle que `DC_CLI_Admin.md` joue pour l'administration réseau (UCADM01–05) : référence unique du nommage des commandes, afin que les use cases de `specs/1-Expression/` et `specs/2-Analyse/` pointent vers une nomenclature stable.

**Parité CLI/REST :** voir [Architecture_Hexagonale.md](Architecture_Hexagonale.md) (CLI Handler et REST Handler consomment tous deux `ModelService`) — ce document liste les commandes `myr model`, chacune ayant un appel API REST équivalent (flux nominal des deux côtés).

**Qui exécute ces commandes ?** Conformément au principe d'exécution distante, le CLI ne tourne jamais sur le poste d'un Concepteur ou d'un Consommateur — uniquement sur le serveur, via SSH. Les commandes ci-dessous sont donc typiquement exécutées par l'administrateur du serveur **pour le compte d'une identité** (`--owner-id`, `--as`), à des fins de script, d'import en masse, de support ou de restauration ; un client de l'API REST (interface graphique tierce, plugin, boutique partenaire…) peut exécuter la même action directement pour son propre compte.

---

## 2. Arbre de commandes

```
myr model
├── add <file>                          — publier un composant                            UCCE01/03/04/05
│                                          (--draft pour soumission différée, RM16)
├── submit <id>                         — soumettre un composant en brouillon              UCCE01, RM16/RM19
├── get <id>                            — afficher un composant                            UCCL01
├── list                                — lister les composants                            UCCL01
├── verify <id>                         — vérifier l'intégrité
├── update <id>                         — modifier les métadonnées                         UCCE02
├── remove <id>                         — retirer un composant
├── children <parentID>                 — lister les dérivés d'un composant                UCREC03
├── thumbnail set <id> <fichier>        — associer une miniature
├── thumbnail get <id>                  — récupérer la miniature
├── license list                        — lister le catalogue de licences
├── license get <id>                    — afficher une licence
├── license check                       — vérifier une compatibilité de licence            UCCE04
├── interface
│   ├── add <assetID>                   — définir une interface                            UCAM03, UCCE06
│   ├── update <id>                     — modifier une interface
│   ├── remove <id>                     — supprimer une interface
│   ├── list <assetID>                  — visualiser les interfaces d'un asset             UCAM02
│   └── get <id>                        — afficher une interface
├── ref
│   ├── list                            — afficher le vocabulaire (catégories/types/unités)
│   ├── add-category <cat>              — étendre le vocabulaire
│   ├── add-type <cat> <type>
│   └── add-unit <cat> <unit>
├── link
│   ├── add                             — créer une liaison entre deux interfaces          UCAM01, UCAM07
│   ├── connect-virtual                 — relier un slot virtuel à une interface physique   UCAM03
│   ├── remove <id>                     — supprimer une liaison
│   └── list                            — visualiser les liaisons d'un module               UCAM02
└── instance
    ├── add <moduleID> <assetID>        — ajouter un composant existant comme instance d'un module UCMOD01, UCAM05
    └── remove <moduleID> <instanceID>  — retirer une instance d'un module (cascade)        UCAM08

myr module
├── create                              — créer un module (état draft)                     UCMOD01, UCAM05
├── get <id>                            — visualiser la composition d'un module            UCMOD04
├── list                                — lister les modules                               UCMOD02
├── interfaces <id>                     — interfaces exposées d'un module                  UCAM02
├── add-assembly <id> <connID>          — rattacher une liaison au module                  UCMOD02
├── remove-assembly <id> <connID>       — détacher une liaison du module
├── submit <id>                         — soumettre le module à la blockchain              UCMOD06
└── remove <id>                         — retirer un module non soumis
```

Voir § 5 pour la table de correspondance complète entre méthode du service domaine, commande CLI et use case.

---

## 3. Détail des commandes les plus citées

### 3.1 `myr model add`

```
myr model add <file> --name <nom> --channel <id> [--category <cat>] [--parent <id>] [--license <id>] [--description <texte>] [--tags <a,b>] [--owner-id <id>] [--draft]
```

Appelle `modelSvc.AddFull(AddRequest{...})`, qui expose `Category`, `ParentID`, `LicenseID`, `Description` en plus de `Name`/`Channel`/`Tags` — le strict équivalent CLI de `POST /api/components` (voir UCCE01, UCCE03, UCCE04, UCCE05 en `2-Analyse`).

**`--draft` (RM16, ADR-02 `Conception_intro.md`) :** optionnel, `false` par défaut — le comportement nominal (une seule transaction Fabric, composant `submitted` immédiatement) reste inchangé. Si `--draft` est passé, le composant est créé `Status: draft` : ses interfaces (`myr model interface add/update`, UCCE06/UCAM03) restent alors en brouillon local jusqu'à `myr model submit` (§ 3.6bis).

### 3.2 `myr model interface add`

```
myr model interface add <assetID> --category <ELEC|MECA|HYD|...> --type <type> --direction <in|out|bidir> [--value-min <f>] [--value-max <f>] [--unit <u>] [--name <label>]
```

Appelle `ModelService.AddInterface(*AssetInterface)`. Équivalent CLI de UCAM03 (« Créer une interface ») et UCCE06 (« Ajouter une interface »).

### 3.3 `myr model interface list`

```
myr model interface list <assetID>
```

Appelle `ModelService.ListInterfacesForAsset(assetID)` (ou `ModelService.GetModuleInterfaces(id)` si l'ID désigne un module — la commande détecte le type via `Get`/`GetModule`). Équivalent CLI de UCAM02 (« Visualiser les interfaces »). Sortie texte : une ligne par interface (`id`, `category`, `type`, `direction`, valeur/plage, `unit`, `virtual`).

### 3.4 `myr model link add`

```
myr model link add --from <ifaceID> --to <ifaceID> [--fastener <assetID>] [--label <texte>] [--from-instance <id>] [--to-instance <id>]
```

Appelle `ModelService.AddAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID)`. Équivalent CLI de UCAM01 (« Liaison entre interfaces ») et UCAM07 (« Asset d'accroche » — flag `--fastener`). La vérification de compatibilité (RM10/RM11) est faite par le service — comportement strictement identique quel que soit le canal (CLI ou REST), les interfaces étant désignées par identifiant explicite.

### 3.5 `myr model instance add` / `remove`

```
myr model instance add <moduleID> <assetID>
myr model instance remove <moduleID> <instanceID>
```

Appellent respectivement `AddAssetToWorkspace` et `RemoveAssetFromWorkspace`. `add` est l'équivalent CLI du placement unitaire décrit dans UCMOD01 (assemblage d'un module) et UCAM05 (transformation composant → module) — une invocation ajoute un composant comme instance ; un placement en lot reste un script shell côté appelant (boucle sur cette commande), pas une commande dédiée. `remove` est l'équivalent CLI de UCAM08 (retrait en cascade — la cascade des connexions est gérée par le service, RM15).

### 3.6 `myr module create` / `submit`

```
myr module create --name <nom> --channel <id> [--owner-id <id>] [--description <texte>] [--license <id>]
myr module submit <moduleID> [--note <texte>]
```

Appellent `CreateModule(ModuleRequest{...})` et `SubmitModule(moduleID, note)`. Équivalents CLI de UCMOD01 et UCMOD06. `submit` échoue si le module n'a aucun assemblage (RM14) — même message d'erreur quel que soit le canal.

### 3.6bis `myr model submit` (RM16/RM19 — soumission d'un composant en brouillon)

```
myr model submit <assetID>
```

Équivalent CLI de UCCE01 (« Flux alternatif — Création en brouillon ») et de la précondition « composant en brouillon » d'UCCE06. Committe l'état courant du brouillon (interfaces incluses, `Model3D.Interfaces`) sur Fabric en une transaction et passe `Status` à `submitted`. **Écart de conception (E8, `specs/roadmap_dev.md` § Écarts structurels — modèle & chaincode) :** la méthode `ModelService` dédiée généralise la logique déjà utilisée par `SubmitModule` (qui, malgré son nom historique, ne fait qu'ancrer l'état courant d'un `Model3D` sur Fabric) plutôt que d'en écrire une seconde implémentation ; voir § 6 point 4. Sans argument requis au-delà de l'ID : contrairement à `myr module submit`, aucune vérification d'assemblage (RM17, module uniquement) ne s'applique à un composant.

### 3.7 `myr model to-module` (UCAM05 — transformation composant → module)

```
myr model to-module <assetID> --name <nom>
```

Pas de méthode dédiée dans `ModelService` : la transformation (« découpage », `Category = decoupage`, cf. écart E1 dans `Architecture_Composition.md`) s'implémente comme `CreateModule` avec une référence au composant d'origine puis dépréciation de celui-ci. Documenté ici comme cible ouverte — voir § 6 point 1.

---

## 4. Format de sortie et erreurs

Mêmes conventions que `DC_CLI_Admin.md` § 7 : succès sur stdout, erreurs sur stderr via `RunE`, tableaux alignés au tabwriter. Les messages d'erreur métier (anti-plagiat RM01, licence incompatible RM03, interface déjà utilisée RM09, seuil de nœuds RM27 non applicable ici…) sont ceux renvoyés par le service domaine — identiques à ceux de l'API REST, seul le canal de sortie change (texte terminal vs JSON HTTP).

---

## 5. Table de correspondance méthode domaine → commande CLI → use case

| Méthode `ModelService` (`domain/model/port_in.go`) | Commande CLI | Use case(s) |
|---|---|---|
| `Add` / `AddFull` (avec `Draft bool`) | `myr model add [--draft]` | UCCE01, UCCE03, UCCE04, UCCE05 |
| `Submit` (généralise `SubmitModule` à tout `Model3D`) | `myr model submit` | UCCE01 (brouillon), UCCE06 (RM16/RM19) |
| `Get` | `myr model get` | UCCL01, UCREC01 |
| `List` | `myr model list` | UCCL01 |
| `Verify` | `myr model verify` | — (intégrité, hors périmètre user) |
| `UpdateAsset` | `myr model update` | UCCE02 |
| `Remove` | `myr model remove` | — |
| `GetChildren` | `myr model children` | UCREC03, UCREC04 |
| `SaveThumbnail` / `GetThumbnail` | `myr model thumbnail set/get` | — |
| `AddInterface` / `UpdateInterface` / `RemoveInterface` | `myr model interface add/update/remove` | UCAM03, UCCE06 |
| `ListInterfacesForAsset` / `GetInterface` | `myr model interface list/get` | UCAM02 |
| `EnsureVirtualSlot` | (appelé automatiquement par `interface list`) | UCAM03 (RM13) |
| `GetRefs` / `AddRefCategory` / `AddRefType` / `AddRefUnit` | `myr model ref list/add-category/add-type/add-unit` | UCAM03 |
| `AddConnection` / `AddAssemblyLink` / `RemoveConnection` / `ListConnections` | `myr model link add/remove/list` | UCAM01, UCAM07 |
| `ConnectVirtualToPhysical` | `myr model link connect-virtual` | UCAM03 |
| `CreateModule` / `GetModule` / `ListModules` / `RemoveModule` | `myr module create/get/list/remove` | UCMOD01, UCMOD02, UCMOD04 |
| `AddAssemblyToModule` / `RemoveAssemblyFromModule` | `myr module add-assembly/remove-assembly` | UCMOD02 |
| `SubmitModule` | `myr module submit` | UCMOD06 |
| `GetModuleInterfaces` | `myr module interfaces` | UCAM02 |
| `AddAssetToWorkspace` / `RemoveAssetFromWorkspace` | `myr model instance add/remove` | UCMOD01, UCAM05 (add) · UCAM08 (remove) |
| `ListLicenses` / `GetLicense` / `CheckLicenseCompatibility` / `CheckModuleLicenseCompatibility` | `myr model license list/get/check` | UCCE04, UCMOD06 |

Recherches et export (UCCL01, UCREC01–05) se combinent à partir de `List`, `Get`, `GetChildren`, `GetModuleInterfaces` côté service. Le point ouvert de savoir si `ModelService` doit exposer une méthode de filtre serveur dédiée (`Search(criteria)`), plutôt que de laisser le filtrage au client sur le résultat de `List`, est documenté en § 6 point 2 ; la commande `myr model search --filter <critère>` dépend de cette décision. Le détail des méthodes pour UCREC02–05 (`FindCompatibleAssets`, `GetLineage`, `FindModulesUsingComponent`, `ResolveBOM`) est conçu dans `DC_D8_Recherche.md`, pas ici.

---

## 6. Écarts et points ouverts

| # | Écart / question | Impact |
|---|---|---|
| 1 | `myr model to-module` (UCAM05) n'a pas de méthode `ModelService` dédiée — la transformation composant → module (catégorie `decoupage`, cf. écart E1 `Architecture_Composition.md`) reste à concevoir au niveau service avant d'être exposée en CLI comme en REST. | UCAM05 non exposable tant que E1 n'est pas résolu, quel que soit le canal (GUI, REST ou CLI) — ce n'est pas un écart spécifique au CLI. |
| 2 | `ModelService` n'expose aucune méthode de filtre serveur (`Search(criteria)`) — `List(channelID)` retourne tout le canal, à charge du dépôt GUI externe de filtrer. Reste à trancher si le filtrage doit devenir un comportement serveur. | UCCL01 et UCREC01–05 : la commande `myr model search` ne peut être qu'un alias de `list` tant que cette décision n'est pas prise et le filtre remonté côté domaine. |
| 3 | Tarification, commission, transfert de PI, clonage inter-réseau, écoconception (UCPI01/02/04/05/06/07/08/09/10/11) et automatisation (UCAUT01/02/04) n'ont aucun port domaine ni entité correspondante (`Price`, `Commission`, `Transfer`…absents de `domain/model` et `domain/payment`). Documentés dans les UC concernés comme commandes CLI de niveau 2 : le nom de commande est proposé, mais dépend d'abord de la conception du domaine (hors périmètre de ce document). | Pas d'implémentation CLI possible avant modélisation du domaine correspondant. |
| 4 | `myr model submit` (§ 3.6bis) n'a pas de méthode `ModelService` dédiée : elle nécessite un changement domaine (voir `specs/roadmap_dev.md` § Écarts structurels — modèle & chaincode, E8) — ajouter `Status`/`Draft` à `AddRequest` et une méthode `Submit` généralisant `SubmitModule` à tout `Model3D`. | UCCE01 (flux brouillon) et UCCE06 (fork) dépendent de ce changement domaine, contrairement aux autres commandes de ce document qui n'exigent qu'un adaptateur CLI sur des méthodes `ModelService` déjà définies. |

---

## 7. Décisions de conception

| ID | Décision | Raison |
|----|---------|--------|
| DC-CLIM-01 | Un seul groupe `myr model` porte composants, interfaces, liaisons et instances ; `myr module` reste séparé pour les opérations propres aux modules (création, soumission) | Miroir de la distinction domaine `Model3D` (composant/module unifié) vs. use cases (UCCE/UCAM d'un côté, UCMOD de l'autre) — évite un groupe `myr model` démesuré tout en gardant `model add/get/list/verify` stables (rétrocompatibilité de la commande existante) |
| DC-CLIM-02 | Les commandes CLI n'ajoutent aucune vérification propre — elles délèguent entièrement au service domaine | Garantit que le comportement (RM01, RM03, RM09-11, RM13-15) est strictement identique quel que soit le canal (CLI ou REST), conformément au principe de parité fonctionnelle |
| DC-CLIM-03 | Le CLI/API ne fait que des actions brutes et directes — les identifiants (interface, instance, asset) sont fournis explicitement en argument/flag, jamais par sélection interactive | `interface list` / `link list` permettent de retrouver les identifiants nécessaires avant d'agir ; toute ergonomie de sélection visuelle relève exclusivement d'un client externe (dépôt GUI) |
