# Déploiement — Infrastructure Myr

> Phase 3 — Arrington | Référence : `specs/3-Conception/Architecture_Hexagonale.md`

---

## 1. Vue d'ensemble

Myr se déploie comme un binaire Go autonome (`myr-app.exe`) qui embarque le frontend (SPA) et sert l'API REST. Il se connecte à une infrastructure HyperLedger Fabric externe gérée par chaque organisation.

```plantuml
@startuml
skinparam componentStyle rectangle

rectangle "Organisation A\n(Paris)" {
  [myr-app\nmyr.db\ndata/] as AppA
  database "SQLite + JSON" as DBA
  [peer0.org1.com:7051] as PeerA
  [Fabric CA\n:7054] as CAA
  AppA --> DBA
  AppA <--> PeerA : Gateway SDK v2
}

rectangle "Organisation B\n(Tokyo)" {
  [peer0.org2.com:7051] as PeerB
  [Fabric CA\n:7054] as CAB
}

rectangle "Organisation C\n(NYC)" {
  [peer0.org3.com:7051] as PeerC
  [Fabric CA\n:7054] as CAC
}

[Orderer\n:7050] as ORD

PeerA <--> ORD : mTLS
PeerB <--> ORD : mTLS
PeerC <--> ORD : mTLS
PeerA <..> PeerB : gossip mTLS
PeerA <..> PeerC : gossip mTLS

note bottom of AppA
  myr-app ne connaît que PeerA.
  Fabric synchronise PeerB et PeerC
  en arrière-plan via gossip.
end note
@enduml
```

---

## 2. Infrastructure minimale par organisation

| Élément | Détail |
|---------|--------|
| Serveur | VPS ou cloud (DigitalOcean, AWS, OVH, serveur physique) |
| OS | Linux recommandé (Ubuntu 22.04 LTS) |
| CPU | 2 vCPU minimum (4 recommandé) |
| RAM | 4 GB minimum (8 GB recommandé) |
| Stockage | 20 GB minimum (SSD recommandé) |
| IP | Fixe et publique |
| DNS | `peer0.orgname.com` → IP (fortement recommandé) |
| Ports ouverts | 7051 (peer), 7050 (orderer si hébergé), 7054 (CA), 8080 (myr-app) |
| Certificats | Fabric CA (PEM) — **pas** Let's Encrypt |
| mTLS | Natif Fabric — aucun VPN requis |

---

## 3. Binaires produits

| Binaire | Description | Commande de build |
|---------|-------------|------------------|
| `bin/myr-app.exe` | Serveur HTTP + GUI embarqué + API REST | `make app` |
| `bin/myr.exe` | CLI d'administration | `make cli` |

**Cross-compilation Linux :**
```bash
make deploy  # compile Linux + déploie via scripts/deploy.ps1
```

---

## 4. Configuration d'un profil réseau

Chaque instance `myr-app` est configurée via un `NetworkProfile` (JSON dans `data/`) et des variables d'environnement :

### Variables d'environnement obligatoires

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Clé de signature JWT (≥ 32 bytes aléatoires) |
| `WALLET_ENCRYPT_KEY` | Clé AES-256 pour les wallets SQLite (32 bytes en hex) |

### Variables optionnelles

| Variable | Défaut | Description |
|----------|--------|-------------|
| `MYR_DB_PATH` | `<data>/myr.db` | Chemin SQLite |
| `REDIS_URL` | — | Sessions Redis (multi-instances) |
| `MYR_DEV` | — | `=1` : hot-reload frontend (développement uniquement) |

### Profil de connexion Fabric

`connection-profiles/gateway-connection.json` ou `.yaml` : configuration du peer, TLS, certificats. Ce fichier est lu via `fabricadapter.ConfigFromEnv()` ou en argument de démarrage.

---

## 5. Lancement

### Mode développement (hot-reload frontend)

```bash
MYR_DEV=1 ./bin/myr-app.exe --open
# Sert ui/static/ depuis le disque — pas besoin de recompiler pour les changements JS/CSS
```

