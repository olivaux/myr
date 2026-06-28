---
categorie: Interface Graphique
titre: "Schéma de navigation"
probabilite: 5
impact: 5
importance: 25
etat: relire
---

# Schéma de navigation

L'interface MYR est accessible depuis un navigateur web. Le client n'installe rien : il se connecte à l'adresse du serveur MYR de son organisation.

## Diagramme de navigation

```plantuml
@startuml
!theme plain
title MYR — Navigation (SPA navigateur)

' ── Démarrage ──────────────────────────────────────────────────────────────
[*] --> Demarrage
state "Démarrage" as Demarrage {
  [*] --> VerifAcces : Vérification du mode d'accès
}

' ── Branchement auth ───────────────────────────────────────────────────────
Demarrage --> MainWindow   : Session active
Demarrage --> MurIdentite  : Aucune session active

' ── Mur d'identité ─────────────────────────────────────────────────────────
state "Mur d'identité" as MurIdentite {
  state "Connexion" as IW_Login
  state "Inscription" as IW_Register
  IW_Login    --> IW_Register : "Créer un compte"
  IW_Register --> IW_Login    : "Connexion"
}
MurIdentite --> MainWindow     : Authentification réussie

' ── MenuBar (persistante sur toutes les pages) ──────────────────────────────
state "MenuBar [toutes les pages]" as MenuBar {
  state "Recherche" as MB_Search
  state "Profil"    as MB_Profil
  state "New Asset" as MB_NewAsset
}

' ── MainWindow ──────────────────────────────────────────────────────────────
state "MainWindow" as MainWindow {
  state "Search UI" as SearchUI {
    state "Add to Explorer" as AddExplorer
  }
  state "Explorer UI" as ExplorerUI {
    state "Liste des assets" as AssetListExplorer
  }
  state "Profil / Compte" as AccountUI
  state "Erreur 404" as Error404

  [*]         --> ExplorerUI
  AddExplorer --> ExplorerUI : Ajouter au résultat
}

' ── Asset UI (page dédiée par asset) ────────────────────────────────────────
state "Asset UI" as AssetUI {
  state "Vue détail" as AssetDetail {
    state "Liste sous-assets / interfaces" as SubAssets
  }
  state "Atelier" as Atelier {
    state "Assets en atelier" as WorkshopAssets
    [*] --> WorkshopAssets
  }
  [*]         --> AssetDetail
  AssetDetail --> Atelier     : Bouton Atelier
  Atelier     --> AssetDetail : Fermer l'atelier
}

' ── Navigation inter-pages ──────────────────────────────────────────────────
MainWindow  --> AssetUI    : Cliquer un asset dans l'Explorer
MB_Search   --> SearchUI   : Ouvrir la recherche (MainWindow)
MB_Profil   --> AccountUI  : Ouvrir le profil (MainWindow)
MB_NewAsset --> AssetUI    : Créer un nouvel asset (vide)

' ── Modales ─────────────────────────────────────────────────────────────────
state "Créer / Éditer un composant" as ModaleAsset
state "Soumettre à la blockchain"   as ModaleSubmit
state "Gérer une interface"         as ModaleIface
state "Configurer un slot virtuel"  as ModaleVC
state "Créer une liaison"           as MadaleLiaison
state "Choisir un asset d'accroche" as FastenerPicker
state "Importer un fichier 3D"      as ModaleImport
state "Demander un rôle"            as ModaleRole

' depuis Asset UI
AssetUI --> ModaleAsset : Créer / Dériver un composant
AssetUI --> ModaleIface : Gérer les interfaces (vue détail)

' depuis Atelier
Atelier --> ModaleIface      : Ajouter une interface
Atelier --> ModaleVC         : Cliquer un slot virtuel
Atelier --> MadaleLiaison    : Relier deux interfaces
Atelier --> ModaleImport     : Déposer un fichier 3D
Atelier --> ModaleSubmit     : Soumettre le module
MadaleLiaison --> FastenerPicker : Choisir un accroche

' depuis Profil / Compte
AccountUI --> ModaleRole : Demander un rôle

' retour des modales
ModaleAsset    --> AssetUI   : Valider / Annuler
ModaleSubmit   --> Atelier   : Confirmer / Annuler
ModaleIface    --> AssetUI   : Fermer (vue détail)
ModaleIface    --> Atelier   : Fermer (contexte atelier)
ModaleVC       --> Atelier   : Créer et lier / Annuler
MadaleLiaison  --> Atelier   : Créer liaison / Annuler
FastenerPicker --> Atelier   : Sélectionner / Ignorer
ModaleImport   --> Atelier   : Importer / Annuler
ModaleRole     --> AccountUI : Confirmer / Annuler

' déconnexion
MainWindow --> MurIdentite : Se déconnecter

@enduml
```

