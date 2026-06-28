# Règles métier — Myr System

Ce document centralise les règles de gestion métier du projet Myr. Chaque règle indique son déclencheur, sa condition et sa conséquence observable.

Ces règles complètent les use cases : elles régissent ce que le système DOIT faire indépendamment du scénario emprunté. Elles servent de référence pour les tests de validation.

> Les règles d'architecture (isolation hexagonale, routing adapters) restent dans `CLAUDE.md`.

---

## 1. Assets et composants

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM01 | **Anti-plagiat obligatoire** | Soumission d'un asset de type `base` | Toujours | Le système calcule le SHA-256 du fichier et effectue une analyse de similarité SCM. Si hash identique ou similarité > 50 % avec un asset existant → soumission rejetée, contact administration requis | UCCE01 |
| RM02 | **Catégorie d'asset obligatoire** | Création ou dérivation d'un asset | Toujours | Chaque asset doit appartenir à l'une des 8 catégories : `base`, `amélioration`, `variation`, `adaptation`, `dérivation`, `extension`, `régression`, `découpage` | UCCE01–06, UCAM05 |
| RM03 | **Compatibilité de licence** | Soumission d'un asset avec `ParentID != ""` et `LicenseID != ""` | Toujours | La licence de l'asset dérivé doit être compatible avec celle du parent. Si incompatible → soumission rejetée | UCCE04, UCMOD06 |
| RM04 | **UUID unique** | Enregistrement blockchain réussi | Toujours | Un UUID unique est généré et attribué à l'asset. Il est immuable et ne peut pas être modifié ultérieurement | UCCE01–06 |
| RM05 | **ParentID obligatoire pour les dérivés** | Création d'un asset non-`base` | Catégorie ≠ `base` | L'asset dérivé doit référencer un `ParentID` valide. Un asset `base` n'a pas de parent | UCCE04, UCCE05 |

---

## 2. Blockchain et immuabilité

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM06 | **Immuabilité des transactions** | Toute soumission blockchain | Toujours | Une transaction soumise ne peut pas être modifiée ou annulée. Il n'existe aucune opération de suppression sur la blockchain | UCCE01, UCMOD06, UCPI07–09 |
| RM07 | **Validation préalable obligatoire** | Avant toute soumission blockchain | Toujours | Toutes les données (métadonnées, licences, interfaces, UUID) sont validées côté serveur avant soumission. Une validation échouée ne génère aucune transaction | UCCE01, UCMOD06 |
| RM08 | **Rejet de la suppression** | Tentative de suppression d'un asset blockchain | Toujours | La réponse `ErrNotSupported` est retournée. Aucune erreur non contrôlée (`panic`) ne doit se produire | — |

---

## 3. Interfaces et liaisons

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM09 | **Interface à usage unique** | Tentative de liaison d'une interface déjà engagée | Interface source ou cible déjà utilisée dans une liaison | La liaison est refusée | UCAM01 |
| RM10 | **Vérification de compatibilité automatique** | Création de toute liaison | Toujours | Le système appelle `ifacesCompatible` pour vérifier la cohérence de la paire avant d'enregistrer la liaison. Une liaison incompatible ne peut pas être créée via l'atelier (les interfaces incompatibles sont rendues non sélectionnables). L'incompatibilité post-création est couverte par RM12 | UCAM01 |
| RM11 | **Critères de compatibilité d'interfaces** | Appel de `ifacesCompatible` | Toujours | Deux interfaces sont compatibles si : (1) même catégorie (Électrique, Mécanique, Numérique…), (2) même tag (Câble, vis…) si renseigné, (3) même type (USB-C, UART…) si renseigné, (4) sens complémentaires (♂/♀ ou bidirectionnel), (5) plages de valeurs se chevauchant (`ValueMin`/`ValueMax`) | UCAM01, UCAM03 |
| RM12 | **Persistance des liaisons incompatibles** | Liaison devenant incompatible après modification d'un asset | Toujours | La liaison n'est pas supprimée automatiquement. Elle passe à `Incompatible: true` et doit rester accessible dans l'atelier jusqu'à suppression manuelle par l'utilisateur | UCAM01 |
| RM13 | **Slot virtuel garanti** | Ajout d'un asset dans l'atelier | Toujours | Chaque asset dispose en permanence d'au moins un slot virtuel (`Virtual: true`). Dès qu'un slot virtuel est matérialisé en interface, un nouveau slot virtuel est recréé automatiquement | UCAM03, UCAM06 |

---

## 4. Atelier (Workspace)

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM14 | **Suppression en cascade des connexions** | Retrait d'un asset de l'atelier (`RemoveAssetFromWorkspace`) | Toujours | Toutes les connexions de l'instance retirée sont supprimées automatiquement. Les assets liés restent dans l'atelier mais leurs interfaces concernées redeviennent libres | UCAM08 |
| RM15 | **Instance indépendante** | Ajout d'un module déjà présent dans l'atelier | Module déjà instancié | Une seconde instance est créée avec ses propres connexions, indépendantes de la première | UCMOD02 |

---

