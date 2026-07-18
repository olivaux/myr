# Analyse des besoins — Myr System

- [Analyse des besoins — Myr System](#analyse-des-besoins--myr-system)
  - [1. Objet de l'analyse](#1-objet-de-lanalyse)
  - [2. Problème à résoudre](#2-problème-à-résoudre)
    - [2.1 Constat](#21-constat)
    - [2.2 Problèmes spécifiques](#22-problèmes-spécifiques)
    - [2.3 Réponse attendue](#23-réponse-attendue)
  - [3. Analyse des acteurs](#3-analyse-des-acteurs)
    - [3.1 Acteurs × objectifs](#31-acteurs--objectifs)
    - [3.2 Acteurs × domaines fonctionnels](#32-acteurs--domaines-fonctionnels)
  - [4. Frontières du système](#4-frontières-du-système)
    - [Ce que Myr fait](#ce-que-myr-fait)
    - [Ce que Myr ne fait pas](#ce-que-myr-ne-fait-pas)
    - [Diagramme de contexte système](#diagramme-de-contexte-système)
  - [5. Analyse fonctionnelle par domaine](#5-analyse-fonctionnelle-par-domaine)
    - [D1 — Compte \& Accès](#d1--compte--accès)
    - [D2 — Administration réseau](#d2--administration-réseau)
    - [D3 — Composant (écriture)](#d3--composant-écriture)
    - [D4 — Composant (lecture)](#d4--composant-lecture)
    - [D5 — Composition de Module (instances et liaisons)](#d5--composition-de-module-instances-et-liaisons)
    - [D6 — Module](#d6--module)
    - [D7 — Propriété Intellectuelle](#d7--propriété-intellectuelle)
    - [D8 — Recherche](#d8--recherche)
    - [D9 — Automatisation](#d9--automatisation)
    - [D13 — Développement autour de Myr](#d13--développement-autour-de-myr)
  - [6. Contraintes structurantes](#6-contraintes-structurantes)
    - [6.1 Architecture hexagonale](#61-architecture-hexagonale)
    - [6.2 Blockchain et immuabilité](#62-blockchain-et-immuabilité)
    - [6.3 Modèle de données unifié](#63-modèle-de-données-unifié)
    - [6.4 Interfaces physiques](#64-interfaces-physiques)
  - [7. Hiérarchisation des domaines](#7-hiérarchisation-des-domaines)
    - [Matrice MoSCoW](#matrice-moscow)
    - [Ordre d'implémentation recommandé](#ordre-dimplémentation-recommandé)
  - [8. Correspondance spec ↔ code existant](#8-correspondance-spec--code-existant)
    - [Domaines avec service domaine ET endpoints exposés](#domaines-avec-service-domaine-et-endpoints-exposés)
    - [Domaines avec service domaine mais sans endpoint](#domaines-avec-service-domaine-mais-sans-endpoint)
    - [Écarts structurels connus (à corriger dans le code, pas dans les specs)](#écarts-structurels-connus-à-corriger-dans-le-code-pas-dans-les-specs)
  - [9. Guide de rédaction des use cases](#9-guide-de-rédaction-des-use-cases)
    - [9.1 Termes canoniques](#91-termes-canoniques)
    - [9.2 Structure minimale d'un use case](#92-structure-minimale-dun-use-case)
    - [9.3 Points de vigilance transversaux](#93-points-de-vigilance-transversaux)
    - [9.4 Ordre de rédaction recommandé](#94-ordre-de-rédaction-recommandé)

---

## 1. Objet de l'analyse

Ce document constitue l'introduction de la phase d'analyse du projet **Myr System**. Il est produit à partir de l'expression des besoins (`specs/1-Expression/`) selon la méthode Arrington et sert de support direct à la rédaction des use cases détaillés.

**Son rôle :**
- Reformuler le problème métier en termes d'analyse structurée
- Identifier les acteurs, leurs objectifs et les frontières du système
- Décomposer les besoins par domaine fonctionnel avec leur état d'implémentation
- Signaler les contraintes transversales à respecter dans chaque use case
- Fournir un glossaire canonique assurant la cohérence entre specs et code

---

## 2. Problème à résoudre

### 2.1 Constat

La société actuelle repose sur un modèle de production linéaire (fabriquer → vendre → jeter) qui génère gaspillage et obsolescence programmée. Malgré des efforts d'amélioration continue, la traçabilité des composants, leur compatibilité inter-systèmes et la rémunération des auteurs de conceptions ne sont pas traitées de manière systémique.

Les concepteurs de pièces (CAO, firmware, modules électroniques…) n'ont pas d'outil permettant de :
- enregistrer leur création de façon immuable et vérifiable,
- garantir la compatibilité de leurs pièces avec d'autres systèmes,
- automatiser la rémunération lors de la réutilisation ou fabrication de leurs designs.

### 2.2 Problèmes spécifiques

| # | Problème | Acteur impacté |
|---|----------|---------------|
| P1 | Absence de traçabilité immuable des créations et de leur généalogie | Concepteur, Consommateur |
| P2 | Incompatibilité non détectée entre composants lors de l'assemblage | Concepteur |
| P3 | Plagiat non contrôlé — absence de vérification d'unicité à la soumission | Réseau, Administrateur |
| P4 | Rémunération manuelle et opaque des auteurs lors de fabrication | Concepteur, Consommateur |
| P5 | Aucun mécanisme de dérivation légale contrôlée (licences) | Concepteur |
| P6 | Accès à la production (fabrication, livraison) non intégré au cycle de conception | Consommateur, Manufactureur |
| P7 | Impossibilité de tracer la réutilisation d'un composant sur plusieurs réseaux | Réseau |

### 2.3 Réponse attendue

Myr est une plateforme serveur Open-Source (AGPL 3.0) proposant :
1. Un **registre blockchain immuable** des assets (composants et modules) avec généalogie, licences et propriétaires.
2. Un **assemblage de composition** permettant d'assembler des composants en modules via des interfaces physiques vérifiées.
3. Un **système de propriété intellectuelle automatisé** — commissions, transferts, clonage inter-réseaux.
4. Une **API REST** et un **CLI d'administration** pour l'intégration et l'exploitation — ses deux seuls points d'entrée.

---

## 3. Analyse des acteurs

### 3.1 Acteurs × objectifs

| Acteur | Objectif principal | Droits clés | Rôle code |
|--------|-------------------|-------------|-----------|
| **Visiteur** | Consulter les assets publics d'un réseau | Lecture seule (publique) | aucun token |
| **Lecteur** | Explorer le réseau — rôle par défaut à la création de compte | Lecture authentifiée | `reader` |
| **Concepteur** | Créer, dériver et assembler des assets | Lecture + écriture assets | `designer` |
| **Consommateur** | Trouver, commander des composants et modules | Lecture + commande | `consumer` |
| **Manufactureur** | Produire les assets commandés | Lecture sur demande | `manufacturer` |
| **Développeur** | Intégrer Myr dans ses outils via API/CLI | Lecture + API REST/CLI | `developer` |
| **Administrateur** | Créer et gouverner un réseau | Tous droits sur le réseau | `admin` |

> **Note code :** Le code existant définit les rôles `reader`, `contributor`, `admin`. Le rôle `contributor` est un alias provisoire du rôle **Concepteur** défini dans les specs. Les rôles `consumer`, `manufacturer`, `developer` sont à créer. Le rôle `reader` est correct — il correspond au rôle **Lecteur** attribué par défaut (RM21).

### 3.2 Acteurs × domaines fonctionnels

| Domaine | Visiteur | Lecteur | Concepteur | Consommateur | Manufactureur | Développeur | Admin |
|---------|:--------:|:-------:|:----------:|:------------:|:-------------:|:-----------:|:-----:|
| D1 Compte & Accès | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| D2 Administration réseau | | | | | | | ✓ |
| D3 Composant écriture | | | ✓ | | | | |
| D4 Composant lecture | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| D5 Composition de Module | | | ✓ | | | | |
| D6 Module | | ✓ | ✓ | ✓ | | ✓ | |
| D7 Propriété Intellectuelle | | ✓ | ✓ | ✓ | ✓ | | ✓ |
| D8 Recherche | | ✓ | ✓ | ✓ | ✓ | ✓ | |
| D9 Automatisation | | | ✓ | ✓ | ✓ | ✓ | |
| D13 Dev autour de Myr | | | | | | ✓ | ✓ |

---

## 4. Frontières du système

### Ce que Myr fait

- Exposer une API REST et un CLI d'administration — ce sont ses deux seuls points d'entrée
- Enregistrer des assets (composants, modules) sur une blockchain Fabric
- Vérifier l'unicité (anti-plagiat SHA-256 + SCM) et la compatibilité de licences
- Gérer les interfaces physiques et vérifier la compatibilité d'assemblage
- Automatiser la distribution des commissions à la livraison
- Gérer les identités (wallets Fabric CA) et les sessions utilisateur

### Ce que Myr ne fait pas

- Servir une interface graphique
- Effectuer la fabrication physique (déléguée à des manufactureurs partenaires)
- Héberger un éditeur 3D natif (le stockage de fichiers CAO est délégué à un dépôt distribué 3D)
- Gérer les paiements fiat (les transactions monétaires sont hors périmètre — seules les commissions blockchain sont traitées)
- Proposer un logiciel client lourd (pas d'Electron, pas d'application mobile)

### Diagramme de contexte système

```plantuml
@startuml
title Frontières du système Myr

actor "Visiteur" as V
actor "Utilisateur\n(tout rôle)" as U
actor "Administrateur" as ADM

rectangle "Myr System" {
    rectangle "myr-api\n(serveur HTTP)" as App {
        rectangle "API REST\n/api/" as REST
        rectangle "Domaine métier\n/domain/" as Domain
    }
}

rectangle "CLI Admin\nmyr-cli" as CLI

rectangle "Dépôt GUI\n(externe, hors périmètre myr)" as GUI

rectangle "Infrastructure" {
    rectangle "Blockchain\nHyperLedger Fabric" as BC
    rectangle "Dépôt distribué 3D\n(IPFS)" as IPFS
    rectangle "Base locale\n(SQLite / JSON)" as DB
}

rectangle "Intégrations tierces" {
    rectangle "Logiciels CAO\n(plugin)" as CAO
    rectangle "Boutiques en ligne" as Shop
}

V --> GUI : navigateur
U --> GUI : navigateur
GUI ..> REST : appels HTTP
ADM --> CLI : terminal

App --> BC : adapters/out/fabric/
App --> IPFS : adapters/out/ipfs/
App --> DB : adapters/out/sqlite/\nadapters/out/localstorage/

CAO ..> REST : UCAUT03
Shop ..> REST : UCAUT02

@enduml
```

---

## 5. Analyse fonctionnelle par domaine

Pour chaque domaine, la structure est :
- **Besoin** : ce que l'acteur cherche à accomplir
- **Service attendu** : ce que le système doit fournir
- **Points d'attention** : contraintes ou risques à traiter dans les use cases
- **Code existant** : état d'implémentation dans le code actuel

---

### D1 — Compte & Accès

**Use cases** : UCA01–UCA08 | **Exigences** : EF01–EF06 | **Règles** : RM20–RM22

**Besoin :** Tout acteur doit pouvoir s'identifier sur un réseau Myr. Les droits d'accès sont contrôlés par rôle.

> Il n'existe pas de compte email/mot de passe : « s'identifier » signifie obtenir une identité enregistrée auprès de la Fabric CA (RM20), et se connecter signifie (ré-)enrôler cette identité pour obtenir un token opaque de session REST.

**Service attendu :**
- Demande d'accès avec attribution automatique du rôle **Lecteur** en cas d'auto-enregistrement réseau (RM21)
- Connexion par enrôlement CA (secret) → token opaque de session REST (`X-Myr-Token`, pas de JWT)
- Identité et connexion sont la même opération — il n'y a pas de provisionnement différé à une étape ultérieure (RM20)
- Contrôle d'accès par permission (RBAC dynamique, `domain/role`) sur chaque action (RM22)
- Changement de rôle réservé à l'administrateur (`myr identity set-role`, UCA08) — aucune auto-attribution possible

**Points d'attention :**
- Le rôle Lecteur n'est garanti par défaut qu'en cas d'auto-enregistrement (`AllowAutoRegister`) — sinon le rôle dépend de ce que l'admin configure manuellement
- Le rôle de session REST doit refléter le rôle CA réel de l'identité à chaque connexion — état de cet écart suivi dans `specs/roadmap_dev.md` § Écarts Identité & Session, E1
- UCA08 n'a pas de flux self-service : c'est une action CLI administrateur uniquement

Domaines mobilisés : `domain/identity/`, `domain/role/`, `adapters/out/localstorage/` (wallets, rôles), `adapters/in/rest/handlers_identity.go` — enrôlement CA, accès invité, RBAC dynamique. Le flux d'approbation des demandes d'accès en attente est suivi dans `specs/roadmap_dev.md` § Écarts Identité & Session, E2.

---

### D2 — Administration réseau

**Use cases** : UCADM01–UCADM07 | **Exigences** : EF07–EF09, EF57–EF58 | **Règles** : RM27–RM28, RM34–RM37

**Besoin :** Un administrateur doit pouvoir créer un réseau Fabric isolé, y ajouter des organisations, étendre ou réduire sa capacité en nœuds, démanteler un réseau de test, et gérer les rôles (définition et attribution aux organisations).

**Service attendu :**
- Création d'un réseau blockchain (channel Fabric, genesis block)
- Ajout d'une organisation membre (MSP, certificats CA)
- Ajout d'un nœud peer ou orderer au réseau existant
- Retrait administratif d'un nœud du canal — refusé si le canal passerait sous 3 nœuds actifs (RM27, UCADM04)
- Démantèlement d'un réseau dev/test — opération d'infrastructure locale hors blockchain, CLI uniquement, jamais via REST, refusée sur un réseau marqué production (RM28, UCADM05)
- Création/édition/suppression de rôles personnalisés, rôle `admin` protégé (RM34, RM36, UCADM07)
- Attribution ou retrait de rôles à une organisation, droits effectifs = union des rôles actifs (RM37, UCADM06)
- Configuration des règles d'accès du réseau (rôles auto-distribués vs validation)

**Points d'attention :**
- Ces opérations sont irréversibles sur la blockchain — validation stricte avant soumission (RM07)
- L'administrateur est le seul acteur de ce domaine
- La création de réseau est le prérequis de tout autre use case — c'est le premier use case à rédiger
- Le démantèlement (UCADM05) et la gestion des rôles (UCADM06/07) sont des opérations CLI locales, sans transaction Fabric pour la seconde

**Code existant :** `domain/channel/`, `domain/network/`, `domain/identity/`, `domain/role/` — services domaine définis. Voir `specs/3-Conception/DC_CLI_Admin.md` pour l'état d'exposition CLI/REST par commande.

---

### D3 — Composant (écriture)

**Use cases** : UCCE01–UCCE06 | **Exigences** : EF10–EF16 | **Règles** : RM01–RM05

**Besoin :** Un concepteur doit pouvoir enregistrer un composant sur la blockchain, le faire évoluer et définir ses interfaces.

**Service attendu :**
- Import d'un fichier CAO (physique) ou déclaration d'un composant numérique
- Vérification anti-plagiat obligatoire pour tout asset de catégorie `base` : SHA-256 + analyse SCM > 50% (RM01)
- Attribution d'un UUID unique et immuable (RM04)
- Catégorie d'asset obligatoire parmi les 8 types (RM02) — voir §6.3
- `ParentID` obligatoire pour tout asset non-`base` (RM05)
- Vérification de compatibilité de licence pour les assets dérivés (RM03)
- Définition des interfaces physiques d'un composant (UCCE06)

**Points d'attention :**
- L'analyse SCM (similarité structurelle > 50%) est un algorithme métier central — à traiter comme un service domaine indépendant
- Les catégories `amélioration`, `variation`, `adaptation`, `dérivation`, `extension`, `régression`, `découpage` impliquent toutes un `ParentID` — le flux nominal des UC dérivés doit le vérifier en précondition
- La catégorie `découpage` transforme un composant en module (UCAM05) — c'est la seule catégorie déclenchant ce changement d'entité

**Code existant :** `domain/model/service.go` — `AddFull()` implémenté. Vérification de licence (RM03) opérationnelle. Anti-plagiat SHA-256 calculé mais **comparaison avec les assets existants absente** (RM01 incomplet). Catégorie `découpage` absente du code (à ajouter).

---

### D4 — Composant (lecture)

**Use cases** : UCCL01 | **Exigences** : EF17

**Besoin :** Tout acteur (y compris visiteur) doit pouvoir rechercher et consulter les assets disponibles sur un réseau.

**Service attendu :**
- Recherche par filtre (nom, catégorie, tags, licence, auteur…)
- Consultation sans authentification pour les assets publics (Visiteur — UCCL01)
- Consultation avec authentification pour les assets protégés

**Points d'attention :**
- L'accès public (sans JWT) est une exigence explicite — la route `GET /api/components` ne doit pas systématiquement exiger un token
- La distinction public/protégé est définie par la configuration du réseau (paramètre admin)

**Code existant :** `adapters/in/rest/handlers.go` — `GET /api/components` implémenté mais requiert `requireAuth`. Accès visiteur à implémenter.

---

### D5 — Composition de Module (instances et liaisons)

**Use cases** : UCAM01–03, 05, 07–08 (UCAM04, UCAM06 hors périmètre) | **Exigences** : EF18–EF20, EF22–EF25 (EF21 hors périmètre) | **Règles** : RM09–RM15

**Besoin :** Un concepteur doit pouvoir ajouter des composants comme instances à un module (action directe, locale au serveur, hors blockchain), y créer des liaisons entre leurs interfaces et composer un assemblage avant soumission.

**Service attendu :**
- Ajout/retrait d'instances de composants sur un module : action brute et directe (`AddAssetToWorkspace`/`RemoveAssetFromWorkspace`), locale au serveur, hors blockchain, modifiable librement
- Gestion d'instances indépendantes lors de l'ajout d'un module déjà instancié (RM15)
- Chaque asset possède toujours au moins un slot virtuel (RM13)
- Création de liaisons entre interfaces : vérification de compatibilité automatique (RM10) selon 5 critères (RM11)
- Interface à usage unique par liaison (RM09)
- Liaison devenant incompatible → `Incompatible: true` sans suppression automatique (RM12)
- Retrait d'une instance → suppression en cascade de toutes ses connexions (RM14)
- Choix d'un asset d'accroche (Fastener) pour une liaison indirecte (UCAM07)

**Points d'attention :**
- Les liaisons incompatibles (`Incompatible: true`) restent consultables jusqu'à suppression manuelle — aucune suppression automatique
- La vérification RM11 porte sur 5 critères : catégorie + **tag** + type + sens complémentaires + plages de valeurs — le champ `Tag` est actuellement absent de `AssetInterface` dans le code (à ajouter)
- Le CLI/API n'expose aucun espace navigable : chaque action (ajout/retrait d'instance, création de liaison) s'exécute directement sur un module identifié. Les méthodes Go historiques (`AddAssetToWorkspace`) portent encore le nom « Workspace » en interne, mais ce nom ne doit apparaître ni dans les use cases ni dans le nommage des commandes CLI

**Code existant :** `domain/model/service.go` — liaisons, slots virtuels, cascade opérationnels. `ifacesCompatible()` implémentée (critère tag manquant). `adapters/in/rest/handlers.go` — endpoints d'instance/liaison exposés.

---

### D6 — Module

**Use cases** : UCMOD01–04, 06 (UCMOD05 hors périmètre) | **Exigences** : EF26–EF29 | **Règles** : RM16–RM19

**Besoin :** Un concepteur doit pouvoir consolider un assemblage de composants en module, le versionner et le soumettre à la blockchain.

**Service attendu :**
- Création d'un module en état **draft** (RM16) — non visible sur le réseau
- Ajout d'un module existant comme instance d'un module hôte (instances indépendantes — RM15)
- Soumission à la blockchain : exige au moins une liaison (RM17)
- `ModuleVersion` immuable créée à la soumission (hash + horodatage — RM18)
- Toute modification d'un module **soumis** crée une nouvelle version (fork — RM19)
- Visualisation de la composition (composants + liaisons)

**Points d'attention :**
- RM19 est critique : un module soumis doit être en lecture seule — le service doit interdire toute modification directe et proposer le fork
- La `ModuleVersion` est le seul artefact ancré sur la blockchain pour les modules — les instances restent côté serveur

**Code existant :** `domain/model/service.go` — `CreateModule()`, `SubmitModule()` opérationnels. RM17 (assemblage requis) vérifiée. RM18 (ModuleVersion) implémentée. **RM19 (fork obligatoire) absente** — un module soumis reste modifiable dans le code actuel.

---

### D7 — Propriété Intellectuelle

**Use cases** : UCPI01–11 (sauf UCPI03 → reclassifié, voir stub UCPI03) | **Exigences** : EF30–EF38, EF59 | **Règles** : RM23–RM26, RM29–RM33

**Besoin :** Le système doit automatiser la rémunération des auteurs, gérer la tarification et la devise des assets, permettre le transfert de propriété et tracer le clonage inter-réseaux.

**Service attendu :**
- Commande d'un module (fabrication ou achat stock) — UCPI01
- Distribution automatique des commissions à la livraison, calculée par smart contract (RM23, RM24)
- Taux de commission uniforme défini par l'administrateur du réseau, non modifiable par l'auteur (RM29)
- Définition d'un prix sur un composant ou module propriétaire (UCPI04, UCPI05)
- Calcul automatique du prix d'un module non tarifé = somme des prix de ses composants (RM30)
- Modification du prix d'un asset, applicable aux commandes futures uniquement (RM31, UCPI11)
- Asset à prix nul librement disponible, sans commission générée (RM32)
- Devise unique par réseau, définie par l'administrateur, sans conversion (RM33)
- Signalement d'un composant similaire (UCPI06)
- Transfert définitif de propriété — immuable (RM25)
- Clonage inter-réseaux avec UUID et traçabilité préservés sur les deux réseaux (RM26)
- Respect d'une norme d'écoconception (UCPI10)

**Points d'attention :**
- La distribution de commissions est déclenchée par la **livraison** (UCAUT01) — ce domaine est couplé à D9
- Le smart contract de commission est distinct du service `payment` côté serveur — il réside dans le chaincode Fabric
- RM26 exige une transaction sur **deux réseaux distincts** — le use case doit préciser la séquence
- UCPI11 (modification de prix) est un ajout récent, absent du diagramme de contexte et de la table `Package` de `Expression_des_besoins_Intro.md` §4

**Code existant :** `domain/payment/service.go` — paiement manuel implémenté. Aucun endpoint REST exposé. Smart contract de commission **absent** du chaincode. Priorité implémentation : HAUTE.

---

### D8 — Recherche

**Use cases** : UCREC01–05 | **Exigences** : EF39–EF43

**Besoin :** Tout utilisateur authentifié doit pouvoir trouver des assets par critères et explorer leurs relations.

**Service attendu :**
- Recherche par référence, filtre multi-critères (UCREC01)
- Identification des composants dont les interfaces libres sont compatibles avec celles d'un composant source (UCREC02)
- Consultation de l'arbre de versions d'un composant (UCREC03)
- Identification des modules intégrant un composant donné (UCREC04)
- Export de la BOM (Bill of Materials) d'un module (UCREC05)

**Points d'attention :**
- UCREC02 est une recherche **approximative** : elle liste les composants candidats et compare leurs interfaces sans appliquer l'algorithme de compatibilité `ifacesCompatible` (RM10/RM11) — à ne pas confondre avec la vérification automatique déclenchée à la création d'une liaison (UCAM01)
- L'export BOM (UCREC05) doit respecter ENF03 (≤ 5 s pour 100 composants)

**Code existant :** `GET /api/components` implémenté (filtrage de base). UCREC02, UCREC04, UCREC05 non exposés.

---

### D9 — Automatisation

**Use cases** : UCAUT01–UCAUT04 | **Exigences** : EF44–EF47

**Besoin :** Le système doit permettre de déclencher la fabrication/livraison automatiquement et d'intégrer des outils externes (CAO, SCM).

**Service attendu :**
- Fabrication et livraison automatique d'un composant commandé (UCAUT01) — déclenche les commissions (RM23)
- Commande en ligne via boutique intégrée (UCAUT02)
- Import de modèle 3D depuis un logiciel CAO (plugin navigateur — UCAUT03)
- Gestion des versions SCM d'un modèle 3D (UCAUT04)

**Points d'attention :**
- UCAUT01 est le déclencheur des commissions (RM23) — la séquence livraison → commission doit être atomique
- UCAUT03 et UCAUT04 impliquent des intégrations tierces (logiciels CAO) — prévoir des interfaces bien délimitées

**Code existant :** Non implémenté.

---

### D13 — Développement autour de Myr

**Use cases** : UCDEV01–UCDEV02 | **Exigences** : EF55–EF56

**Besoin :** Les développeurs tiers et l'administrateur doivent pouvoir exploiter Myr via des interfaces programmatiques.

**Service attendu :**
- API REST documentée et stable pour les intégrations tierces (boutiques, éditeurs CAO, plugins)
- CLI d'administration (`myr-cli`) pour les opérations serveur (réseau, organisation, identités)

**Code existant :** `adapters/in/rest/` — API REST opérationnelle. `adapters/in/cli/` + `cmd/cli/` — CLI cobra en place. `cmd/mangen/` — génération de man pages.

---

## 6. Contraintes structurantes

Ces contraintes s'appliquent à **tous** les use cases. Chaque UC doit les respecter sans les répéter systématiquement.

### 6.1 Architecture hexagonale

Le domaine (`domain/`) ne connaît que des interfaces Go. Aucune dépendance à Fabric, Redis, SQLite ou IPFS dans le domaine. Cette règle est non négociable (ENF18) et vérifiée par script CI (`scripts/ci/check-domain-imports.sh`).

Flux obligatoire : `adapter in (REST/CLI)` → `service domaine` → `adapter out (Fabric/IPFS/SQLite)`.

### 6.2 Blockchain et immuabilité

- Toute soumission blockchain est **définitive** — validation complète côté serveur avant toute transaction (RM07)
- Il n'existe **aucune opération de suppression** — `ErrNotSupported` est retourné (RM08)
- En cas d'échec blockchain, l'état local (draft) est conservé intact (ENF30)

Ces contraintes impactent directement les flux d'erreur de tous les UC d'écriture (UCCE, UCMOD, UCPI).

### 6.3 Modèle de données unifié

L'entité centrale du système est `Model3D` — elle représente à la fois un composant et un module. La distinction se fait par le contenu :

| Critère | Composant | Module |
|---------|-----------|--------|
| `Hash` | renseigné (fichier CAO) | vide |
| `WorkspaceInstances` | vide | renseigné |
| `Status` | vide | `draft` ou `submitted` |

**Les 8 catégories d'asset (RM02) :**

| Code | Terme métier | ParentID requis | Déclencheur |
|------|-------------|:---------------:|-------------|
| `base` | Base | non | Nouvelle création |
| `amelioration` | Amélioration | oui | Même fonctionnalité, renforcée |
| `variation` | Variation | oui | Même fonctionnalité, différente |
| `adaptation` | Adaptation | oui | Même fonctionnalité, nouvelles dimensions |
| `derivation` | Dérivation | oui | Ajout de fonctionnalité |
| `extension` | Extension | oui | Pièce complémentaire |
| `regression` | Régression | oui | Retrait de fonctionnalité |
| `decoupage` | Découpage | oui | Redécoupage en sous-systèmes → module |

> La catégorie `decoupage` est définie dans les specs mais absente du code — à ajouter dans `domain/model/entity.go`.

### 6.4 Interfaces physiques

Une `AssetInterface` est définie par 6 attributs (RM11) :

| Champ | Description | Exemple |
|-------|-------------|---------|
| `Category` | Famille physique | `ELEC`, `MECA`, `HYD` |
| `Tag` | Type de connecteur | `Câble`, `Vis`, `Connecteur` |
| `Type` | Standard précis | `USB-C`, `Vis M3`, `BSP 1/4"` |
| `Direction` | Sens du flux | `in`, `out`, `bidir` |
| `ValueMin/Max` | Plage de valeurs | `5.0` / `5.0` (fixe) ou `3.3` / `5.0` (plage) |
| `Unit` | Unité associée | `V`, `mm`, `bar` |

> Le champ `Tag` est défini dans les specs (RM11) mais absent de la struct `AssetInterface` dans le code — à ajouter.

Deux interfaces sont **compatibles** si et seulement si les 5 critères de RM11 sont satisfaits : même catégorie + même tag (si renseigné) + même type (si renseigné) + sens complémentaires + plages de valeurs se chevauchant.

---

## 7. Hiérarchisation des domaines

### Matrice MoSCoW

| Priorité | Domaines | Justification |
|----------|---------|--------------|
| **Must have** | D1 Compte & Accès, D3 Composant écriture, D4 Composant lecture, D5 Composition de Module, D6 Module | Cœur fonctionnel — sans ces domaines, le système ne peut pas fonctionner |
| **Should have** | D2 Administration réseau, D7 Propriété Intellectuelle, D8 Recherche, D13 Dev autour de Myr | Nécessaires pour une mise en production réelle |
| **Could have** | D9 Automatisation | Améliorent l'expérience mais non bloquants |
| **Won't have (v1)** | Intégrations CAO avancées, paiements fiat, app mobile | Hors périmètre v1 |

### Ordre d'implémentation recommandé

```
D2 (réseau) → D1 (auth) → D4 (lecture) → D3 (écriture) → D5 (composition) → D6 (module) → D8 (recherche) → D7 (PI) → D9 (automatisation)
```

---

## 8. Correspondance spec ↔ code existant

### Domaines avec service domaine ET endpoints exposés

| Domaine | Service | REST | CLI |
|---------|:-------:|:----:|:---:|
| model (D3/D4/D5/D6) | ✅ | ✅ | ✅ |
| auth (D1) | 🔶 partiel | ✅ | — |
| channel (D2 partiel) | ✅ | ❌ | ✅ |
| payment (D7 partiel) | ✅ | ❌ | ✅ |

### Domaines avec service domaine mais sans endpoint

| Domaine | Service | À exposer |
|---------|:-------:|-----------|
| identity | ✅ | REST + CLI |
| network | ✅ | REST + CLI |
| session | ✅ | REST |

### Écarts structurels connus (à corriger dans le code, pas dans les specs)

| ID | Écart | Fichier | RM/ENF |
|----|-------|---------|--------|
| E1 | Catégorie `decoupage` absente | `domain/model/entity.go` | RM02 |
| E2 | `AssetInterface.Tag` absent | `domain/model/entity.go` | RM11 |
| E3 | Rôle de session REST figé à `contributor`, jamais lu depuis le rôle CA réel | `adapters/in/rest/handlers_identity.go` (`handleIdentitySession`) | RM22 |
| E4 | Anti-plagiat RM01 : comparaison avec assets existants absente | `domain/model/service.go` | RM01 |
| E5 | Fork module soumis non contraint | `domain/model/service.go:461` | RM19 |
| E6 | Entité chaincode `Model3D` incomplète (7 vs 20+ champs) | `chaincode/model/entity.go` | RM06 |

Ces écarts sont documentés ici à titre de référence — les use cases sont rédigés selon les specs (comportement cible), pas selon le code actuel.

---

## 9. Guide de rédaction des use cases

### 9.1 Termes canoniques

| Terme specs | Terme code | À utiliser dans les UC |
|-------------|-----------|------------------------|
| Asset | `Model3D` | **Asset** |
| Composant | `Model3D` (sans WorkspaceInstances) | **Composant** |
| Module | `Model3D` (avec WorkspaceInstances) | **Module** |
| Instance (d'un composant/module dans un module hôte) | `WorkspaceInstance` (nom interne, ne pas exposer) | **Instance** |
| Interface physique | `AssetInterface` | **Interface** |
| Liaison | `Connection` | **Liaison** |
| Asset d'accroche | `FastenerAssetID` | **Asset d'accroche** |
| Réseau | `Network` | **Réseau** |
| Canal | `Channel` (Fabric) | **Canal** |
| Wallet | `WalletEntry` (fichiers MSP locaux, non chiffrés) | **Wallet** |
| Rôle Lecteur | `reader` | **Lecteur** |
| Rôle Concepteur | `contributor` (provisoire) | **Concepteur** |

### 9.2 Structure minimale d'un use case

Chaque use case doit contenir :
1. **Frontmatter YAML** : catégorie, titre, probabilité (1–5), impact (1–5), importance (= prob × impact), état
2. **Diagramme d'acteurs** (PlantUML)
3. **Contexte** : pourquoi ce use case, quel problème résout-il
4. **Préconditions** : état du système et de l'acteur requis
5. **Scénario** : flux nominal + flux alternatifs + flux d'erreur
6. **Post-conditions** : état garanti après exécution réussie
7. **Diagramme d'activités** (PlantUML)
8. **Règles métier déclenchées** (références RM)

### 9.3 Points de vigilance transversaux

À vérifier dans **tout** use case d'écriture (UCCE, UCMOD, UCPI) :

- [ ] La validation est-elle côté serveur avant toute soumission blockchain ? (RM07)
- [ ] Un flux d'erreur blockchain (échec endorsement) est-il prévu ? (ENF30)
- [ ] Le rôle de l'acteur est-il vérifié côté serveur ? (ENF12)
- [ ] L'UUID est-il généré par le système, pas par le client ? (RM04)

À vérifier dans tout use case de **composition de module** (UCAM) :

- [ ] La vérification de compatibilité RM11 est-elle appelée ? (5 critères dont Tag)
- [ ] Un slot virtuel est-il garanti après toute matérialisation ? (RM13)
- [ ] La cascade de suppression est-elle décrite pour le retrait d'asset ? (RM14)

### 9.4 Ordre de rédaction recommandé

```
UCA01 → UCA02 → UCADM02 → UCADM01 → UCCE01 → UCCE06
→ UCAM01–03 → UCMOD01 → UCMOD06 → UCCL01
→ UCREC01–02 → UCA08 → UCPI01–02
→ (reste par ordre d'importance décroissante)
```

---

*Document produit à partir de `specs/1-Expression/` — méthode Arrington — version initiale.*
*Référence code : commit courant, branche `feature/project-base`.*
