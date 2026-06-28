---
categorie: Propriété Intellectuelle
titre: "Déclarer un composant similaire"
probabilite: 1
impact: 5
importance: 5
etat: relire
---

# Déclarer un composant similaire

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U
actor "Administrateur" as ADM

rectangle "Application MYR" {
    usecase "Signaler un composant similaire" as UC1
    usecase "Examiner le signalement" as UC2
}

U --> UC1
ADM --> UC2
UC1 .> UC2 : <<extend>>

@enduml
```

## Contexte

Signaler un composant similaire à un autre non pris en compte par le système. Forme de protection communautaire pour garantir la pertinence et l'intégrité du réseau.

## Pré-conditions

- Être connecté au réseau
- Avoir identifié deux composants similaires non liés dans le système

## Scénario

**Étape initiale :** L'utilisateur sélectionne un composant et choisit "Signaler un similaire"

### Flux nominal — Signalement soumis

1. L'utilisateur indique le composant de référence (similaire existant)
2. Il ajoute une justification
3. Le signalement est soumis à l'administration

## Post-conditions

- Le signalement est enregistré et soumis à l'examen de l'administration

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Déclarer un composant similaire
start
:Sélectionner un composant et choisir "Signaler un similaire";
:Indiquer le composant de référence (similaire existant);
:Ajouter une justification;
:Soumettre le signalement à l'administration;
stop
@enduml
```
