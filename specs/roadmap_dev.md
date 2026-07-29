# Roadmap développement — Myr

## État de l'implémentation

| Domaine | Service | CLI | REST | État réel |
|---------|---------|-----|------|-----------|
| `model` | ✅ | ✅ | ✅ | Bugs : rôle `contributor` au lieu de `reader` (RM21), module `submitted` modifiable sans fork (RM19), anti-plagiat SHA-256 non comparatif |
| `channel` | ✅ | ✅ | ❌ | `myr channel get/list` + `myr org add`/`myr node add/remove` (AddOrganisation/AddNode/RemoveNode) ; aucune route REST dédiée |
| `identity` | ✅ | ✅ | ✅ | `myr identity wallets/status/enroll/re-enroll/request/requests/set-role` ; pas de flux d'approbation d'une demande déjà en attente (ni CLI ni REST) |
| `network` | ✅ | ✅ | partiel | `myr network/org/node` complets ; REST ne couvre que `List`/`GetActive` (pas de create/update/activate/delete/test/addPeer) |
| `session` | ✅ | ✅ | — | `myr session create/show/logout` ; pas de REST par conception (compte local machine, sans rapport avec les sessions REST) |
| `auth` | 🔶 | ❌ | partiel | REST basique, bug rôle initial |
| `payment` | 🔶 | ❌ | ❌ | Entité anémique, pas connectée |
| Chaincode | ❌ | — | — | Entity seulement (7 champs vs 20+), fonctions manquantes |
| IPFS | ❌ | — | — | Adapter vide |

---

## Écarts Identité & Session

> Référencé depuis `specs/3-Conception/DC_D1_Auth_Identity.md` (DC-D1-03, § 4 Relations et dépendances, myrSession) et `Conception_intro.md` (ADR-04, ADR-07).

### CRITIQUE — `myr-api` ne construit son client CA que depuis les variables d'environnement, jamais depuis le `NetworkProfile` actif

**Constat (vérifié en direct le 2026-07-17 sur le serveur de production, `net-1783775510080523221`) :** `myr network update <id> --auto-register` modifie correctement `networks.json` et `GET /api/identity/policy` reflète immédiatement `allow_auto_register: true` (pas de cache — `JSONNetworkStore.load()` relit le fichier à chaque appel). Pourtant `POST /api/identity/request` continue de retourner `status: "pending"` sans secret. Confirmé sur le conteneur `ca.ca-org1` (`docker logs`) : **aucune requête `/api/v1/register` n'atteint jamais la CA** — l'appel échoue avant même de partir.

**Cause :** `cmd/api/main.go:83` construit le client CA (`caPort`, utilisé par `identity.NewService(walletDir, caPort)` à la ligne 130) exclusivement via `fabricadapter.ConfigFromEnv()` — variables d'environnement `FABRIC_CA_ENDPOINT`/`FABRIC_CA_ADMIN_CERT_PATH`/`FABRIC_CA_ADMIN_KEY_PATH`/etc., ou un fichier `fabric.env` optionnel auto-détecté (`~/.Myr/data/fabric.env`). **Les champs `CAEndpoint`/`CAAdminCertPath`/`CAAdminKeyPath` du `NetworkProfile` actif — ceux que `myr network update`/`show` exposent et modifient — ne sont jamais lus pour construire ce client CA.** Pourtant `adapters/out/fabric/config.go` définit bien `ConfigFromProfile(np *NetworkProfile) Config`, et cette fonction **est** utilisée ailleurs (`cmd/cli/main.go:63,82` pour le CLI, `adapters/out/fabric/network_pool.go:89` pour le pool blockchain multi-réseau du serveur) — seul le câblage identité/CA de `cmd/api/main.go` en fait l'impasse.

