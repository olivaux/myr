# DC — CLI Admin : Référence des commandes administrateur

> Phase 3 — Arrington | Use cases : UCADM01–UCADM05, UCDEV02 | Outil : `myr-cli` (`bin/myr.exe`)

---

## 1. Objectif

Ce document définit le **contrat d'interface du CLI administrateur** (`myr.exe`). Il précise, pour chaque commande, la syntaxe complète, les drapeaux, les messages de sortie, les codes d'erreur, et le service domaine appelé.

Le CLI est le **seul point d'entrée** pour les opérations d'installation et de configuration d'un réseau Myr (UCADM01–05). L'API REST n'expose jamais ces opérations. Les opérations d'infrastructure irréversibles (`myr network destroy`) n'existent que dans ce binaire.

La structure de ce document suit la hiérarchie des commandes cobra définie dans `adapters/in/cli/`.

---

## 2. Arbre de commandes

```
myr
├── network                     — gestion des profils réseau Fabric
│   ├── list                    — liste les profils configurés
│   ├── show <id>               — affiche les détails d'un profil
│   ├── add                     — configure un nouveau profil réseau
│   ├── update <id>             — modifie un profil existant (flags fournis seulement)
│   ├── activate <id>           — définit le réseau actif
│   ├── delete <id>             — supprime un profil de configuration
│   ├── test [id]               — teste la connectivité TCP vers le peer
│   ├── import                  — importe un profil depuis un fichier JSON/YAML
│   ├── destroy <id> --confirm  — démantèle un réseau dev/test (UCADM05, irréversible)
│   ╌╌ create                   — [POST-V1] crée un réseau Fabric from scratch (configtx, genesis block)
│   └─╌ sync                    — [POST-V1] synchronise la configuration d'un canal distant
├── org
│   └── add                     — ajoute une organisation au canal Fabric (UCADM01)
├── node
│   ├── add                     — ajoute un nœud peer ou orderer au réseau (UCADM03)
│   └── remove                  — retire administrativement un nœud du réseau (UCADM04)
├── peer
│   └── add                     — enregistre un peer dans la CA et génère ses certificats (existant)
├── model                       — gestion des modèles 3D (existant)
├── channel                     — lecture des canaux Fabric (existant)
└── payment                     — commandes de paiement (existant)
```

---

## 3. Groupe `myr network` — Gestion des profils réseau

Un **profil réseau** (`NetworkProfile`) est la configuration qui permet à `myr-app` de se connecter à un peer Fabric d'une organisation. Myr se connecte à **un seul peer** (celui de son organisation) — Fabric synchronise ensuite avec les autres orgs.

### 3.1 `myr network list`

```
myr network list
```

**Comportement :** Liste tous les profils réseau configurés dans `data/networks.json`.

**Sortie :**

```
ID                   NOM                  ENDPOINT                  MSP          ACTIF
net-1700000000000    diy-network          89.167.102.193:7051       Org1MSP      *
net-1700000000001    sandbox-test         localhost:7051             Org2MSP
```

**Sortie (aucun réseau) :** `Aucun réseau configuré. Utilisez "myr network add" ou "myr network import".`

**Service :** `networkSvc.List()`

---

### 3.2 `myr network show <id>`

```
myr network show <id>
```

**Comportement :** Affiche tous les champs d'un profil réseau identifié par son `id`.

**Sortie :**

```
────────────────────────────────────────────────────
Réseau : diy-network  (id: net-1700000000000)
────────────────────────────────────────────────────
Peer endpoint    : 89.167.102.193:7051
Gateway peer     : peer0.org1.diy-network.com
MSP ID           : Org1MSP
Canal Fabric     : sandbox
Certificat       : data/certs/org1/client.pem
Clé privée       : data/certs/org1/client-key.pem
Certificat TLS   : data/certs/org1/tls-root.pem
Chaincode        : myrcc
CA endpoint      : https://89.167.102.193:7054
CA name          : ca-org1
Server URL       : (non configuré)
Auto guest       : false
Auto register    : false
Auto role        : reader
Production       : false
Actif            : true
Créé le          : 2026-01-15 10:23:45
────────────────────────────────────────────────────
```

