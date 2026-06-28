---
categorie: Documentation
titre: "Compréhension de la documentation"
probabilite: 1
impact: 1
importance: 1
etat: relire
---

# Compréhension de la documentation

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Rechercher une information" as UC1
    usecase "Naviguer par sommaire" as UC2
    usecase "Utiliser la recherche plein texte" as UC3
}

U --> UC1
UC1 .> UC2 : <<extend>>
UC1 .> UC3 : <<extend>>

@enduml
```

## Contexte

La documentation doit être claire et les informations recherchées facilement trouvables pour tous les profils d'utilisateurs.

## Pré-conditions

- Avoir accès à la documentation

## Scénario

**Étape initiale :** L'utilisateur cherche une information précise dans la documentation

### Flux nominal — Information trouvée rapidement

1. L'utilisateur utilise la recherche plein texte ou le sommaire
2. L'information est trouvée en moins de 3 clics

## Post-conditions

- L'information recherchée est trouvée facilement et comprise

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Compréhension de la documentation
start
:Chercher une information précise dans la documentation;
if (Méthode de recherche?) then (sommaire)
  :Naviguer par sommaire;
else (recherche plein texte)
  :Utiliser la recherche plein texte;
endif
:Trouver l'information en moins de 3 clics;
stop
@enduml
```
