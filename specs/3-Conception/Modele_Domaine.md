# Modèle de domaine — Agrégats et relations

> Phase 3 — Arrington | Référence : `specs/2-Analyse/Analyse_des_besoins.md` §6.3

---

## 1. Vue d'ensemble

Le modèle de données de Myr est distribué sur trois supports de persistance aux responsabilités distinctes — il n'y a pas de base SQL :

| Support | Adapter | Entités | Mutabilité |
|---------|---------|---------|-----------|
| Fichiers MSP locaux (non chiffrés) | `adapters/out/localstorage/` (wallets) | WalletEntry (PEM), MyrIdentity (attributs CA) | Mutable |
| JSON files (`data/`, `~/.Myr/`) | `adapters/out/localstorage/` | Connection, AssetInterface (brouillon — tant que l'asset porteur n'est pas soumis, ADR-02), InterfaceRefs, NetworkProfile, Session (local CLI), AccountRequest, Role | Mutable |
| HyperLedger Fabric | `adapters/out/fabric/` | Model3D (asset), AssetInterface (embarquée dans `Model3D.Interfaces` **à partir de la soumission** — ADR-02, `Conception_intro.md`), ModuleVersion (hash) | Immuable |
| IPFS | `adapters/out/ipfs/` | Fichiers CAO (3D) — référencés par `Model3D.Hash` | Immuable (CID) |

> La session REST (token opaque, `myrSession`) n'est pas un agrégat métier — c'est un détail d'implémentation de l'adaptateur `in/rest/`, en mémoire ou JSON/Redis selon la configuration. Voir `DC_D1_Auth_Identity.md`.

Le modèle est organisé par **agrégats** (au sens DDD) plutôt que par schéma relationnel : chaque agrégat regroupe les objets dont le cycle de vie est solidaire (composition), et référence les autres agrégats uniquement par identifiant (UUID, pseudo, nom) — jamais par une contrainte d'intégrité référentielle appliquée par un moteur de base de données, puisqu'aucun des quatre supports du tableau ci-dessus n'en fournit une de façon transversale. Cette cohérence inter-agrégats est portée par le domaine (`domain/`), pas par le support de stockage.

---

## 2. Vue globale du domaine