**Erreur :** `Erreur : réseau introuvable : <id>`

**Service :** `networkSvc.List()` puis recherche par ID (aucun `Get(id)` dans le port v1 — recherche linéaire acceptable).

---

### 3.3 `myr network add`

```
myr network add --name <nom> --peer <host:port> --msp <mspID> [options]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--name` | string | ✅ | Nom lisible du réseau |
| `--peer` | string | ✅ | Endpoint du peer gateway (`host:port`) |
| `--msp` | string | ✅ | MSP ID de l'organisation (`ex: Org1MSP`) |
| `--gateway` | string | — | Nom TLS du peer gateway (ex: `peer0.org1.com`) |
| `--cert` | string | — | Chemin vers le certificat client PEM |
| `--key` | string | — | Chemin vers la clé privée PEM |
| `--tls-cert` | string | — | Chemin vers le certificat TLS du peer PEM |
| `--channel` | string | — | Nom du canal Fabric par défaut |
| `--chaincode` | string | — | Nom du chaincode déployé |
| `--ca` | string | — | URL de la CA Fabric (`https://host:port`) |
| `--ca-name` | string | — | Nom de la CA (ex: `ca-org1`) |
| `--server` | string | — | URL du MYR Server central (`https://host:port`) |
| `--auto-guest` | bool | — | Autoriser les accès invité automatiques (défaut: `false`) |
| `--auto-register` | bool | — | Autoriser l'enregistrement CA automatique (défaut: `false`) |
| `--auto-role` | string | — | Rôle auto-register : `reader`, `contributor`, `auditor` (défaut: `reader`) |
| `--production` | bool | — | Marquer ce réseau comme réseau de production — protège contre `destroy` (défaut: `false`) |

**Comportement :** Crée un nouveau profil réseau et le persiste dans `data/networks.json`. N'active PAS le réseau automatiquement.

**Sortie :**
```
Réseau "diy-network" configuré (id: net-1700000000000).
Activez-le avec : myr network activate net-1700000000000
```

**Erreurs :**
- `Erreur : le nom est requis`
- `Erreur : adresse du peer invalide "localhost" — utilisez le format host:port`
- `Erreur : un réseau avec ce nom existe déjà — utilisez "myr network update <id>"`

**Service :** `networkSvc.Add(...)`

---

### 3.4 `myr network update <id>`

```
myr network update <id> [--name <nom>] [--peer <host:port>] [--msp <mspID>] [options]
```

**Drapeaux :** mêmes que `add`, tous optionnels. Seuls les drapeaux explicitement fournis écrasent les valeurs existantes.

**Comportement :**
1. Charge le profil existant via `networkSvc.List()` (recherche par ID).
2. Fusionne les champs : pour chaque drapeau fourni (`cmd.Flags().Changed("flag")`), la nouvelle valeur remplace l'ancienne.
3. Appelle `networkSvc.Update(id, ...)` avec les valeurs fusionnées.

**Sortie :** `Réseau net-1700000000000 mis à jour.`

**Erreurs :**
- `Erreur : réseau introuvable : <id>`
- `Erreur : adresse du peer invalide "<val>" — utilisez le format host:port`

**Service :** `networkSvc.List()` + `networkSvc.Update(...)`

> **Note implémentation :** `networkSvc.Update()` remplace tous les champs — le merge doit se faire côté CLI Handler avant l'appel.

---

### 3.5 `myr network activate <id>`

```
myr network activate <id>
```

**Comportement :** Marque le profil `<id>` comme actif. Le précédent réseau actif est désactivé automatiquement.

**Sortie :** `Réseau "diy-network" activé (id: net-1700000000000).`

**Erreurs :**
- `Erreur : réseau introuvable : <id>`

**Service :** `networkSvc.Activate(id)`

---

