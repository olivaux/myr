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
- Avoir un identifiant de composant

## Scénario

**Étape initiale :** Les descendants directs et récursifs sont listés (`myr model children <parentID>`, ou l'appel API équivalent)

### Flux nominal — Arbre de versions retourné

1. Les descendants directs et récursifs sont listés (`myr model children <parentID>`)
2. Les ascendants sont reconstitués par appels successifs à `myr model get <parentID>` en suivant `ParentID` jusqu'à la racine
3. L'arbre complet (ascendants + descendants) est recomposé, avec le type de chaque version (AMELIORATION, VARIATION, ADAPTATION...)

## Post-conditions

- L'arbre complet des versions du composant est visible

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Rechercher les versions des Composants
start
:Lister les descendants (myr model children);
:Reconstituer les ascendants (myr model get suivant ParentID);
:Retourner l'arbre complet (parent → enfants avec leur type);
stop
@enduml
```