```plantuml
@startuml
skinparam class {
  BackgroundColor #FEFECE
  BorderColor #A80036
}
hide circle
hide empty members

package "Agrégat RBAC (D1)" {
  class Role <<JSON>> {
    name : string
    permissions : json <<[]Permission>>
    built_in : bool
    created_at : timestamp
  }
}

package "Agrégat Identité (D1)" {
  class MyrIdentity <<Fabric CA>> {
    id : string
    display_name : string
    legal_name : string
    email : string
    country : string
    organization : string
    license_default : string
    ip_agreement : bool
    ip_agreed_at : timestamp
    status : enum(pending,active,suspended)
    role : enum(reader,contributor,auditor,admin)
    created_at : timestamp
  }
  class WalletEntry <<PEM>> {
    handle : string <<pseudo@org>>
    name : string
    org_id : string
    status : string <<lu depuis le certificat X.509>>
    msp_dir : string <<chemin absolu, fichiers PEM non chiffrés>>
  }
  class AccountRequest <<JSON>> {
    id : uuid
    pseudo : string
    display_name : string
    email : string
    org_id : string
    message : string
    status : enum(pending,approved,rejected)
    created_at : string <<ISO 8601>>
  }
  class Session <<JSON, compte local CLI>> {
    name : string <<pseudo@org>>
    org_id : string
    created_at : timestamp
  }
  MyrIdentity ..> WalletEntry : "handle = pseudo@org"
  AccountRequest ..> MyrIdentity : "admin crée l'identité\n(pas de flux d'approbation\nautomatisé — écart)"
  Session ..> WalletEntry : "lien conceptuel (Name = Handle)\nnon exploité par du code câblé\nactuellement"
}

package "Agrégat Réseau (D2)" {
  class NetworkProfile <<JSON>> {
    id : uuid
    name : string
    peer_endpoint : string
    gateway_peer : string
    msp_id : string
    cert_path : string
    key_path : string
    tls_cert_path : string
    fabric_channel : string
    ca_endpoint : string
    ca_name : string
    server_url : string
    allow_auto_guest : bool
    allow_auto_register : bool
    auto_register_role : string
    active : bool
    created_at : timestamp
  }
  class ChaincodeConfig <<embedded>> {
    name : string
    version : string
    checksum : string
  }
  class Channel <<JSON>> {
    id : string <<nom Fabric>>
    name : string
  }
  NetworkProfile *-- "1" ChaincodeConfig : "config chaincode"
  NetworkProfile *-- "0..*" Channel : "canaux"
}

package "Agrégat Asset (D3-D6)" {
  class Model3D <<Fabric>> {
    id : uuid
    name : string
    description : string
    category : enum(base,amelioration,variation,\nadaptation,derivation,extension,\nregression,decoupage)
    parent_id : uuid <<réf. Model3D, self>>
    block_id : string <<Fabric txID>>
    hash : string <<SHA-256 fichier CAO>>
    channel_id : string <<réf. Channel>>
    owner_id : string <<pseudo, non validé vs\nsession — écart sécurité>>
    license_id : string
    tags : json
    links : json
    created_at : timestamp
    status : enum(draft,submitted) <<tout asset — RM16/RM19\ngénéralisées ; module : toujours\ndraft à la création (RM16) ;\ncomposant : submitted par défaut,\ndraft si demandé explicitement>>
  }
  class Version <<embedded>> {
    number : int
    hash : string
    created_at : timestamp
  }
  class AssetInterface <<JSON brouillon\n→ Fabric embedded>> {
    id : uuid
    name : string
    category : string <<ELEC,MECA,HYD,...>>
    tag : string <<à ajouter — E2>>
    type : string
    direction : enum(in,out,bidir)
    value_min : float
    value_max : float
    is_range : bool
    unit : string
    virtual : bool
  }
  class Connection <<JSON>> {
    id : uuid
    label : string
    incompatible : bool
  }
  class WorkspaceInstance <<JSON>> {
    id : uuid
    x : float
    y : float
  }
  class ModuleVersion <<embedded>> {
    number : int
    assemblies : json <<snapshot []connID>>
    hash : string
    note : string
    created_at : timestamp
    block_id : string <<Fabric txID>>
  }
  class InterfaceRefs <<JSON, singleton\npar réseau>> {
    categories : json <<[]string>>
    types : json <<map cat→[]type>>
    units : json <<map cat→[]unit>>
  }

  Model3D *-- "0..*" Version : "historique"
  Model3D *-- "1..*" AssetInterface : "interfaces\n(≥1 virtuelle, RM13 ;\nbrouillon local, puis Fabric\nà la soumission)"
  Model3D *-- "0..*" WorkspaceInstance : "instances (module)"
  Model3D *-- "0..*" ModuleVersion : "versions module"
  Model3D "1" *-- "0..*" Connection : "assemblages (module)"

  Connection ..> Model3D : "from / to"
  Connection ..> AssetInterface : "from / to iface"
  Connection ..> WorkspaceInstance : "from / to instance"
  Connection ..> Model3D : "fastener asset"
}

package "Agrégat Paiement (D7)" {
  class Payment <<JSON>> {
    id : uuid
    from : string <<pseudo>>
    to : string <<pseudo>>
    model_id : uuid
    amount : float
    created_at : timestamp
  }
}

' -- Relations inter-agrégats (références logiques par ID, pas de FK enforced) --
MyrIdentity ..> Model3D : "owns (OwnerID)\n#incoherence — non validé vs session"
Model3D ..> Channel : "canal (ChannelID)"
Model3D ..> Model3D : "dérivé de (ParentID)"
Payment ..> Model3D : "pour asset (ModelID)"

note as N1
  Une flèche pleine avec losange (*--) est une composition : l'objet cible
  n'a pas de cycle de vie propre en dehors de son parent, et les deux
  vivent sur le même support de persistance.
  Une flèche en pointillés (..>) est une référence logique par identifiant
  (UUID, pseudo, nom), résolue au niveau du domaine — jamais une contrainte
  d'intégrité référentielle imposée par un moteur de base de données.
  Chaque agrégat peut vivre sur un support de persistance différent (§1) ;
  la cohérence inter-agrégats est de la responsabilité du domaine.
end note

@enduml
```

---

## 3. Détail par agrégat

### Agrégat RBAC (D1 — role)

4 rôles intégrés (`reader`, `contributor`, `auditor`, `admin`) — non supprimables. Rôles personnalisés créés via `myr role create`.

> Il n'existe pas d'agrégat « Utilisateur/compte web » — voir `DC_D1_Auth_Identity.md`. L'identité de référence est `MyrIdentity` (agrégat suivant).

### Agrégat Identité (D1 — identity)

Voir §2 — entités MyrIdentity, WalletEntry, AccountRequest, Session.

### Agrégat Réseau (D2)

Voir §2 — entités NetworkProfile, ChaincodeConfig (embarquée), Channel.

### Agrégat Asset (D3/D4/D5/D6)

Voir §2 — entités Model3D, AssetInterface, Connection, WorkspaceInstance, ModuleVersion, InterfaceRefs.

