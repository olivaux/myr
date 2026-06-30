# Architecture Hexagonale — Myr System

> Phase 3 — Arrington | Contrainte ENF18 vérifiée par CI : `scripts/ci/check-domain-imports.sh`

---

## 1. Principes

L'architecture hexagonale (Ports & Adapters) garantit que la logique métier (`domain/`) est indépendante de toute technologie d'infrastructure. Le domaine ne connaît que des **interfaces Go** — jamais de références à Fabric, SQLite, Redis ou IPFS.

**Règle de dépendance :** les dépendances pointent toujours **vers l'intérieur** (vers le domaine). Le domaine ne dépend de rien d'externe.

```
Adapter IN  →  Port IN  →  Service Domaine  →  Port OUT  →  Adapter OUT  →  Infrastructure
```

---

## 2. Diagramme d'architecture global

```plantuml
@startuml
title Architecture Hexagonale — Myr System

skinparam componentStyle rectangle
skinparam rectangle {
  BackgroundColor #f8f9fa
  BorderColor #6c757d
}
skinparam package {
  BackgroundColor #f0f4ff
}

' ── Entrypoints ──────────────────────────────────────
package "Points d'entrée (cmd/)" {
  [myr-app\ncmd/api/main.go] as App
  [myr-cli\ncmd/cli/main.go] as CLI
}

' ── Adapters IN ──────────────────────────────────────
package "Adapters entrants" #e8f4f8 {
  [REST Handler\nadapters/in/rest/\nhandlers*.go + server.go] as REST
  [CLI Handler\nadapters/in/cli/] as CLIHandler
}

' ── Domaine ──────────────────────────────────────────
package "Domaine métier\n(domain/)" #fff9c4 {

  package "Ports entrants (interfaces Go)" {
    interface "ModelService" as IModel
    interface "AuthService" as IAuth
    interface "IdentityService" as IIdentity
    interface "NetworkService" as INetwork
    interface "ChannelService" as IChannel
    interface "PaymentService" as IPayment
    interface "SessionService" as ISession
  }

  package "Implémentations service" {
    [model.Service\ndomain/model/service.go] as SvcModel
    [auth.Service\ndomain/auth/service.go] as SvcAuth
    [identity.Service\ndomain/identity/service.go] as SvcIdentity
    [network.Service\ndomain/network/service.go] as SvcNetwork
    [channel.Service\ndomain/channel/service.go] as SvcChannel
    [payment.Service\ndomain/payment/service.go] as SvcPayment
    [session.Service\ndomain/session/service.go] as SvcSession
  }

  package "Ports sortants (interfaces Go)" {
    interface "BlockchainPort" as PBC
    interface "FileStoragePort" as PFS
    interface "ConnectionStore" as PConn
    interface "InterfaceStore" as PIface
    interface "ThumbnailStore" as PThumb
    interface "UserStore" as PUser
    interface "TokenStore" as PToken
    interface "WalletStore" as PWallet
    interface "NetworkStore" as PNet
    interface "CAPort" as PCA
    interface "RequestStore" as PReq
    interface "PaymentPort" as PPay
  }
}

' ── Adapters OUT ─────────────────────────────────────
package "Adapters sortants" #f0f4e8 {
  [FabricBlockchain\nadapters/out/fabric/\nblockchain.go] as AdFabric
  [JSONBlockchain (fallback)\nadapters/out/localstorage/\njson_blockchain.go] as AdJSON
  [IPFSStorage\nadapters/out/ipfs/] as AdIPFS
  [SQLiteStore\nadapters/out/sqlite/] as AdSQLite
  [LocalStore\nadapters/out/localstorage/] as AdLocal
}

' ── Infrastructure ───────────────────────────────────
package "Infrastructure" {
  database "HyperLedger Fabric\n(Ledger)" as HLF
  database "IPFS\n(Fichiers 3D)" as IPFSdb
  database "SQLite\nmyr.db" as SQLiteDB
  database "JSON files\ndata/" as JSONdb
}

' ── SPA ──────────────────────────────────────────────
package "Frontend" {
  [SPA Vanilla JS\nui/static/] as SPA
}

' ── Connexions ───────────────────────────────────────
App --> REST
App --> SPA : embed.FS
CLI --> CLIHandler

REST --> IModel
REST --> IAuth
REST --> IIdentity
REST --> INetwork
REST --> IChannel
CLIHandler --> IModel
CLIHandler --> IAuth
CLIHandler --> IChannel
CLIHandler --> IPayment

IModel <|.. SvcModel
IAuth <|.. SvcAuth
IIdentity <|.. SvcIdentity
INetwork <|.. SvcNetwork
IChannel <|.. SvcChannel
IPayment <|.. SvcPayment
ISession <|.. SvcSession

SvcModel --> PBC
SvcModel --> PFS
SvcModel --> PConn
SvcModel --> PIface
SvcModel --> PThumb
SvcAuth --> PUser
SvcAuth --> PToken
SvcAuth --> PWallet
SvcIdentity --> PCA
SvcIdentity --> PReq
SvcNetwork --> PNet
SvcPayment --> PPay
SvcSession --> PUser

PBC <|.. AdFabric
PBC <|.. AdJSON
PFS <|.. AdIPFS
PConn <|.. AdLocal
PIface <|.. AdLocal
PThumb <|.. AdLocal
PNet <|.. AdLocal
PReq <|.. AdLocal
PUser <|.. AdSQLite
PToken <|.. AdSQLite
PWallet <|.. AdSQLite
PCA <|.. AdFabric
PPay <|.. AdFabric

AdFabric --> HLF
AdIPFS --> IPFSdb
AdSQLite --> SQLiteDB
AdJSON --> JSONdb
AdLocal --> JSONdb

@enduml
```

