# MCD — Modèle Conceptuel de Données

> Phase 3 — Arrington | Référence : `specs/2-Analyse/Analyse_des_besoins.md` §6.3

---

## 1. Vue d'ensemble

Le modèle de données de Myr est distribué sur quatre supports de persistance aux responsabilités distinctes :

| Support | Adapter | Entités | Mutabilité |
|---------|---------|---------|-----------|
| SQLite (`myr.db`) | `adapters/out/sqlite/` | User, RefreshToken, EncryptedWallet | Mutable |
| JSON files (`data/`) | `adapters/out/localstorage/` | Connection, AssetInterface, InterfaceRefs, NetworkProfile, Session | Mutable |
| HyperLedger Fabric | `adapters/out/fabric/` | Model3D (asset), ModuleVersion (hash) | Immuable |
| IPFS | `adapters/out/ipfs/` | Fichiers CAO (3D) — référencés par `Model3D.Hash` | Immuable (CID) |

---

## 2. MCD global

```plantuml
@startuml
skinparam linetype ortho
skinparam entity {
  BackgroundColor #FEFECE
  BorderColor #A80036
}

' ── Auth ─────────────────────────────────────────────
entity "User" as USR {
  * id : uuid <<PK>>
  --
  * email : string <<unique>>
  * password : string <<bcrypt>>
  * display_name : string
  * role : enum(reader,contributor,admin)
  * org_id : string <<FK NetworkProfile.msp_id>>
  * status : enum(active,suspended)
  * created_at : timestamp
}

entity "RefreshToken" as RT {
  * id : uuid <<PK>>
  --
  * user_id : uuid <<FK User>>
  * token_hash : string <<SHA-256>>
  * expires_at : timestamp
  * created_at : timestamp
}

entity "EncryptedWallet" as EW {
  * user_id : uuid <<FK User>>
  * handle : string <<PK, pseudo@org>>
  --
  * org_id : string
  * cert_pem : string <<X.509>>
  * enc_key : string <<AES-256-GCM b64>>
  * key_iv : string <<12B b64>>
  * ca_cert_pem : string
  * status : enum(pending,active)
  * created_at : timestamp
}

' ── Identity ─────────────────────────────────────────
entity "MyrIdentity" as MI {
  * id : string <<PK>>
  --
  * display_name : string
  * legal_name : string
  * email : string
  * country : string
  * organization : string
  license_default : string
  * ip_agreement : bool
  ip_agreed_at : timestamp
  * status : enum(pending,active,suspended)
  * role : enum(reader,contributor,auditor,admin)
  * created_at : timestamp
}

entity "AccountRequest" as AR {
  * id : uuid <<PK>>
  --
  * pseudo : string
  * display_name : string
  * email : string
  * org_id : string
  message : string
  * status : enum(pending,approved,rejected)
  * created_at : string <<ISO 8601>>
}

entity "Session" as SES {
  * name : string <<PK, pseudo@org>>
  --
  * org_id : string
  * created_at : timestamp
}

' ── Network ──────────────────────────────────────────
entity "NetworkProfile" as NP {
  * id : uuid <<PK>>
  --
  * name : string
  * peer_endpoint : string
  * gateway_peer : string
  * msp_id : string
  * cert_path : string
  * key_path : string
  * tls_cert_path : string
  * fabric_channel : string
  channels : json
  chaincode : json <<ChaincodeConfig>>
  ca_endpoint : string
  ca_name : string
  server_url : string
  allow_auto_guest : bool
  allow_auto_register : bool
  auto_register_role : string
  * active : bool
  * created_at : timestamp
}

entity "Channel" as CH {
  * id : string <<PK, nom Fabric>>
  --
  * name : string
}

' ── Model ────────────────────────────────────────────
entity "Model3D" as M3D {
  * id : uuid <<PK>>
  --
  * name : string
  description : string
  * category : enum(base,amelioration,variation,\nadaptation,derivation,extension,\nregression,decoupage)
  parent_id : uuid <<FK Model3D, self>>
  block_id : string <<Fabric txID>>
  hash : string <<SHA-256 fichier CAO>>
  * channel_id : string <<FK Channel>>
  * owner_id : uuid <<FK User>>
  license_id : string
  tags : json
  links : json
  * created_at : timestamp
  ' -- Module seulement --
  status : enum(draft,submitted)
  assemblies : json <<[]connID>>
  workspace_instances : json
  module_versions : json
}

entity "Version" as VER {
  ' embedded dans Model3D.Versions
  * number : int
  --
  * hash : string
  * created_at : timestamp
}

entity "AssetInterface" as AI {
  * id : uuid <<PK>>
  --
  * asset_id : uuid <<FK Model3D>>
  name : string
  * category : string <<ELEC,MECA,HYD,...>>
  tag : string <<à ajouter — E2>>
  * type : string
  * direction : enum(in,out,bidir)
  * value_min : float
  * value_max : float
  * is_range : bool
  unit : string
  * virtual : bool
}

entity "Connection" as CON {
  * id : uuid <<PK>>
  --
  * from : uuid <<FK Model3D>>
  * to : uuid <<FK Model3D>>
  label : string
  from_iface_id : uuid <<FK AssetInterface>>
  to_iface_id : uuid <<FK AssetInterface>>
  from_instance_id : uuid <<FK WorkspaceInstance>>
  to_instance_id : uuid <<FK WorkspaceInstance>>
  fastener_asset_id : uuid <<FK Model3D>>
  * incompatible : bool
}

entity "WorkspaceInstance" as WI {
  * id : uuid <<PK>>
  --
  * asset_id : uuid <<FK Model3D>>
  * x : float
  * y : float
}

entity "ModuleVersion" as MV {
  * number : int <<PK dans le module>>
  --
  * assemblies : json <<snapshot []connID>>
  * hash : string
  note : string
  * created_at : timestamp
  block_id : string <<Fabric txID>>
}

entity "InterfaceRefs" as IR {
  ' singleton par réseau
  --
  * categories : json <<[]string>>
  * types : json <<map cat→[]type>>
  * units : json <<map cat→[]unit>>
}

' ── Payment ──────────────────────────────────────────
entity "Payment" as PAY {
  * id : uuid <<PK>>
  --
  * from : uuid <<FK User>>
  * to : uuid <<FK User>>
  * model_id : uuid <<FK Model3D>>
  * amount : float
  * created_at : timestamp
}

' ── Relations ────────────────────────────────────────

USR ||--o{ RT : "owns"
USR ||--o{ EW : "has wallet"
USR ||--o{ M3D : "owns (OwnerID)"

NP ||--o{ CH : "has channels"

M3D |o--o{ M3D : "derived from (ParentID)"
M3D ||--o{ VER : "has versions"
M3D ||--o{ AI : "has interfaces\n(InterfaceStore)"
M3D ||--o{ WI : "has workspace\ninstances (module)"
M3D ||--o{ MV : "has module\nversions"

CON }o--|| M3D : "from asset"
CON }o--|| M3D : "to asset"
CON }o--o| AI : "from interface"
CON }o--o| AI : "to interface"
CON }o--o| WI : "from instance"
CON }o--o| WI : "to instance"
CON }o--o| M3D : "fastener asset"

M3D }o--|| CH : "on channel"

PAY }o--|| USR : "from user"
PAY }o--|| USR : "to user"
PAY }o--|| M3D : "for asset"

SES ..> EW : "references\n(Name = Handle)"
MI ..> USR : "<<mapping conceptuel>>\nmême utilisateur"
AR ..> MI : "managed by admin"

@enduml
```