Distinction Composant vs Module :

| Critère | Composant | Module |
|---------|-----------|--------|
| `Hash` | non vide (SHA-256 fichier CAO) | vide |
| `WorkspaceInstances` | vide | non vide |
| `ModuleVersions` | vide | non vide si `submitted` |

> `Status` (`draft`/`submitted`) n'est plus un critère distinctif depuis la généralisation de RM16/RM19 — il s'applique aux deux types (voir §1, §2 et ADR-02 dans `Conception_intro.md`). Seuls `Hash`, `WorkspaceInstances` et `ModuleVersions` distinguent un composant d'un module.

### Agrégat Paiement (D7)

Entité `Payment` implémentée (paiement manuel).

**Entités à concevoir** pour RM23/RM24 :

| Entité | Champs proposés | UC déclencheur |
|--------|----------------|----------------|
| `Order` | id, consumer_id, module_id, amount, status, created_at | UCPI01 |
| `OrderItem` | order_id, manufacturer_id, component_id, quantity | UCPI01 |
| `Commission` | id, order_id, recipient_id, asset_id, amount, ratio, tx_id | UCPI02, UCAUT01 |

> Ces entités sont à concevoir avec le PO — elles n'existent pas dans le code actuel.

---

## 4. Règles d'intégrité (invariants du domaine)

| Règle | Entité | Contrainte |
|-------|--------|-----------|
| RM01 | `Model3D` | `Category = base` → `Hash` unique parmi tous les assets Fabric |
| RM02 | `Model3D.Category` | valeur parmi les 8 catégories (dont `decoupage` — E1) |
| RM03 | `Model3D` | `ParentID != ""` → compatibilité licence vérifiée avant soumission |
| RM04 | `Model3D.ID` | UUID généré côté serveur — jamais fourni par le client |
| RM05 | `Model3D` | `Category != base` → `ParentID` obligatoire |
| RM09 | `AssetInterface` | un `id` ne peut apparaître qu'une seule fois dans `Connection.FromIfaceID` ou `ToIfaceID` |
| RM11 | `Connection` | `FromIface` et `ToIface` doivent satisfaire les 5 critères de compatibilité |
| RM12 | `Connection` | jamais supprimée automatiquement — `Incompatible: true` si interfaces évoluent |
| RM13 | `AssetInterface` | tout asset possède toujours au moins un `Virtual: true` |
| RM14 | `WorkspaceInstance` | suppression → cascade sur toutes les `Connection` liées |
| RM15 | `WorkspaceInstance` | plusieurs instances du même `AssetID` dans un module sont indépendantes |
| RM16 | `Model3D` (module) | `CreateModule` → `Status = draft` obligatoire |
| RM17 | `Model3D` (module) | `SubmitModule` → `len(Assemblies) > 0` requis |
| RM18 | `ModuleVersion` | créée à `SubmitModule` — immuable, horodatée, hashée |
| RM19 | `Model3D` (module) | `Status = submitted` → lecture seule — toute modification crée un fork |
| RM21 | `MyrIdentity.Role` | rôle `reader` par défaut à l'auto-enregistrement, sauf rôle explicite |
| RM22 | `myrSession.Role` (REST) | déterminé à la connexion depuis `MyrIdentity.Role` #incoherence — cette ligne décrit un écart de synchronisation (état de suivi : `specs/roadmap_dev.md` § Écarts Identité & Session, E1), pas la règle RM22 telle que formulée dans `Regles_Metier.md` (changement de rôle réservé à l'admin, effectif au prochain ré-enrôlement) ; à réconcilier |
| RM25 | `Model3D.OwnerID` | transfert définitif et immuable sur Fabric |
| RM26 | `Model3D.ID` | UUID préservé lors du clonage inter-réseaux |

---

## 5. Informations manquantes

- **Commission, Order, OrderItem** : entités D7 à concevoir (voir §3 Agrégat Paiement)
- **Licence** : `Model3D.LicenseID` référence un catalogue de licences (`domain/model/license.go`) — entité `License` à ajouter au modèle de domaine
- **Organization** : entité logique référencée par `MyrIdentity.Organization` et `NetworkProfile.MSPID` — pas d'entité Go dédiée dans le code
- **Chiffrement des wallets** : décision de conception à trancher entre wallet chiffré au repos ou fichiers PEM en clair (`0600`) — voir `Securite.md`, `Conception_intro.md` ADR-03
- **Peer** : entité infrastructure Fabric (peer endpoint) — pas modélisée côté applicatif
- **Adresse de livraison** : UCPI01 mentionne une adresse dans le profil consommateur — entité `ConsumerProfile` absente