---

## 3. Règle de dépendance — vérification CI

Le script `scripts/ci/check-domain-imports.sh` vérifie qu'aucun fichier dans `domain/` n'importe de packages infrastructure :

```bash
# check-domain-imports.sh (principe)
forbidden=("fabric" "redis" "sqlite" "ipfs" "net/http")
for pkg in "${forbidden[@]}"; do
  if grep -r "\"$pkg" domain/; then
    echo "ERREUR : import interdit dans domain/"
    exit 1
  fi
done
```

**Violation → build CI échoue.**

---

## 4. Adapters entrants (IN)

| Adapter | Package | Port consommé | Rôle |
|---------|---------|--------------|------|
| REST Handler | `adapters/in/rest/` | ModelService, AuthService, IdentityService, NetworkService, ChannelService | Sert l'API REST et le GUI embarqué |
| CLI Handler | `adapters/in/cli/` | ModelService, AuthService, ChannelService, PaymentService | Commandes cobra CLI admin |

### Routes REST — niveaux d'accès

| Niveau | Middleware | Exemples de routes |
|--------|-----------|-------------------|
| Public | aucun | `/api/auth/register`, `/api/auth/login`, `/api/identity/policy` |
| Authentifié (`reader+`) | `requireAuth` | `GET /api/components`, `GET /api/modules`, `/api/refs` |
| Contributeur (`contributor+`) | `requireRole("contributor")` | `POST /api/components`, `PUT /api/modules/`, `POST /api/assembly-links` |
| Admin | `requireRole("admin")` | `/api/admin/users`, `/api/admin/sessions` |

---

## 5. Adapters sortants (OUT)

| Adapter | Package | Implémente | Infrastructure |
|---------|---------|-----------|--------------|
| `FabricBlockchain` | `adapters/out/fabric/` | `BlockchainPort`, `CAPort`, `PaymentPort` | HyperLedger Fabric Gateway SDK v2 |
| `JSONBlockchain` | `adapters/out/localstorage/` | `BlockchainPort` | JSON files (fallback sans Fabric) |
| `IPFSStorage` | `adapters/out/ipfs/` | `FileStoragePort` | IPFS (stockage fichiers 3D) |
| `SQLiteStore` | `adapters/out/sqlite/` | `UserStore`, `TokenStore`, `WalletStore` | SQLite (`myr.db`) |
| `LocalStore` | `adapters/out/localstorage/` | `ConnectionStore`, `InterfaceStore`, `ThumbnailStore`, `NetworkStore`, `RequestStore` | JSON files (`data/`) |

---

## 6. Mode de fallback (JSONBlockchain)

En développement ou si Fabric est indisponible, `JSONBlockchain` simule le comportement blockchain avec des fichiers JSON locaux. Il implémente `BlockchainPort` mais sans immuabilité réelle.

```plantuml
@startuml
skinparam classAttributeIconSize 0

interface BlockchainPort {
  + StoreModelRecord(m *Model3D) : error
  + GetModelRecord(id, channelID string) : (*Model3D, error)
  + ListModelRecords(channelID string) : ([]*Model3D, error)
  + VerifyIntegrity(id, hash, channelID string) : (bool, error)
}

class FabricBlockchain {
  - gc : *gateway.Client
  --
  + StoreModelRecord(m *Model3D) : error
  + GetModelRecord(id, channelID string) : (*Model3D, error)
  + ListModelRecords(channelID string) : ([]*Model3D, error)
  + VerifyIntegrity(id, hash, channelID string) : (bool, error)
}

class JSONBlockchain {
  - store : *JSONStore
  --
  + StoreModelRecord(m *Model3D) : error
  + GetModelRecord(id, channelID string) : (*Model3D, error)
  + ListModelRecords(channelID string) : ([]*Model3D, error)
  + VerifyIntegrity(id, hash, channelID string) : (bool, error)
}

BlockchainPort <|.. FabricBlockchain : production
BlockchainPort <|.. JSONBlockchain : dev/test fallback

@enduml
```

---

## 7. Points d'assemblage (cmd/)

`cmd/api/main.go` est le seul endroit où les adapters concrets sont instanciés et injectés dans les services domaine (injection de dépendances manuelle — pas de framework IoC) :

```
main.go
├── Crée SQLiteStore(dbPath, encKey)
├── Crée FabricBlockchain(networkProfile) ou JSONBlockchain()
├── Crée IPFSStorage() ou LocalFileStorage()
├── Crée LocalStore(dataDir)
├── Injecte dans model.NewService(blockchain, storage, connStore, ifaceStore, ...)
├── Injecte dans auth.NewService(userStore, tokenStore, walletStore, jwtSecret)
├── Injecte dans identity.NewService(caPort, requestStore)
├── Crée Handler(modelSvc, authSvc, identitySvc, ...)
├── Crée Server(handler, addr)
└── Server.Start()
```

---

## 8. Variables d'environnement

| Variable | Requis | Effet |
|----------|--------|-------|
| `JWT_SECRET` | ✅ pour auth JWT | Active l'authentification email/password — absent = auth désactivée |
| `WALLET_ENCRYPT_KEY` | ✅ pour wallets | Clé AES-256 chiffrement/déchiffrement wallets SQLite |
| `MYR_DB_PATH` | ❌ | Chemin SQLite — défaut : `<data>/myr.db` |
| `REDIS_URL` | ❌ | Active les sessions partagées Redis (multi-instances) |
| `MYR_DEV` | ❌ | `=1` : sert les statiques depuis le disque (hot-reload frontend) |
| Variables Fabric | ✅ pour Fabric | Lues via `fabricadapter.ConfigFromEnv()` ou fichier `fabric.env` |