---

## 3. Agrégats détaillés

### Agrégat Utilisateur (D1 — auth)

```plantuml
@startuml
skinparam linetype ortho
entity "User" as USR {
  * id : uuid <<PK>>
  * email : string <<unique>>
  * role : reader | contributor | admin
  * status : active | suspended
}
entity "RefreshToken" as RT {
  * id : uuid
  * user_id <<FK>>
  * token_hash <<SHA-256>>
  * expires_at
}
entity "EncryptedWallet" as EW {
  * handle <<PK, pseudo@org>>
  * user_id <<FK>>
  * cert_pem <<X.509>>
  * enc_key <<AES-256-GCM>>
  * status : pending | active
}
USR ||--o{ RT : "owns"
USR ||--o{ EW : "has wallet"
@enduml
```

### Agrégat Identité Blockchain (D1 — identity)

```plantuml
@startuml
skinparam linetype ortho
entity "MyrIdentity" as MI {
  * id : string
  * email : string
  * status : pending | active | suspended
  * role : reader | contributor | auditor | admin
  ip_agreement : bool
}
entity "WalletEntry" as WE {
  * handle <<pseudo@org>>
  * org_id : string
  * status : lu depuis X.509
  * msp_dir : chemin absolu
}
entity "AccountRequest" as AR {
  * id : uuid
  * pseudo : string
  * status : pending | approved | rejected
}
MI ..> WE : "handle = pseudo@org"
AR ..> MI : "admin creates identity"
@enduml
```

