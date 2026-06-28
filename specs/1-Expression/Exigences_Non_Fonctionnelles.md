# Exigences non-fonctionnelles — Myr System

Ce document structure les contraintes non-fonctionnelles du projet Myr selon les catégories ISO 25010. Pour chaque exigence, un critère d'acceptance mesurable est défini. La section 2.3 de l'introduction présente ces mêmes contraintes sous forme textuelle ; ce document en est la version formalisée et vérifiable.



## Tableau des exigences non-fonctionnelles

| ID | Catégorie | Exigence | Critère d'acceptance | Priorité |
|-|--||-||
| **Performance** | | | | |
| ENF01 | Performance | Temps de réponse des endpoints REST | ≤ 500 ms au 95e percentile, hors opérations blockchain | HAUTE |
| ENF02 | Performance | Temps de soumission d'une transaction Fabric | ≤ 30 s (endorsement + commit), réseau local | HAUTE |
| ENF03 | Performance | Génération d'une BOM module | ≤ 5 s pour un module contenant jusqu'à 100 composants | MOYENNE |
| ENF04 | Performance | Chargement initial de l'interface graphique (SPA) | ≤ 3 s sur connexion 10 Mbit/s | MOYENNE |
| **Disponibilité** | | | | |
| ENF05 | Disponibilité | Disponibilité du serveur Myr (instance unique) | ≥ 99 % sur 30 jours glissants, hors maintenance planifiée | HAUTE |
| ENF06 | Disponibilité | Disponibilité du réseau blockchain (multi-nœuds) | ≥ 99,9 % — le réseau reste opérationnel si un nœud sur trois est indisponible | HAUTE |
| ENF07 | Disponibilité | Reprise après redémarrage du serveur | Le serveur est opérationnel en ≤ 60 s après redémarrage | MOYENNE |
| **Sécurité** | | | | |
| ENF08 | Sécurité | Chiffrement des communications | Toutes les communications serveur ↔ client en HTTPS (TLS 1.2 minimum) | HAUTE |
| ENF09 | Sécurité | Durée de vie des tokens JWT | Expiration ≤ 24 h ; révocation possible côté serveur | HAUTE |
| ENF10 | Sécurité | Chiffrement des wallets Fabric | Wallets stockés chiffrés (AES-256) via `WALLET_ENCRYPT_KEY` | HAUTE |
| ENF11 | Sécurité | Secrets absents des logs | Aucune valeur de `JWT_SECRET`, `WALLET_ENCRYPT_KEY` ou certificat X.509 ne doit apparaître dans les logs applicatifs | HAUTE |
| ENF12 | Sécurité | Contrôle d'accès par rôle | Toute action est vérifiée côté serveur selon le rôle de l'utilisateur authentifié — le client ne peut pas élever ses droits | HAUTE |
| ENF13 | Sécurité | Protection contre l'injection | Les entrées utilisateur sont validées avant toute soumission blockchain ou requête base de données | HAUTE |
| **Scalabilité** | | | | |
| ENF14 | Scalabilité | Nombre d'assets par réseau | ≤ 10 000 assets sans dégradation mesurable des temps de réponse (ENF01) | MOYENNE |
| ENF15 | Scalabilité | Taille maximale d'un fichier CAO | Jusqu'à 100 Mo par fichier asset (stocké via le dépôt distribué 3D) | MOYENNE |
| ENF16 | Scalabilité | Utilisateurs simultanés par instance | 50 utilisateurs simultanés sans dégradation des temps de réponse (ENF01) | MOYENNE |
| ENF17 | Scalabilité | Extension horizontale | L'architecture supporte plusieurs instances Myr en parallèle via sessions Redis partagées (`REDIS_URL`) | BASSE |
| **Maintenabilité** | | | | |
| ENF18 | Maintenabilité | Isolation du domaine métier | Le domaine (`domain/`) ne contient aucune dépendance directe à Fabric, Redis, SQLite ou IPFS — vérifié par revue de code | HAUTE |
| ENF19 | Maintenabilité | Couverture de tests unitaires | ≥ 80 % des services domaine couverts par des tests unitaires (`go test ./domain/...`) | MOYENNE |
| ENF20 | Maintenabilité | Durée de déploiement d'une mise à jour | `make deploy` complet ≤ 10 min sur le serveur cible (compilation Linux + transfert + redémarrage) | BASSE |
| ENF21 | Maintenabilité | Compatibilité multi-réseaux blockchain | L'adapter Fabric est interchangeable sans modification du domaine — un nouvel adapter blockchain peut être branché en implémentant les ports `out` existants | HAUTE |
| **Portabilité** | | | | |
| ENF22 | Portabilité | Navigateurs supportés | Interface fonctionnelle sur Chrome 120+, Firefox 120+, Safari 17+, Edge 120+ sans installation locale | HAUTE |
| ENF23 | Portabilité | Plateformes serveur | Binaires disponibles pour Linux amd64 et Windows amd64 (produits par `make app` et `make cli`) | HAUTE |
| ENF24 | Portabilité | Absence de dépendance client | Aucun logiciel tiers n'est requis côté utilisateur final (pas de Java, .NET, Electron, etc.) | HAUTE |
| **Conformité légale** | | | | |
| ENF25 | Conformité | Licence du code source | Code source publié sous AGPL 3.0 ; toute contribution ou extension doit respecter cette licence | HAUTE |
| ENF26 | Conformité | Compatibilité de licence des assets | La compatibilité entre licence parent et licence dérivé est vérifiée automatiquement avant toute soumission d'un asset dérivé (`ParentID != ""`) | HAUTE |
| ENF27 | Conformité | Protection des données personnelles (RGPD) | Les données personnelles (email, profil) sont stockées uniquement en base de données locale ; aucun transfert vers un tiers sans consentement explicite | MOYENNE |
| **Fiabilité** | | | | |
| ENF28 | Fiabilité | Immuabilité des transactions blockchain | Aucune opération de suppression n'est implémentée sur la blockchain ; `ErrNotSupported` est retourné si une telle opération est tentée | HAUTE |
| ENF29 | Fiabilité | Anti-plagiat obligatoire | Tout asset de type `base` passe une vérification SHA-256 + similarité SCM > 50 % avant enregistrement — aucune exception possible | HAUTE |
| ENF30 | Fiabilité | Intégrité en cas d'échec blockchain | Si une transaction blockchain échoue, l'état local (draft) est conservé intact — aucune perte de données côté serveur | HAUTE |
| ENF31 | Fiabilité | Validation avant soumission | Toutes les données sont validées côté serveur avant soumission blockchain — les erreurs de validation ne génèrent pas de transaction partielle | HAUTE |



## Récapitulatif par priorité

| Priorité | Nombre | IDs |
|-|--|--|
| HAUTE | 20 | ENF01, ENF02, ENF05, ENF06, ENF08–ENF13, ENF18, ENF21–ENF26, ENF28–ENF31 |
| MOYENNE | 8 | ENF03, ENF04, ENF07, ENF14–ENF16, ENF19, ENF27 |
| BASSE | 2 | ENF17, ENF20 |



## Correspondance avec la section 2.3 de l'introduction

| Contrainte textuelle (section 2.3) | ENF correspondant |
|-|-|
| Immuabilité blockchain | ENF28, ENF31 |
| Vérification anti-plagiat | ENF29 |
| Compatibilité de licence | ENF26 |
| Licence Open-Source AGPL 3.0 | ENF25 |
| Architecture hexagonale | ENF18, ENF21 |
| Compatibilité multi-réseaux | ENF21 |
| Accès navigateur uniquement | ENF22, ENF24 |
