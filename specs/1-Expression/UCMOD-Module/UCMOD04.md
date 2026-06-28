---
categorie: Module
titre: "Visualiser les composants d'un Module"
probabilite: 2
impact: 5
importance: 10
---

# Visualiser les composants d'un Module

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Visualiser les composants d'un module" as UC1
    usecase "Explorer la hiérarchie" as UC2
}

U --> UC1
UC1 .> UC2 : <<extend>>

@enduml
```

## Contexte

Un Module est composé de composants visualisables. L'utilisateur peut explorer la structure interne d'un module pour en comprendre la composition.

## Pré-conditions

- Être connecté au réseau
- Avoir un module sélectionné

## Scénario

**Étape initiale :** L'utilisateur sélectionne un module dans l'atelier ou la bibliothèque

### Flux nominal — Vue composants affichée

1. L'utilisateur accède à "Voir les composants"
2. La liste des composants constitutifs est affichée avec leurs liaisons
3. L'utilisateur peut naviguer dans la hiérarchie (sous-modules éventuels)

## Post-conditions

- La composition complète du module est visible

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Visualiser les composants d'un Module
start
:Sélectionner un module dans l'atelier ou la bibliothèque;
:Accéder à "Voir les composants";
:Afficher la liste des composants constitutifs avec leurs liaisons;
if (Sous-modules à explorer?) then (oui)
  :Naviguer dans la hiérarchie des sous-modules;
  stop
else (non)
  stop
endif
@enduml
```