### Mode production

```bash
JWT_SECRET=<secret> WALLET_ENCRYPT_KEY=<key> ./bin/myr-app.exe
```

### CLI admin

```bash
./bin/myr.exe --help
./bin/myr.exe channel list
./bin/myr.exe model add --file ./mypart.stl --name "Vis M3" --channel greenchannel
```

---

## 6. Topologie réseau Fabric

La topologie des peers est entièrement définie dans `configtx.yaml` et distribuée sur le ledger Fabric. Myr n'a pas à gérer les connexions inter-peers.

```plantuml
@startuml
skinparam linetype ortho

package "Channel 'greenchannel'" {
  node "peer0.org1.com:7051\n(Org1MSP)" as P1
  node "peer0.org2.com:7051\n(Org2MSP)" as P2
  node "peer0.org3.com:7051\n(Org3MSP)" as P3
  node "orderer.example.com:7050\n(OrdererOrg)" as ORD

  P1 <--> ORD : mTLS (soumission tx)
  P2 <--> ORD : mTLS
  P3 <--> ORD : mTLS
  P1 <..> P2 : gossip (sync ledger)
  P1 <..> P3 : gossip
}

note right of P1
  IP fixe + DNS requis
  Port 7051 ouvert
  Certificats Fabric CA
  (pas Let's Encrypt)
end note
@enduml
```

**Annuaire des peers** : déclaré dans la configuration du canal (`configtx.yaml`), distribué sur le ledger. Chaque peer connaît les autres dès qu'il rejoint le canal — pas de DNS central.

---

## 7. Scalabilité — Mode multi-instances

Pour déployer plusieurs instances `myr-app` derrière un load balancer :

```bash
REDIS_URL=redis://localhost:6379 ./bin/myr-app.exe
```

Avec `REDIS_URL`, les sessions sont partagées entre instances via Redis. Sans `REDIS_URL`, chaque instance a ses propres sessions (mode standalone).

```plantuml
@startuml
skinparam componentStyle rectangle

[Load Balancer\n(nginx/HAProxy)] as LB
[myr-app #1] as A1
[myr-app #2] as A2
database "SQLite\n(partagé NFS ou réplication)" as DB
database "Redis\nSessions" as Redis
[HyperLedger Fabric] as Fabric

LB --> A1
LB --> A2
A1 --> DB
A2 --> DB
A1 --> Redis
A2 --> Redis
A1 --> Fabric
A2 --> Fabric
@enduml
```

> ⚠️ SQLite n'est pas conçu pour les accès concurrents multi-processus. En mode multi-instances, PostgreSQL est recommandé (non implémenté en v1 — adapter SQLite).

---

## 8. Monitoring

| Endpoint | Accès | Description |
|----------|-------|-------------|
| `GET /api/ping` | Public | Liveness sans appel Fabric |
| `GET /api/health` | Public | Health check complet (appelé par infra/load balancer) |
| `GET /api/status` | Public | Statut applicatif détaillé |
| `GET /metrics` | Public | Métriques Prometheus |

---

## 9. Génération de documentation CLI

```bash
go run ./cmd/mangen  # génère les man pages dans docs/man/
```

---

## 10. Informations manquantes

- **SQLite multi-instances** : SQLite n'est pas adapté pour plusieurs processus en écriture simultanée — migration PostgreSQL à prévoir si besoin de scalabilité horizontale
- **Rotation de clé WALLET_ENCRYPT_KEY** : aucun mécanisme de re-chiffrement des wallets existants lors d'un changement de clé
- **Sauvegardes** : stratégie de backup de `myr.db` et `data/` non documentée
- **TLS pour myr-app** : le serveur HTTP de `myr-app` ne fait pas TLS lui-même — à placer derrière un reverse proxy (nginx, Caddy) avec certificat Let's Encrypt
- **Procédure de mise à jour du chaincode** : le versionnement et l'upgrade du chaincode en production (lifecycle Fabric v2) n'est pas documenté
