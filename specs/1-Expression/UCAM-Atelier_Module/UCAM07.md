---
categorie: Atelier Module
titre: "Choisir un asset d'accroche (Fastener)"
probabilite: 3
impact: 3
importance: 9
etat: relire
---

# Choisir un asset d'accroche (Fastener)

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Sélectionner un asset d'accroche" as UC1
    usecase "Rechercher dans les composants" as UC2
    usecase "Ignorer (liaison directe)" as UC3
}

C --> UC1
C --> UC3
UC1 .> UC2 : <<extend>>

@enduml
```

## Contexte

Lors de la création d'une liaison (voir UCAM01), le système propose optionnellement de désigner un **asset d'accroche** : un composant existant sur le réseau qui sert d'intermédiaire physique entre les deux interfaces (vis, câble, connecteur, raccord…).

L'identifiant de cet asset (`FastenerAssetID`) est enregistré sur la `Connection`. Si l'utilisateur ignore cette étape, `FastenerAssetID` reste vide et la liaison est directe.

## Pré-conditions

- Être en cours de création d'une liaison (UCAM01)
- Avoir des composants disponibles sur le réseau (potentiels fasteners)

## Scénario

**Étape initiale :** Le système affiche le sélecteur d'accroche après validation de la compatibilité des interfaces

### Flux nominal — Asset d'accroche sélectionné

1. Le système affiche la liste des composants utilisables comme accroche (filtrée par compatibilité de type d'interface)
2. L'utilisateur sélectionne le composant d'accroche (ex : vis M3, câble USB-C)
3. Le `FastenerAssetID` est enregistré sur la `Connection`
4. La liaison est créée avec l'asset d'accroche référencé

### Flux nominal — Liaison directe (accroche ignorée)

1. L'utilisateur clique "Ignorer" ou ferme le sélecteur
2. La liaison est créée sans asset d'accroche (`FastenerAssetID` vide)

## Post-conditions

- La `Connection` est enregistrée avec ou sans `FastenerAssetID`
- Si un fastener est sélectionné, il apparaît visuellement sur la liaison dans l'atelier

## Diagrammes

### Types d'accroche selon la catégorie d'interface

```plantuml
@startuml
skin rose
title Exemples d'assets d'accroche par catégorie

(MECA) --> (Vis M2) : accroche
(MECA) --> (Vis M3) : accroche
(MECA) --> (Clip snap) : accroche

(ELEC) --> (Câble USB-C) : accroche
(ELEC) --> (Nappe FFC) : accroche
(ELEC) --> (Câble CSI) : accroche

(HYD) --> (Raccord rapide 6mm) : accroche
@enduml
```

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Choisir un asset d'accroche (Fastener)
start
:Valider la compatibilité des interfaces (depuis UCAM01);
:Afficher le sélecteur d'accroche;
if (Utilisateur sélectionne un asset d'accroche?) then (oui)
  :Afficher la liste des composants compatibles (filtrée par type d'interface);
  :Sélectionner le composant d'accroche;
  :Enregistrer le FastenerAssetID sur la Connection;
  :Créer la liaison avec l'asset d'accroche référencé;
  stop
else (non)
  :Cliquer "Ignorer" ou fermer le sélecteur;
  :Créer la liaison sans asset d'accroche (FastenerAssetID vide);
  stop
endif
@enduml
```