## Légende des écrans

| Écran | Rôle | UC associé |
|---|---|---|
| Mur d'identité | Contrôle d'accès — affiché si réseau actif et pas de session | UCA01, UCA02 |
| MenuBar | Barre persistante (toutes les pages) — recherche, profil, création d'asset | UCA03, UCA07 |
| MainWindow | Vue principale — recherche et exploration des assets du réseau | — |
| Search UI | Saisie de critères, affichage des résultats, ajout à l'Explorer | UCREC01–05 |
| Explorer UI | Liste des assets ajoutés depuis la recherche, point d'entrée vers Asset UI | UCCL01 |
| Asset UI | Page dédiée à un asset — détail, sous-assets, interfaces, accès à l'Atelier | UCCL01 |
| Atelier (embarqué dans Asset UI) | Composition et assemblage d'un module — liaisons entre interfaces | UCMOD01, UCMOD06, UCAM01–08 |
| Profil / Compte | Informations du compte utilisateur et rôles attribués | UCA03, UCA07, UCA08 |
| Créer / Éditer un composant | Formulaire de création ou d'édition d'un composant | UCCE01–06 |
| Soumettre à la blockchain | Confirmation de l'ancrage blockchain d'un module | UCMOD06 |
| Gérer une interface | Ajout ou édition d'une interface sur un asset | UCAM03 |
| Configurer un slot virtuel | Matérialisation d'un slot virtuel en interface physique | UCAM03 |
| Créer une liaison | Création d'une liaison entre deux interfaces | UCAM01 |
| Choisir un asset d'accroche | Sélection optionnelle d'un fastener pour une liaison | UCAM07 |
| Importer un fichier 3D | Import d'un fichier de modélisation par glisser-déposer | UCAM04, UCAM06 |
| Demander un rôle | Formulaire de demande de rôle (attribution automatique ou validation admin) | UCA08 |

## Modes d'accès au démarrage

| Mode | Condition | Comportement |
|---|---|---|
| Session active | Token JWT valide en mémoire | La MainWindow s'affiche directement |
| Connexion | Compte existant, pas de session | L'utilisateur saisit email + mot de passe — provisionnement blockchain transparent à la première connexion |
| Inscription | Aucun compte | L'utilisateur crée un compte — validé automatiquement avec le rôle **Lecteur** par défaut |

> Un rôle supplémentaire peut être demandé depuis le profil (voir UCA08).

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Schéma de navigation — démarrage et accès
start
:Ouvrir l'application MYR dans le navigateur;
if (Session active?) then (oui)
  :Afficher la MainWindow;
  stop
else (non)
  :Afficher le Mur d'identité;
  if (Compte existant?) then (oui)
    :Se connecter (email + mot de passe);
    :Afficher la MainWindow;
    stop
  else (non)
    :Cliquer "Créer un compte";
    :Remplir le formulaire d'inscription;
    :Compte créé — rôle Lecteur attribué par défaut;
    :Se connecter automatiquement;
    :Afficher la MainWindow;
    stop
  endif
endif
@enduml
```