### 3.6 `myr network delete <id>`

```
myr network delete <id> [--yes]
```

**Drapeaux :**

| Drapeau | Type | Description |
|---------|------|-------------|
| `--yes` | bool | Ignore la demande de confirmation interactive |

**Comportement :** Supprime le profil de configuration. Si le réseau est actif, affiche un avertissement supplémentaire.

**Sortie (confirmation) :**
```
Supprimer le réseau "diy-network" (net-1700000000000) ? [o/N] : o
Réseau supprimé.
```

**Sortie (--yes) :** `Réseau "diy-network" supprimé.`

**Erreurs :**
- `Erreur : réseau introuvable : <id>`
- `Avertissement : ce réseau est actuellement actif. La suppression désactive la connexion Fabric.`

**Service :** `networkSvc.Delete(id)`

> **Distinction avec `destroy` :** `delete` supprime uniquement le profil de configuration. Il ne touche pas aux processus Fabric, ni aux données ledger locales. Utilisé pour retirer un réseau mal configuré ou désusage.

---

### 3.7 `myr network test [id]`

```
myr network test [id]
```

**Comportement :** Teste la connectivité TCP vers le peer endpoint du profil. Si `id` est omis, utilise le réseau actif.

**Sortie :**
```
Test de connectivité vers 89.167.102.193:7051...
OK — peer joignable
```

> La latence n'est pas retournée par `networkSvc.TestConnection(id)` en v1 (le port retourne uniquement `error`). Un affichage de latence supposerait que le port retourne une durée — à ajouter si nécessaire en v2.

**Erreurs :**
- `Erreur : aucun réseau actif — activez-en un avec "myr network activate <id>"`
- `Erreur : impossible de joindre 89.167.102.193:7051 — vérifiez le pare-feu et la disponibilité réseau`

**Service :** `networkSvc.TestConnection(id)`

---

### 3.8 `myr network import`

```
myr network import --profile <chemin> [--name <nom>] [--activate]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--profile` | string | ✅ | Chemin vers le fichier JSON ou YAML du profil |
| `--name` | string | — | Surcharge le nom importé depuis le fichier |
| `--activate` | bool | — | Active automatiquement le réseau importé |

**Comportement :**
Le CLI Handler tente de parser le fichier selon deux formats :
1. **Format Fabric gateway-connection.json** (ex: `connection-profiles/connection-org1.json`) — extrait :
   - `name` → `Name`
   - `client.organization` → organisation cible
   - `organizations[org].mspid` → `MSPID`
   - premier peer de l'organisation → `PeerEndpoint` (URL sans `grpcs://`)
   - `peers[peer].grpcOptions.ssl-target-name-override` → `GatewayPeer`
   - `peers[peer].tlsCACerts.pem` → écrit dans `data/certs/<nom>/tls-root.pem`, chemin stocké dans `TLSCertPath`
   - `certificateAuthorities[ca].url` → `CAEndpoint`
   - `certificateAuthorities[ca].caName` → `CAName`
   - premier canal → `FabricChannel`