### Agrégat Réseau (D2)

```plantuml
@startuml
skinparam linetype ortho
entity "NetworkProfile" as NP {
  * id : uuid
  * name : string
  * peer_endpoint : string
  * msp_id : string
  * active : bool
  allow_auto_guest : bool
  allow_auto_register : bool
}
entity "ChaincodeConfig" as CC {
  ' embedded dans NetworkProfile
  * name : string
  version : string
  checksum : string
}
entity "Channel" as CH {
  * id : string <<nom Fabric>>
}
NP ||--|| CC : "has chaincode config\n(embedded)"
NP ||--o{ CH : "has channels"
@enduml
```

### Agrégat Asset (D3/D4/D5/D6)

Voir MCD global §2 — entités Model3D, AssetInterface, Connection, WorkspaceInstance, ModuleVersion.

Distinction Composant vs Module :

| Critère | Composant | Module |
|---------|-----------|--------|
| `Hash` | non vide (SHA-256 fichier CAO) | vide |
| `WorkspaceInstances` | vide | non vide |
| `Status` | vide | `draft` ou `submitted` |
| `ModuleVersions` | vide | non vide si `submitted` |

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

## 4. Règles d'intégrité (contraintes MCD)

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
| RM13 | `AssetInterface` | tout asset dans l'Atelier possède toujours au moins un `Virtual: true` |
| RM14 | `WorkspaceInstance` | suppression → cascade sur toutes les `Connection` liées |
| RM15 | `WorkspaceInstance` | plusieurs instances du même `AssetID` dans un module sont indépendantes |
| RM16 | `Model3D` (module) | `CreateModule` → `Status = draft` obligatoire |
| RM17 | `Model3D` (module) | `SubmitModule` → `len(Assemblies) > 0` requis |
| RM18 | `ModuleVersion` | créée à `SubmitModule` — immuable, horodatée, hashée |
| RM19 | `Model3D` (module) | `Status = submitted` → lecture seule — toute modification crée un fork |
| RM21 | `User.Role` | création → `reader` (E3 : le code assigne `contributor`) |
| RM25 | `Model3D.OwnerID` | transfert définitif et immuable sur Fabric |
| RM26 | `Model3D.ID` | UUID préservé lors du clonage inter-réseaux |

---

## 5. Informations manquantes

- **Commission, Order, OrderItem** : entités D7 à concevoir (voir §3 Agrégat Paiement)
- **Licence** : `Model3D.LicenseID` référence un catalogue de licences (`domain/model/license.go`) — entité `License` à ajouter au MCD
- **Organization** : entité logique référencée par `User.OrgID` et `NetworkProfile.MSPID` — pas d'entité Go dédiée dans le code
- **Peer** : entité infrastructure Fabric (peer endpoint) — pas modélisée côté applicatif
- **Adresse de livraison** : UCPI01 mentionne une adresse dans le profil consommateur — entité `ConsumerProfile` absente
