---
categorie: Compte et Accès
titre: "Vérification des accès du rôle attribué"
probabilite: 1
impact: 3
importance: 3
etat: relire
---

# Vérification des accès du rôle attribué

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U
actor "Administrateur" as ADM

rectangle "Application MYR" {
    usecase "Effectuer une action" as UC1
    usecase "Vérifier les droits du rôle" as UC2
    usecase "Attribuer un rôle" as UC3
}

U --> UC1
UC1 ..> UC2 : <<include>>
ADM --> UC3

@enduml
```

## Contexte

Vérifier que l'utilisateur peut réaliser les actions autorisées par son rôle et ne peut pas réaliser celles qui lui sont refusées.

## Pré-conditions

- Être connecté au réseau
- Rôle attribué à l'utilisateur

## Scénario

**Étape initiale :** L'utilisateur tente d'effectuer une action

### Flux nominal — Action autorisée

1. L'action est dans les droits du rôle attribué
2. Le système exécute l'action

### Flux erreur — Action non autorisée

1. L'action n'est pas dans les droits du rôle attribué
2. Message d'erreur : "Vous n'avez pas les droits nécessaires pour cette action"

## Post-conditions

- Les droits du rôle sont respectés

## Diagrammes

### Rôles disponibles dans le système

```plantuml
@startuml
:Administrateur:
:Lecteur:
:Concepteur:
:Consommateur:
:Manufactureur:
:Developpeur:
@enduml
```

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Vérification des accès du rôle attribué
start
:Tenter d'effectuer une action;
if (Action dans les droits du rôle?) then (oui)
  :Exécuter l'action;
  stop
else (non)
  :Afficher "Vous n'avez pas les droits nécessaires pour cette action";
  stop
endif
@enduml
```
