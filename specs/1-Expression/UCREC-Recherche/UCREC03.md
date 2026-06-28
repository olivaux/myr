---
categorie: Recherche
titre: "Rechercher les versions des Composants"
probabilite: 3
impact: 4
importance: 12
---

# Rechercher les versions des Composants

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C
actor "Consommateur" as CL

rectangle "Application MYR" {
    usecase "Rechercher versions d'un composant" as UC1
    usecase "Explorer l'arbre de dépendances" as UC2
}

C --> UC1
CL --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

À partir d'un composant, lister les différentes versions qu'elles soient parentes ou enfants (améliorations, variations, adaptations, extensions, dérivations...).

## Pré-conditions

- Être connecté au réseau
- Avoir un composant sélectionné

## Scénario

**Étape initiale :** L'utilisateur sélectionne un composant et accède à "Versions"

### Flux nominal — Arbre de versions affiché

1. Le système remonte la chaîne de dépendances sur la blockchain
2. L'arbre de versions est affiché (parent → enfants avec leur type : AMELIORATION, VARIATION, ADAPTATION...)
3. L'utilisateur peut naviguer dans l'arbre et consulter chaque version

## Post-conditions

- L'arbre complet des versions du composant est visible

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Rechercher les versions des Composants
start
:Sélectionner un composant et accéder à "Versions";
:Remonter la chaîne de dépendances sur la blockchain;
:Afficher l'arbre de versions (parent → enfants avec leur type);
if (Navigation dans l'arbre?) then (oui)
  :Consulter une version spécifique;
  stop
else (non)
  stop
endif
@enduml
```