2. **Format NetworkProfile JSON** (export d'un profil myr existant) — désérialisation directe.

Les champs non extractibles du profil Fabric (CertPath, KeyPath) restent vides — l'admin les complète via `myr network update <id>`.

**Sortie :**
```
Profil importé depuis connection-org1.json
  Nom        : diy-network
  MSP ID     : Org1MSP
  Peer       : 89.167.102.193:7051
  Canal      : sandbox
  CA         : https://89.167.102.193:7054
  TLS cert   : data/certs/diy-network/tls-root.pem (extrait du profil)

Réseau importé (id: net-1700000000000).
Complétez la configuration avec : myr network update net-1700000000000 --cert <cert> --key <key>
```

**Erreurs :**
- `Erreur : fichier introuvable : <chemin>`
- `Erreur : format de profil invalide — ni Fabric connection profile ni NetworkProfile JSON reconnus`
- `Erreur : champ obligatoire manquant dans le profil : <champ>`

**Service :** CLI Handler parse le fichier, puis appelle `networkSvc.Add(...)`

---

### 3.9 `myr network destroy <id> --confirm` *(UCADM05)*

```
myr network destroy <id> --confirm [--data-path <chemin>]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--confirm` | bool | ✅ | Confirmation explicite obligatoire (RM28) |
| `--data-path` | string | — | Chemin du répertoire ledger Fabric local à supprimer (défaut: `MYR_FABRIC_DATA_PATH` ou `/var/hyperledger/production`) |

**Comportement (CLI Handler uniquement — pas dans le domaine, règle DC-D2-05) :**

1. Vérifier la présence du flag `--confirm`. Si absent → afficher l'avertissement et sortir.
2. Charger le profil via `networkSvc.List()`.
3. Vérifier `profile.IsProduction == false`. Si production → refuser et sortir.
4. **Arrêter les processus Fabric** (dépend du mode de déploiement — cf. § 3.9.1).
5. **Supprimer les données ledger locales** : `os.RemoveAll(dataPath)`.
6. **Supprimer les artefacts cryptographiques** : `os.RemoveAll("data/certs/" + profile.ID)`.
7. **Supprimer le profil** : `networkSvc.Delete(id)`.
8. Afficher confirmation.

**Sortie (--confirm absent) :**
```
ATTENTION : Cette opération est irréversible.
Elle supprimera toutes les données locales du réseau "diy-network".
  - Processus Fabric (peer, orderer, CA) arrêtés
  - Données ledger supprimées (/var/hyperledger/production)
  - Artefacts cryptographiques supprimés (data/certs/net-1700000000000/)
  - Profil réseau supprimé

Utilisez --confirm pour confirmer :
  myr network destroy net-1700000000000 --confirm
```

**Sortie (réseau de production) :**
```
Erreur : "diy-network" est marqué comme réseau de production (IsProduction: true).
Démantèlement refusé. Pour forcer, retirez le flag production :
  myr network update net-1700000000000 --production=false
```

**Sortie (avertissement processus) :**
```
Avertissement : processus peer (PID 1234) non arrêté — poursuite de la suppression.
Avertissement : processus orderer (PID 1235) non arrêté — poursuite de la suppression.
Données ledger supprimées.
Artefacts cryptographiques supprimés.
Réseau "diy-network" démantelé. Toutes les données locales ont été supprimées.
```

**Sortie (succès) :**
```
Réseau "diy-network" démantelé. Toutes les données locales ont été supprimées.
```

**Service :** `networkSvc.List()` (validation) + `networkSvc.Delete(id)` (suppression profil)
Opérations OS restent dans le CLI Handler (ENF18, DC-D2-05).

#### 3.9.1 Arrêt des processus Fabric

Le mode d'arrêt dépend du déploiement. Le CLI tente dans l'ordre :

| Méthode | Condition | Commande |
|---------|-----------|---------|
| Docker Compose | `docker ps` détecte des conteneurs Fabric | `docker compose -f <compose-file> down` |
| systemd | `/etc/systemd/system/fabric-*.service` existe | `systemctl stop fabric-peer fabric-orderer fabric-ca` |
| pkill (fallback) | Toujours | `pkill -SIGTERM peer orderer fabric-ca-server` |
| Timeout | Processus toujours actif après 10s | SIGKILL + avertissement |

Le chemin vers le fichier `docker-compose.yaml` est lu depuis `MYR_FABRIC_COMPOSE_PATH` ou `data/docker-compose.yaml`.

---

## 4. Groupe `myr org` — Gestion des organisations *(UCADM01)*

### 4.1 `myr org add`

```
myr org add --msp <mspID> --name <nom> --cert <cert.pem> [options]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--msp` | string | ✅ | MSP ID de l'organisation (`[a-zA-Z0-9_.-]{1,128}`) |
| `--name` | string | ✅ | Nom lisible de l'organisation |
| `--cert` | string | ✅ | Chemin vers le certificat CA racine PEM |
| `--channel` | string | — | ID du canal cible (défaut: canal du réseau actif) |
| `--role` | string | — | Rôle par défaut : `member` ou `admin` (défaut: `member`) |
| `--tls-cert` | string | — | Chemin vers le certificat TLS CA racine PEM |
| `--update` | bool | — | Force la mise à jour si l'organisation est déjà membre (sans prompt interactif) |

**Comportement :**
1. Le CLI Handler lit le certificat depuis `--cert` et optionnellement depuis `--tls-cert`.
2. Valide que le MSP ID respecte le format Fabric (`[a-zA-Z0-9_.\-]{1,128}`).
3. Résout le canal cible : `--channel` si fourni, sinon `FabricChannel` du réseau actif.
4. Appelle `channelSvc.AddOrganisation(channelID, org)`.

**Sortie (succès) :**
```
Organisation "Mon Organisation" (MSP: MonOrgMSP) ajoutée au canal sandbox.
La configuration du canal est mise à jour sur le réseau.
```

**Sortie (MSP déjà membre — mise à jour) :**
```
L'organisation MonOrgMSP est déjà membre du canal sandbox.
Mise à jour du rôle et de la politique d'accès...
Organisation "Mon Organisation" mise à jour sur le canal sandbox.
```

**Erreurs :**

| Code interne | Message CLI |
|-------------|------------|
| `ErrInvalidMSPID` | `Erreur : MSP ID invalide — caractères non autorisés ou longueur hors limites (max 128).` |
| `ErrFabricUnavailable` | `Erreur : adapter Fabric non configuré — vérifiez le réseau actif et les variables d'environnement Fabric.` |
| `ErrEndorsementPolicy` | `Erreur : politique d'endorsement non satisfaite. Contactez les autres administrateurs d'organisation.` |
| `file not found` | `Erreur : certificat introuvable : <chemin>` |
| `ErrAlreadyMember` (si pas de --update) | Déclenche le flux de mise à jour (interactif ou via flag `--update`) |

**Service :** `channelSvc.AddOrganisation(channelID, Organization{MSPID, Name, Role, RootCert, TLSCert})`

**Port requis :** `ChannelService.AddOrganisation()` — méthode à ajouter au port `domain/channel/port_in.go`

---

## 5. Groupe `myr node` — Gestion des nœuds réseau

### 5.1 `myr node add` *(UCADM03)*

```
myr node add --type <peer|orderer> --addr <host:port> --org <mspID> [options]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--type` | string | ✅ | Type de nœud : `peer` ou `orderer` |
| `--addr` | string | ✅ | Adresse du nœud (`host:port`, ex: `89.167.102.193:7051`) |
| `--org` | string | ✅ | MSP ID de l'organisation propriétaire du nœud |
| `--channel` | string | — | Canal cible (défaut: canal du réseau actif) |
| `--cert` | string | — | Chemin vers le certificat TLS du nœud PEM |

**Comportement :**
1. Valide le format `host:port` de `--addr`.
2. Résout le canal cible (cf. `--channel` ou réseau actif).
3. Lit le certificat TLS si `--cert` fourni.
4. Appelle `channelSvc.AddNode(channelID, nodeType, addr, orgMSP, certs)`.

**Sortie :**
```
Soumission de la demande d'ajout du peer 89.167.102.193:7051 au canal sandbox...
Nœud peer 89.167.102.193:7051 ajouté. Synchronisation du ledger en cours.
Synchronisation terminée à la hauteur 1042.
```

**Erreurs :**

| Code interne | Message CLI |
|-------------|------------|
| `ErrNodeUnreachable` | `Erreur : impossible de joindre 89.167.102.193:7051 — vérifiez le pare-feu et la disponibilité réseau.` |
| `ErrFabricUnavailable` | `Erreur : adapter Fabric non configuré.` |
| `ErrEndorsementPolicy` | `Erreur : politique d'endorsement non satisfaite. Contactez les autres administrateurs.` |
| `ErrSyncTimeout` | `Avertissement : le peer est ajouté mais la synchronisation a dépassé le délai. Vérifiez la connectivité avec : myr network test` |

**Cas orderer :** Mêmes flags. Le CLI Handler communique que les paramètres Raft sont gérés par Fabric — seuls addr, org et cert sont requis pour l'enregistrement.

**Service :** `channelSvc.AddNode(channelID, nodeType, addr, orgMSP, certs)`

**Port requis :** `ChannelService.AddNode()` — méthode à ajouter au port `domain/channel/port_in.go`

---

### 5.2 `myr node remove` *(UCADM04)*

```
myr node remove --addr <host:port> [--channel <channelID>]
```

**Drapeaux :**

| Drapeau | Type | Requis | Description |
|---------|------|--------|-------------|
| `--addr` | string | ✅ | Adresse du nœud à retirer (`host:port`) |
| `--channel` | string | — | Canal cible (défaut: canal du réseau actif) |

**Comportement :**
1. Résout le canal cible.
2. Appelle `channelSvc.RemoveNode(channelID, addr)`.
3. Le service vérifie (côté domaine) : appartenance au canal, seuil minimum 3 nœuds (RM27).
4. Affiche le nombre de nœuds actifs restants.

**Sortie :**
```
Nœud 89.167.102.193:7051 retiré du canal sandbox.
3 nœuds actifs restants sur ce canal.
```

**Avertissement gateway peer :**
```
Avertissement : le nœud retiré était le gateway peer du réseau actif.
Reconfigurez le gateway avec :
  myr network update net-1700000000000 --gateway <nouveau-peer>
```

**Erreurs :**

| Code interne | Message CLI |
|-------------|------------|
| `ErrNodeNotMember` | `Erreur : 89.167.102.193:7051 n'est pas membre actif du canal sandbox.` |
| `ErrMinNodesRequired` | `Erreur : le réseau doit conserver au moins 3 nœuds actifs. Retrait impossible (actuellement 3 nœuds actifs). [RM27]` |
| `ErrFabricUnavailable` | `Erreur : adapter Fabric non configuré.` |
| `ErrEndorsementPolicy` | `Erreur : politique d'endorsement non satisfaite. Contactez les autres administrateurs.` |

**Service :** `channelSvc.RemoveNode(channelID, addr)`

**Port requis :** `ChannelService.RemoveNode()` — méthode à ajouter au port `domain/channel/port_in.go`

---

## 6. Ports Go requis

### 6.1 Extensions de `ChannelService` (port_in)

Les trois commandes UCADM nécessitent l'extension du port d'entrée `domain/channel/port_in.go` :

```plantuml
@startuml
skinparam classAttributeIconSize 0
interface ChannelService {
  + Get(id string) : (*Channel, error)
  + List() : ([]*Channel, error)
  ..À ajouter..
  + AddOrganisation(channelID string, org Organization) : error
  + AddNode(channelID string, nodeType NodeType, addr, orgMSP string, certs NodeCerts) : error
  + RemoveNode(channelID, addr string) : error
}
@enduml
```

### 6.2 Nouvelles entités dans `domain/channel/entity.go`

```plantuml
@startuml
skinparam classAttributeIconSize 0

class Organization {
  + MSPID : string <<[a-zA-Z0-9_.-]{1,128}>>
  + Name : string
  + Role : string <<member|admin>>
  + RootCert : string <<PEM>>
  + TLSCert : string <<PEM>>
}

class NodeCerts {
  + TLSCert : string <<PEM>>
}

enum NodeType {
  peer
  orderer
}
@enduml
```

### 6.3 Extension de `ChannelConfigPort` (port_out)

Nouveau port sortant `domain/channel/ports.go` implémenté par `adapters/out/fabric/` :

```plantuml
@startuml
skinparam classAttributeIconSize 0
interface ChannelConfigPort {
  + AddOrganisation(channelID string, org Organization) : error
  + AddNode(channelID string, nodeType NodeType, addr, orgMSP string, certs NodeCerts) : error
  + RemoveNode(channelID, addr string) : error
}
note right : Si nil → service retourne ErrFabricUnavailable\nPermet de tester le CLI sans adapter Fabric
@enduml
```

### 6.4 Champ à ajouter à `NetworkProfile` (`domain/network/entity.go`)

| Champ | Type | Défaut | Description |
|-------|------|--------|-------------|
| `IsProduction` | `bool` | `false` | Protège contre `myr network destroy`. `true` → refuse le démantèlement. |

---

## 7. Format de sortie et codes de retour

### 7.1 Codes de sortie

| Code | Signification |
|------|--------------|
| `0` | Succès |
| `1` | Erreur (cobra retourne 1 par défaut — sortie sur stderr) |

### 7.2 Conventions de sortie

- **Sortie nominale** : stdout — texte lisible, formaté
- **Erreurs** : stderr via `cobra.Command.RunE` qui retourne une `error`
- **Avertissements** : stdout, prefixés `Avertissement :`
- **Tableaux** : colonnes alignées avec tabwriter (`%-20s  %s\n`)
- **Séparateurs** : `─` (U+2500) × 60 pour les sections détaillées
- **Unités** : dates en local format `2006-01-02 15:04:05`, durées en `Xh Xm Xs`

### 7.3 Mode machine-parsable (futur)

Réservé pour v2 : flag global `--output json` pour les intégrations CI/CD.

---

## 8. Résolution du canal cible

Plusieurs commandes acceptent un `--channel` optionnel. La règle de résolution est :

```
1. Si --channel fourni → utiliser cette valeur
2. Sinon → charger le réseau actif via networkSvc.GetActive()
           → utiliser profile.FabricChannel
3. Si FabricChannel vide → erreur : "canal non configuré — utilisez --channel ou myr network update --channel <id>"
```

Cette résolution est mutualisée dans une fonction `resolveChannel(cmd, networkSvc)` dans `adapters/in/cli/`.

---

## 9. Injection de services

```plantuml
@startuml
participant "cmd/cli/main.go" as Main
participant "adapters/in/cli/root.go" as Root
participant "domain/channel/service.go" as ChanSvc
participant "domain/network/service.go" as NetSvc

Main -> NetSvc : network.NewService(store, tester).WithProvisioner(fabricProvisioner)
Main -> ChanSvc : channel.NewService(repo).WithFabricConfig(fabricConfigAdapter)
Main -> Root : cli.Execute(modelSvc, channelSvc, paymentSvc, networkSvc)
Root -> Root : rootCmd.AddCommand(networkCmd, orgCmd, nodeCmd, ...)
@enduml
```

La commande `myr network destroy` est la seule commande CLI qui appelle directement l'OS (processus, fichiers). Elle n'injecte pas de nouveau service — elle opère sur le système de fichiers local via les packages `os` et `os/exec` directement dans le CLI Handler.

---

## 10. Décisions de conception

| ID | Décision | Raison |
|----|---------|--------|
| DC-CLI-01 | Résolution du canal par défaut via le réseau actif | Évite à l'admin de répéter `--channel <id>` sur chaque commande — le réseau actif fournit le contexte implicite. |
| DC-CLI-02 | `myr network destroy` opère en CLI Handler, pas dans le domaine | Les opérations OS (pkill, os.RemoveAll) violent ENF18 si placées dans `domain/network/`. Conforme à DC-D2-05. |
| DC-CLI-03 | `ChannelConfigPort` nullable — service retourne `ErrFabricUnavailable` si nil | Permet d'utiliser le CLI sans adapter Fabric (mode dev, simulation). La commande existe et valide ses flags même sans Fabric. |
| DC-CLI-04 | `--confirm` obligatoire pour `destroy`, sans alternative interactive | Cohérence avec les outils d'admin Unix standard. Le `--confirm` est scriptable, le prompt interactif ne l'est pas. |
| DC-CLI-05 | `myr network update` fusionne côté CLI Handler | `networkSvc.Update()` remplace tous les champs. La fusion (`Changed("flag")`) doit être faite dans le CLI Handler pour ne modifier que les champs fournis. |
| DC-CLI-06 | `myr network import` supporte deux formats | Le format Fabric gateway-connection.json est le format naturel des réseau Fabric. Le format NetworkProfile JSON facilite la portabilité entre instances Myr. |
| DC-CLI-07 | Arrêt processus Fabric dans `destroy` : Docker > systemd > pkill | Ordre de priorité adapté aux déploiements réels : Docker Compose (dev/test), systemd (prod), pkill (fallback). |

---

## 11. Écarts code → specs

| ID | Écart | Fichier | Impact |
|----|-------|---------|--------|
| E-CLI-01 | `myr network` (list/show/add/update/activate/delete/test/import/destroy) absent | `adapters/in/cli/network.go` (à créer) | CLI non exploitable pour la gestion réseau |
| E-CLI-02 | `myr org add` absent | `adapters/in/cli/org.go` (à créer) | UCADM01 non exposé |
| E-CLI-03 | `myr node add` et `myr node remove` absents | `adapters/in/cli/node.go` (à créer) | UCADM03 et UCADM04 non exposés |
| E-CLI-04 | `ChannelService.AddOrganisation()`, `AddNode()`, `RemoveNode()` absents | `domain/channel/port_in.go`, `service.go` | Ports d'entrée UCADM01/03/04 non définis |
| E-CLI-05 | `ChannelConfigPort` absent | `domain/channel/ports.go` | Port sortant UCADM non défini |
| E-CLI-06 | `Organization`, `NodeType`, `NodeCerts` absents | `domain/channel/entity.go` | Entités UCADM01/03/04 non définies |
| E-CLI-07 | `IsProduction bool` absent de `NetworkProfile` | `domain/network/entity.go` | Protection UCADM05 non disponible |
| E-CLI-08 | `networkCmd`, `orgCmd`, `nodeCmd` non enregistrés dans `root.go` | `adapters/in/cli/root.go` | Commandes non accessibles via `myr` |
| E-CLI-09 | ~~UCDEV02 Analyse ne mentionne pas les commandes UCADM~~ — **résolu** | `specs/2-Analyse/UCDEV-Developpement/UCDEV02.md` | Corrigé : UCDEV02 inclut maintenant les commandes UCADM |

---

## 12. Informations manquantes / points ouverts

| # | Question | Impact |
|---|---------|--------|
| 1 | **Format de sortie des nœuds actifs** dans `RemoveNode` — l'API Fabric Gateway v2 expose-t-elle un compteur ou faut-il interroger la config du canal ? | Détermine si le message `<N> nœuds actifs restants` est calculé côté service ou affichage estimé |
| 2 | **Politique d'endorsement multi-admin** — pour UCADM01/03/04, la configuration du canal Fabric requiert la signature de la majorité des admins d'organisation. Quel workflow Myr propose-t-il pour collecter ces signatures ? | Workflow multi-admin non modélisé en v1 |
| 3 | **Chemin ledger configurable** pour `myr network destroy` — `MYR_FABRIC_DATA_PATH` ou dans `config/` ? | Nécessaire pour UCADM05 fonctionnel |
| 4 | **Mode déploiement** de l'infrastructure Fabric pour `destroy` — Docker Compose ou bare-metal ? Quelle est la cible principale de déploiement ? | Détermine l'ordre de priorité de la logique d'arrêt processus (DC-CLI-07) |
| 5 | **`myr network create`** (UCADM02 flux nominal — création réseau Fabric from scratch) — reporté post-v1. En v1, `myr network import` depuis un profil `gateway-connection.json` existant couvre le cas nominal. La création from scratch (configtx.yaml, genesis block, cryptogen) est documentée manuellement hors CLI. Visible dans l'arbre avec tag `[POST-V1]`. | Résolu — reporté explicitement. |