**Conséquence :** Sur un serveur où aucun `fabric.env` n'existe et aucune variable `FABRIC_CA_*` n'a été exportée avant le lancement de `myr-api` (cas vérifié en production le 2026-07-17 : aucune des deux), `caPort` reste `nil` en permanence. `identitySvc.AutoRegister` échoue alors systématiquement avec `"aucun CA configuré"` ([domain/identity/service.go:210-213](../domain/identity/service.go#L210-L213)), quel que soit l'état du `NetworkProfile` — l'auto-enregistrement (UCA01) est silencieusement inopérant même quand la CLI et l'API `/identity/policy` affichent une configuration correcte. Une simple relance du service (`systemctl restart` ou équivalent) **ne corrige rien** tant que les variables d'environnement/`fabric.env` correspondantes n'ont pas été positionnées au préalable.

**Piste de correction (à valider par le PO avant implémentation) :** aligner `cmd/api/main.go` sur le pattern déjà utilisé par le CLI et le `NetworkPool` — construire la config CA via `fabricadapter.ConfigFromProfile(activeProfile)` en base, avec les variables d'environnement en surcharge optionnelle (comportement actuel conservé en repli), plutôt que l'inverse actuel (environnement uniquement, profil réseau ignoré).

### Écart connexe — échec `AutoRegister` avalé sans trace

`adapters/in/rest/handlers_identity.go:250-252` : le commentaire affirme *« logguer l'erreur »* en cas d'échec de `AutoRegister`, mais aucun appel de log n'existe réellement pour `err3` — l'échec est invisible côté serveur (pas de log) et côté client (message générique `pending`, aucun détail). Rend le diagnostic du point précédent impossible sans accès SSH direct aux conteneurs (comme fait manuellement lors de cette investigation). À corriger indépendamment de la cause racine ci-dessus : au minimum `log.Printf("auto-register CA échoué pour %s : %v", req.Pseudo, err3)`.

### Rôle de session figé à la création (`myrSession.Role`)

Voir `handlers_identity.go:375` (annotation `#question` posée sur place) et `specs/2-Analyse/UCA-Compte_et_Acces/UCA02.md` § Écart critique : le rôle de session REST est codé en dur à `"contributor"` à la connexion, jamais lu depuis l'attribut `Myr.role` du certificat CA réellement enrôlé, et jamais resynchronisé après un `myr identity set-role`.

### Registrar CA unique par organisation

Voir `Conception_intro.md` ADR-07 — un seul certificat admin CA par `NetworkProfile` signe tout enregistrement pour une organisation ; risque accepté à l'échelle de cette organisation, mitigation (rotation de clé, HSM, registrar de secours) non entreprise à ce jour.

---

## Écarts Infrastructure — Chaincode

### CRITIQUE — le chaincode `myrcc` référencé par le profil réseau de production n'a jamais été construit ni déployé

**Constat (vérifié en direct le 2026-07-29 sur le serveur de production, réseau `diy-network`, canal `sandbox`) :** `GET /api/components` et `GET /api/modules` retournent 500 côté `myr-api` ; `journalctl --user -u myr` montre `erreur interne : fabric ListModelRecords evaluate : rpc error: code = FailedPrecondition desc = no peers available to evaluate chaincode myrcc in channel sandbox` en continu (une requête par minute). Les logs du peer (`docker logs peer0.org1.diy-network.com`) confirment côté Fabric : `no peers available to evaluate chaincode myrcc in channel sandbox`, et une tentative antérieure de soumission échoue avec `No metadata was found for chaincode myrcc in channel sandbox`.

**Cause racine :** `chaincode/` ne contient à ce jour qu'un `Dockerfile` (image chaincode-as-a-service, non versionné — voir `.gitignore`) qui attend un `go build` sur du code source absent (aucun fichier `.go`, aucun `chaincode/model/`) — voir `install.md` §7, qui documente déjà ce point : « `myr network create` provisionne l'infrastructure Fabric (peers, orderer, CA, canal) mais n'installe, n'approuve ni ne commit aucun chaincode » et « aucune implémentation Go n'est encore présente dans ce dépôt pour être installée telle quelle ». Le profil réseau a malgré tout été mis à jour avec `chaincode_name=myrcc` (`myr network update <id> --contract myrcc`, décrit comme l'étape à faire *après* un déploiement chaincode réel) sans qu'aucun cycle de vie Fabric (`peer lifecycle chaincode install` / `approveformyorg` / `commit`) n'ait jamais été exécuté sur le peer pour ce nom. Recherche exhaustive sur le serveur (`docker ps -a`, `docker images`, arborescence `/home/fabricadmin`, unités systemd, `fabric.env`) : aucune trace d'un chaincode `myrcc` packagé ou installé — cohérent avec une implémentation qui n'a jamais existé, plutôt qu'un artefact supprimé après coup.

**Conséquence :** toute opération passant par la blockchain échoue avec une erreur Fabric brute (500, pas 503 — `internalErr` ne classe cette erreur `FailedPrecondition` ni comme `ErrBlockchainUnavailable` ni comme un cas géré, voir [handlers.go:1893](../adapters/in/rest/handlers.go#L1893)) : listing des composants/modules, soumission, toute lecture/écriture sur le canal `sandbox`. Aucune relance du service `myr-api` ne corrige ce point : le problème est entièrement situé dans l'infrastructure Fabric (chaincode manquant), pas dans le processus `myr-api`.

**Piste de correction (à valider par le PO avant implémentation) :** implémenter le chaincode Go `myrcc` (voir `specs/3-Conception/Chaincode.md` pour le modèle de données attendu), le construire via `chaincode/Dockerfile`, puis exécuter le cycle de vie complet Fabric (package → install sur le(s) peer(s) → approve pour Org1MSP → commit sur le canal `sandbox`) avant de faire pointer `chaincode_name` du profil réseau vers ce nom — dans cet ordre, pas l'inverse.

---

## Alpha — "Créer, assembler, soumettre"

> **Objectif :** Un administrateur peut démarrer un réseau. Un concepteur peut créer des composants, les assembler dans l'atelier et soumettre un module sur la blockchain.

> **Bilan Arrington ✅ — Specs complètes sur les 3 phases.** Expression, Analyse et Conception couvrent tous les UC de cette phase. Développement peut démarrer immédiatement.

### Bugs bloquants — à corriger avant tout

- [ ] `domain/auth/service.go` — rôle initial `reader` au lieu de `contributor` *(RM21 — faille sécurité)*
- [ ] `domain/model/service.go` — garde `SubmitModule` : module `submitted` non modifiable sans fork *(RM19)*
- [ ] `chaincode/model/entity.go` — aligner les champs (7 → 20+) avec le domaine *(perte de données silencieuse)*

### CLI Admin — UCADM / UCDEV02

> Specs dans [specs/3-Conception/DC_CLI_Admin.md](3-Conception/DC_CLI_Admin.md)

- [ ] `myr network list / show / add / update / activate / delete / test / import` — gestion des profils réseau
- [ ] `myr org add` — ajouter une organisation au canal Fabric *(UCADM01)*
- [ ] `myr node add / remove` — ajouter/retirer un peer ou orderer *(UCADM03, UCADM04)*
- [ ] Extensions domaine requises : `ChannelService.AddOrganisation()`, `AddNode()`, `RemoveNode()`, entités `Organization` / `NodeType` / `NodeCerts`, champ `IsProduction` sur `NetworkProfile`

### Compte & Accès — UCA01–08

- [ ] UCA01 — Création de compte (email + password + organisation → rôle Lecteur automatique)
- [ ] UCA02 — Connexion JWT ; provisionnement wallet Fabric X.509 à la première connexion *(RM20)*
- [ ] UCA03 — Déconnexion (invalidation session)
- [ ] UCA04 — Vérification de session active
- [ ] UCA05 — Contrôle d'accès par rôle
- [ ] UCA06 — Consultation des assets possédés
- [ ] UCA07 — Vérification de rôle
- [ ] UCA08 — Demande d'un rôle supplémentaire depuis le profil *(RM22)*
- [x] Exposition CLI : `identity` (`myr identity ...`), `network` (`myr network/org/node ...`), `session` (`myr session ...`, pas de REST par conception)
- [ ] Exposition REST : `network` reste partielle (create/update/activate/delete/test/addPeer absents) ; `channel` (org/node) sans route dédiée
- [ ] UCCL01 — Lecture publique composants sans JWT *(actuellement bloqué par auth)*

### Composants — Écriture UCCE01–06

- [ ] UCCE01 — Ajouter un composant physique (import fichier 3D, hash SHA-256, métadonnées, soumission blockchain)
- [ ] UCCE02 — Configurer un composant (éditer métadonnées)
- [ ] UCCE03 — Ajouter un composant numérique (firmware, logiciel)
- [ ] UCCE04 — Améliorer / dériver un composant (`ParentID` requis, vérification licence RM03)
- [ ] UCCE05 — Ajouter une extension à un composant
- [ ] UCCE06 — Ajouter une interface à un composant existant
- [ ] Correctif entité : catégorie `découpage` ajoutée *(RM02 — 8e type, bloque UCAM05)*
- [ ] Correctif entité : champ `Tag` dans `AssetInterface` *(5e critère de compatibilité RM11)*

### Atelier (Workspace) — UCAM01–08

- [ ] UCAM01 — Créer une liaison entre deux interfaces compatibles (grisage des interfaces incompatibles, FastenerAssetID optionnel) *(RM09–RM11)*
- [ ] UCAM02 — Visualiser les interfaces physiques d'un composant dans l'atelier
- [ ] UCAM03 — Définir une interface sur un composant dans l'atelier (slot virtuel → interface concrète) *(RM13)*
- [ ] UCAM04 — Ajouter plusieurs composants simultanément à l'atelier
- [ ] UCAM05 — Transformer un composant en module (`découpage`) *(RM02)*
- [ ] UCAM06 — Slot virtuel garanti et recréé automatiquement *(RM13)*
- [ ] UCAM07 — Choisir un asset d'accroche (fastener picker) pour une liaison
- [ ] UCAM08 — Retirer un composant de l'atelier (suppression en cascade des connexions) *(RM14)*
- [ ] Liaisons incompatibles post-modification : passer à `Incompatible: true`, affichage rouge *(RM12)*
- [ ] Seconde instance indépendante si module déjà dans l'atelier *(RM15)*

### Modules — UCMOD01–06

- [ ] UCMOD01 — Créer un module en état `draft` (≥ 1 liaison requise) *(RM16, RM17)*
- [ ] UCMOD02 — Ajouter un module existant à l'atelier (nouvelle instance indépendante)
- [ ] UCMOD03 — Lier un module via URL externe
- [ ] UCMOD04 — Visualiser la composition d'un module (composants, liaisons, interfaces libres)
- [ ] UCMOD06 — Soumettre un module à la blockchain (création `ModuleVersion` immuable horodatée, vérification licences) *(RM17, RM18, RM19)*

### Recherche — base

- [ ] UCREC01 — Rechercher un asset par référence / UUID → ajouter à l'Explorer

### Chaincode Go — fonctions minimales

- [ ] Implémenter `StoreModel`, `GetModel`, `ListModels`, `VerifyModel` dans `chaincode/`
- [ ] Aligner l'entité chaincode avec l'entité domaine (20+ champs)

### UI & Navigation

- [ ] UCIG01 — Navigation cohérente : MenuBar (Recherche, Profil, New Asset), Explorer, Asset UI, Atelier
- [ ] UCIG02 — Gestion des erreurs UI explicite (page 404, messages d'erreur typés)

---

## Beta — "IPFS, recherche avancée, publication"

> **Objectif :** Les fichiers 3D sont stockés décentralisés. La recherche est complète. L'anti-plagiat structurel fonctionne. Les premières notions tarifaires sont disponibles.

> **Bilan Arrington ⚠️ — Specs partiellement prêtes.** 4 décisions PO bloquantes non résolues (algorithme SCM anti-plagiat RM01, modèle économique UCPI01/02, répartition commissions RM24, UCAM05 nouvel UUID vs mutation). UCPI11 non encore analysé (phase 2 manquante). Documents conception `Architecture_Atelier.md`, `Architecture_PI.md` et les diagrammes de séquence soumission asset/module à produire avant d'attaquer ces UC.

### IPFS — Stockage 3D distribué

- [ ] `adapters/out/ipfs/` — upload fichier 3D → CID, download par CID, pin local
- [ ] Lier le CID au champ `Model3DIPFS` de l'asset on-chain (Fabric stocke le CID, pas le fichier)
- [ ] UCCE01 flux alternatif — import depuis formats CAO non natifs (STL, STEP, OBJ) avec extraction automatique des métadonnées géométriques

### Anti-plagiat structurel *(RM01)*

- [ ] Comparaison SHA-256 des assets existants dans `AddFull()` *(hash existant → rejet)*
- [ ] Analyse de similarité SCM > 50 % *(algorithme à confirmer — librairie Go, service externe, ou propriétaire)*

### Recherche avancée — UCREC02–05

- [ ] UCREC02 — Composants compatibles (par type d'interface, catégorie, tag, sens)
- [ ] UCREC03 — Arbre de versions d'un composant (historique de dérivation)
- [ ] UCREC04 — Modules qui intègrent un composant donné
- [ ] UCREC05 — Export BOM (Bill of Materials) d'un module

### Propriété Intellectuelle — tarification

- [ ] UCPI04 — Définir un prix sur un composant propriétaire (prix unitaire + devise du réseau)
- [ ] UCPI05 — Définir un prix sur un module propriétaire (prix manuel ou agrégation automatique composants — RM30)
- [ ] UCPI06 — Signaler un composant similaire à un existant (soumission à l'administration)
- [ ] UCPI11 — Modifier le prix d'un asset (effet futures commandes uniquement, avertissement si price=0 — RM31, RM32)
- [ ] `payment` domain : entités `Order`, `AssetPrice` (avec champ `CommissionRate` snapshot — RM33), exposition REST

### Réseau — démantèlement

- [ ] `myr network destroy <id> --confirm` *(UCADM05 — CLI uniquement, jamais REST, RM28)*

### Internationalisation

- [ ] UCPAR01 — Interface en anglais
- [ ] UCPAR02 — Interface en chinois

### Documentation intégrée

- [ ] UCDOC01 — Accéder à la documentation du système
- [ ] UCDOC02 — Consulter la FAQ
- [ ] UCDOC03 — Documentation technique compréhensible

---

## V1 — "Économie circulaire complète"

> **Objectif :** Commandes, fabrication, commissions, transferts de propriété. Tous les rôles. Production-ready.

> **Bilan Arrington ❌ — Specs insuffisantes.** Les questions PO bloquantes de la Beta doivent être résolues en premier. Les rôles `consumer`, `manufacturer`, `developer` ne sont pas encore définis côté conception technique. Le smart contract `DistributeCommissions` et les flux de clonage inter-réseaux (UCPI07–09) n'ont pas de document de conception dédié.

### Propriété Intellectuelle complète — UCPI

- [ ] UCPI01 — Commander un module complet (achat en stock ou fabrication sur demande)
- [ ] UCPI02 — Distribution automatique des commissions aux auteurs à la livraison *(RM23)*
- [ ] Répartition proportionnelle multi-auteurs *(RM24)*
- [ ] UCPI07 — Transfert de propriété intellectuelle d'un asset (définitif, immuable) *(RM25)*
- [ ] UCPI08 — Cloner un composant sur un réseau externe (UUID préservé) *(RM26)*
- [ ] UCPI09 — Cloner un module sur un réseau externe *(RM26)*
- [ ] UCPI10 — Vérification et respect d'une norme d'écoconception

### Automatisation — UCAUT

- [ ] UCAUT01 — Fabrication et livraison automatisée via manufactureur agréé (transmission CAO via blockchain, gestion commande en lot)
- [ ] UCAUT02 — Commande en ligne d'un asset (boutique partenaire ou interface MYR)
- [ ] UCAUT01 (canal `external_adapter`) — `ManufacturingPort` (out) + un adapter par partenaire industriel externe (`adapters/out/manufacturing/<partenaire>/`, ex. Sculpteo/Xometry/PCBWay), route webhook de confirmation de livraison, champ `OrderItem.FulfillmentChannel` *(`Conception_intro.md` ADR-09, `DC_D7_Payment.md` DC-D7-09)*

### Rôles complets

- [ ] Rôles `Consommateur` et `Manufactureur` : entités, accès différenciés, intégration dans auth/identity
- [ ] `ConsumerProfile` : adresse de livraison *(requis UCPI01)*
- [ ] Agrément manufactureur par l'administrateur (rôle `manufacturer` via `myr org add`) — canal `network_node` uniquement ; le canal `external_adapter` n'utilise pas ce rôle *(ADR-09)*

### Taux de commission réseau *(RM29)*

- [ ] Champ `CommissionRate float64` sur `NetworkProfile` (défaut 10 %)
- [ ] Flag `--commission-rate` sur `myr network add` et `myr network update`
- [ ] Le smart contract `DistributeCommissions` lit ce taux depuis le profil réseau actif
- [ ] Gestion wallet inactif : commission en état `sequestered` 90 jours puis redistribuée *(DC-D7-06)*

### UCMOD05

- [ ] Ajouter un module existant via plugin navigateur

### Tests

- [ ] `domain/*/tests/` — tests unitaires pour tous les domaines (UC + RM)
- [ ] `adapters/*/integration/` — tests d'intégration Fabric et IPFS
- [ ] `tests/e2e/` — tests Playwright (auth, navigation)
- [ ] `scripts/load/rest_load.js` — tests de charge k6
- [ ] `scripts/ci/check-domain-imports.sh` — vérification imports interdits dans `domain/`
- [ ] `scripts/ci/check-licenses.sh` — conformité AGPL 3.0

### Création réseau from scratch

- [ ] `myr network create` — génération `configtx.yaml`, genesis block, cryptogen *(UCADM02 flux nominal — marqué `[POST-V1]` dans DC_CLI_Admin, à décider)*

---

## Compléments — Post-V1

| Fonctionnalité | UC | Note |
|---------------|----|------|
| Intégration plugin CAO (import modèle 3D depuis CAO) | UCAUT03 | À développer par la communauté (rôle Développeur) |
| Gestion versions SCM des modèles 3D | UCAUT04 | Git-like pour fichiers 3D |
| API REST documentée pour développeurs tiers | UCDEV01 | OpenAPI/Swagger + clé API ou certificat pour le rôle Développeur |
| Synchronisation canal distant | `myr network sync` | `[POST-V1]` dans DC_CLI_Admin |
| Sessions Redis multi-instances | — | `REDIS_URL` déjà prévu en variable d'environnement |
| Output JSON pour CLI (`--output json`) | — | Mode machine-parsable, réservé v2 dans DC_CLI_Admin |
| Révocation JWT (blacklist access tokens) | — | Sécurité avancée — tokens compromis avant expiration |
| Rotation de `WALLET_ENCRYPT_KEY` | — | Mécanisme de re-chiffrement des wallets SQLite |
| Mitigation registrar CA unique par organisation | UCA01 | Aujourd'hui, une seule identité admin CA (`CAAdminCertPath`/`CAAdminKeyPath`) signe tout enregistrement pour une organisation (ADR-07, `Conception_intro.md` §6) — point de défaillance unique à l'échelle de cette organisation. Pistes à évaluer : rotation de clé, HSM, ou registrar de secours par organisation |
| `ConsumerProfile` multi-adresses | — | Livraison à plusieurs adresses |
| Taux de commission sur le canal `external_adapter` | UCAUT01, UCPI01 | Un partenaire industriel externe prélève probablement sa propre marge de fabrication en amont, hors du prix suivi par `AssetPrice` — reste à trancher si RM29/RM24 s'appliquent identiquement sur ce canal (voir `Conception_intro.md` ADR-09, point ouvert) |
| Migration SQLite → PostgreSQL | — | Si scalabilité horizontale requise |

---

---

## Exigences Non-Fonctionnelles — à satisfaire par phase

> Référence complète : [Exigences_Non_Fonctionnelles.md](1-Expression/Exigences_Non_Fonctionnelles.md)

| Phase | ENF à valider |
|-------|--------------|
| **Alpha** | ENF08 (HTTPS/TLS), ENF09 (JWT ≤24h + révocation), ENF11 (secrets absents des logs), ENF12 (contrôle rôle serveur), ENF13 (validation entrées), ENF22 (navigateurs), ENF23 (binaires Linux+Win), ENF24 (pas de logiciel client) |
| **Beta** | ENF04 (SPA ≤3s), ENF15 (fichier CAO ≤100Mo via IPFS), ENF25 (AGPL — `check-licenses.sh`), ENF26 (compat licence auto), ENF27 (RGPD — audit champs perso), ENF28 (immuabilité), ENF29 (anti-plagiat obligatoire), ENF30 (draft préservé sur échec Fabric), ENF31 (validation avant soumission) |
| **V1** | ENF01 (REST ≤500ms — k6), ENF02 (Fabric ≤30s), ENF03 (BOM ≤5s/100 composants), ENF05 (dispo ≥99%), ENF06 (blockchain ≥99,9%), ENF07 (restart ≤60s), ENF10 (AES-256 wallets), ENF14 (10k assets), ENF16 (50 users simultanés), ENF18 (isolation domaine — `check-domain-imports.sh`), ENF19 (couverture ≥80%), ENF20 (deploy ≤10min), ENF21 (adapter multi-blockchain interchangeable) |
| **Post-V1** | ENF17 (Redis multi-instances) |

---

## Ordre recommandé pour démarrer

1. **Bugs bloquants** — rôle initial, module guard, chaincode entity *(1–2 jours)*
2. **`myr network / org / node`** — specs complètes dans [DC_CLI_Admin.md](3-Conception/DC_CLI_Admin.md), implémentation directe
3. **Exposition REST identity / session / network** — débloque UCA01–08 côté frontend
4. **Chaincode fonctions** — sans ça, toute intégration Fabric est factice
5. **UCCE01–06 + UCAM01–08 + UCMOD01/06** — cycle de vie complet composant → module → blockchain