## 5. Modules

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM16 | **État draft obligatoire** | Création d'un module dans l'atelier | Toujours | Un module nouvellement créé est en état `draft`. Il n'est pas visible sur le réseau. Il peut être modifié librement tant qu'il n'est pas soumis | UCMOD01 |
| RM17 | **Assemblage requis pour soumission** | Appel de `SubmitModule` | Toujours | Le module doit contenir au moins une liaison entre composants. Si aucune liaison → soumission rejetée avec message explicite | UCMOD06 |
| RM18 | **ModuleVersion immuable** | Soumission réussie d'un module | Toujours | Une `ModuleVersion` est créée avec un hash de l'assemblage et un horodatage. Ce snapshot est immuable. Toute modification ultérieure exige la création d'une nouvelle version | UCMOD06 |
| RM19 | **Fork de module** | Modification d'un module publié | Toujours | Un module publié ne peut pas être modifié directement. Une nouvelle version (fork) doit être créée en état `draft` | UCMOD06 |

---

## 6. Compte et accès

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM20 | **Provisionnement blockchain différé** | Première connexion d'un utilisateur | Compte existant, première connexion JWT | L'identité Fabric CA (certificat X.509) est provisionnée automatiquement par le backend. Elle n'est PAS créée à la création du compte | UCA01, UCA02 |
| RM21 | **Rôle Lecteur par défaut** | Création de compte | Toujours | Tout compte nouvellement créé reçoit automatiquement le rôle **Lecteur** (lecture seule). Aucune approbation admin n'est requise pour l'activation du compte | UCA01 |
| RM22 | **Distribution d'un rôle supplémentaire** | Demande de rôle depuis le profil | Toujours | Si le rôle cible est configuré en auto-distribution → il est attribué immédiatement. Sinon → la demande est soumise à l'administrateur, le rôle reste en attente de validation | UCA08 |

---

## 7. Propriété intellectuelle et commissions

| ID | Règle | Déclencheur | Condition | Conséquence | UC |
|----|-------|------------|-----------|-------------|-----|
| RM23 | **Distribution automatique des commissions** | Livraison d'un composant ou module commandé | Toujours | Le smart contract calcule et distribue les commissions automatiquement à chaque auteur dans la chaîne de propriété. Chaque transaction de commission est enregistrée individuellement sur la blockchain | UCPI02, UCAUT01 |
| RM24 | **Répartition proportionnelle multi-auteurs** | Module commandé avec plusieurs auteurs | Plusieurs concepteurs impliqués dans le module | Chaque auteur reçoit une commission proportionnelle à son apport. Les transactions sont indépendantes par auteur | UCPI02 |
| RM25 | **Transfert de propriété définitif** | Transfert accepté par le destinataire | Toujours | La propriété est transférée de manière immuable. L'ancien propriétaire perd immédiatement les droits d'édition | UCPI07 |
| RM26 | **Traçabilité du clonage inter-réseaux** | Clonage d'un asset sur un réseau externe | Toujours | Le clone conserve l'UUID et les métadonnées originales. La transaction de clonage est enregistrée sur les deux réseaux pour assurer la traçabilité de l'origine | UCPI08, UCPI09 |

---

## 8. Résumé — index des règles

| ID | Règle (résumé) | Domaine |
|----|---------------|---------|
| RM01 | Anti-plagiat obligatoire (SHA-256 + SCM > 50 %) | Assets |
| RM02 | Catégorie d'asset obligatoire (8 types) | Assets |
| RM03 | Compatibilité de licence pour tout asset dérivé | Assets |
| RM04 | UUID unique et immuable à l'enregistrement | Assets |
| RM05 | ParentID obligatoire pour les assets non-`base` | Assets |
| RM06 | Transactions blockchain immuables, pas de suppression | Blockchain |
| RM07 | Validation complète avant toute soumission blockchain | Blockchain |
| RM08 | ErrNotSupported retourné sur tentative de suppression | Blockchain |
| RM09 | Interface à usage unique dans une liaison | Interfaces |
| RM10 | Vérification de compatibilité automatique à la création | Interfaces |
| RM11 | Critères de compatibilité : catégorie + tag + type + sens + plage de valeurs | Interfaces |
| RM12 | Liaison incompatible persistante (rouge, Incompatible:true) | Interfaces |
| RM13 | Slot virtuel garanti et recréé automatiquement à chaque matérialisation | Interfaces |
| RM14 | Suppression en cascade des connexions à la sortie d'atelier | Atelier |
| RM15 | Seconde instance indépendante si module déjà dans l'atelier | Atelier |
| RM16 | Module en état draft à la création | Modules |
| RM17 | Au moins un assemblage requis pour soumettre | Modules |
| RM18 | ModuleVersion immuable horodatée à la soumission | Modules |
| RM19 | Toute modification d'un module publié crée une nouvelle version | Modules |
| RM20 | Provisionnement blockchain à la première connexion (pas à la création) | Compte |
| RM21 | Rôle Lecteur attribué par défaut à la création de compte | Compte |
| RM22 | Rôle supplémentaire : auto si configuré, sinon validation admin | Compte |
| RM23 | Commissions distribuées automatiquement à la livraison | PI |
| RM24 | Répartition proportionnelle par auteur | PI |
| RM25 | Transfert de propriété définitif et immuable | PI |
| RM26 | Traçabilité UUID préservée lors du clonage inter-réseaux | PI |
