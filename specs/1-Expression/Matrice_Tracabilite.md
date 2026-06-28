# Matrice de traçabilité — Exigences fonctionnelles ↔ Use Cases

Ce document croise les exigences fonctionnelles (EF) du projet Myr avec les use cases (UC) qui les couvrent. Il permet de vérifier qu'aucune exigence n'est orpheline et qu'aucun UC n'est sans motivation fonctionnelle.

---

## 1. Exigences fonctionnelles et UC couvrant

Les exigences sont dérivées des objectifs du projet (section 2.1), des contraintes (section 2.3) et de l'ensemble des use cases.


| ID                                               | Exigence fonctionnelle                                                  | UC couvrant               |
| ------------------------------------------------ | ----------------------------------------------------------------------- | ------------------------- |
| **Compte et Accès**                             |                                                                         |                           |
| EF01                                             | Permettre à un visiteur de créer un compte sur un réseau             | UCA01                     |
| EF02                                             | Authentifier un utilisateur (email/password + JWT)                      | UCA02                     |
| EF03                                             | Déconnecter un utilisateur et invalider sa session                     | UCA03                     |
| EF04                                             | Vérifier la validité d'une session active                             | UCA04                     |
| EF05                                             | Contrôler les accès selon le rôle attribué                          | UCA05, UCA07              |
| EF06                                             | Consulter les assets possédés par l'utilisateur                       | UCA06                     |
| **Administration**                               |                                                                         |                           |
| EF07                                             | Créer un réseau blockchain indépendant                               | UCADM02                   |
| EF08                                             | Gérer les organisations membres d'un réseau (ajout, mise à jour)     | UCADM01                   |
| EF09                                             | Étendre un réseau avec de nouveaux nœuds (peer ou orderer)           | UCADM03                   |
| **Composants — Création et édition**          |                                                                         |                           |
| EF10                                             | Enregistrer un composant physique sur la blockchain                     | UCCE01                    |
| EF11                                             | Enregistrer un composant numérique sur la blockchain                   | UCCE03                    |
| EF12                                             | Configurer les métadonnées d'un composant                             | UCCE02                    |
| EF13                                             | Faire évoluer un composant (amélioration, dérivation, extension)     | UCCE04, UCCE05            |
| EF14                                             | Ajouter une interface à un composant existant                          | UCCE06                    |
| EF15                                             | Vérifier l'unicité d'un composant (anti-plagiat SHA-256 + SCM > 50 %) | UCCE01                    |
| EF16                                             | Vérifier la compatibilité de licence lors d'une dérivation           | UCCE04, UCMOD06           |
| **Composants — Lecture**                        |                                                                         |                           |
| EF17                                             | Rechercher et filtrer les composants disponibles sur le réseau         | UCCL01                    |
| **Atelier (Workspace)**                          |                                                                         |                           |
| EF18                                             | Créer des liaisons entre interfaces compatibles de composants          | UCAM01                    |
| EF19                                             | Visualiser les interfaces physiques d'un composant                      | UCAM02                    |
| EF20                                             | Définir une interface sur un composant dans l'atelier                  | UCAM03                    |
| EF21                                             | Ajouter plusieurs composants simultanément à l'atelier                | UCAM04                    |
| EF22                                             | Transformer un composant en module (découpage en sous-systèmes)       | UCAM05                    |
| EF23                                             | Choisir un asset d'accroche (fastener) pour une liaison                 | UCAM07                    |
| EF24                                             | Retirer un composant de l'atelier (avec cascade des connexions)         | UCAM08                    |
| EF25                                             | Garantir un slot virtuel disponible sur chaque asset de l'atelier       | UCAM06                    |
| **Modules**                                      |                                                                         |                           |
| EF26                                             | Assembler plusieurs composants en module (état draft)                  | UCMOD01                   |
| EF27                                             | Soumettre un module à la blockchain (ModuleVersion immuable)           | UCMOD06                   |
| EF28                                             | Ajouter un module existant à l'espace de travail                       | UCMOD02, UCMOD03, UCMOD05 |
| EF29                                             | Visualiser la composition d'un module                                   | UCMOD04                   |
| **Propriété Intellectuelle et Rémunération** |                                                                         |                           |
| EF30                                             | Commander un module complet (fabrication ou achat en stock)             | UCPI01                    |
| EF31                                             | Distribuer automatiquement les commissions aux auteurs à la livraison  | UCPI02, UCAUT01           |
| EF32                                             | Définir un prix sur un composant propriétaire                         | UCPI04                    |
| EF33                                             | Définir un prix sur un module propriétaire                            | UCPI05                    |
| EF34                                             | Signaler un composant similaire à un existant                          | UCPI06                    |
| EF35                                             | Transférer la propriété intellectuelle d'un asset                    | UCPI07                    |
| EF36                                             | Cloner un composant sur un réseau externe                              | UCPI08                    |
| EF37                                             | Cloner un module sur un réseau externe                                 | UCPI09                    |
| EF38                                             | Respecter et vérifier une norme d'écoconception                       | UCPI10                    |
| **Recherche**                                    |                                                                         |                           |
| EF39                                             | Rechercher un asset par référence ou filtre                           | UCREC01                   |
| EF40                                             | Identifier les composants compatibles entre eux (interfaces)            | UCREC02                   |
| EF41                                             | Consulter l'arbre de versions d'un composant                            | UCREC03                   |
| EF42                                             | Identifier tous les modules qui intègrent un composant donné          | UCREC04                   |
| EF43                                             | Exporter la BOM (Bill Of Materials) d'un module                         | UCREC05                   |
| **Automatisation**                               |                                                                         |                           |
| EF44                                             | Automatiser la fabrication et la livraison d'un composant               | UCAUT01                   |
| EF45                                             | Automatiser la commande en ligne d'un asset                             | UCAUT02                   |
| EF46                                             | Intégrer un modèle 3D depuis un logiciel CAO (plugin)                 | UCAUT03                   |
| EF47                                             | Gérer les versions SCM d'un modèle 3D                                 | UCAUT04                   |
| **Interface Graphique**                          |                                                                         |                           |
| EF48                                             | Proposer une navigation UI cohérente (MenuBar, Explorer, Atelier)      | UCIG01                    |
| EF49                                             | Gérer les erreurs de manière explicite côté UI                      | UCIG02                    |
| **Paramètres**                                  |                                                                         |                           |
| EF50                                             | Configurer la langue de l'interface (Anglais)                           | UCPAR01                   |
| EF51                                             | Configurer la langue de l'interface (Chinois)                           | UCPAR02                   |
| **Documentation**                                |                                                                         |                           |
| EF52                                             | Accéder à la documentation du système                                | UCDOC01                   |
| EF53                                             | Consulter la FAQ                                                        | UCDOC02                   |
| EF54                                             | Comprendre la documentation technique                                   | UCDOC03                   |
| **Développement autour de MYR**                 |                                                                         |                           |
| EF55                                             | Exposer une API REST pour les intégrations tierces                     | UCDEV01                   |
| EF56                                             | Proposer un CLI d'administration serveur                                | UCDEV02                   |

