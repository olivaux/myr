---
categorie: Compte et Accès
titre: "Demander un rôle"
probabilite: 4
impact: 4
importance: 16
etat: relire
---

# Demander un rôle

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U
actor "Administrateur" as ADM

rectangle "Application MYR" {
    usecase "Demander un rôle" as UC1
    usecase "Traiter la demande" as UC2
}

U --> UC1
ADM --> UC2
UC1 .> UC2 : <<extend>>

@enduml
```

## Contexte

Un utilisateur connecté (rôle Lecteur ou autre) peut demander un rôle supplémentaire depuis son profil. La demande est traitée automatiquement si le rôle est configuré en auto-distribution sur ce réseau, ou soumise à l'administrateur dans le cas contraire.

## Pré-conditions

- Être connecté
- Rôle cible différent du rôle déjà détenu

## Scénario

**Étape initiale :** L'utilisateur ouvre son profil et clique sur "Demander un rôle"

### Flux nominal — Attribution automatique

1. L'utilisateur sélectionne le rôle souhaité
2. Le rôle cible est configuré en auto-distribution sur ce réseau
3. Le rôle est attribué immédiatement
4. Message de confirmation : "Rôle [X] attribué"

### Flux alternatif — Validation manuelle par l'administrateur

1. L'utilisateur sélectionne le rôle souhaité
2. Le rôle cible nécessite une validation admin
3. La demande est transmise à l'administrateur
4. Message : "Demande envoyée — en attente de validation"
5. L'utilisateur conserve son rôle courant jusqu'à la décision

### Flux erreur — Rôle déjà attribué

1. Le rôle sélectionné est déjà détenu par l'utilisateur
2. Message : "Vous possédez déjà ce rôle"

## Post-conditions

- Rôle attribué immédiatement (auto-distribution), ou demande en attente de validation (manuel)

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Demander un rôle
start
:Ouvrir le profil — cliquer "Demander un rôle";
:Sélectionner le rôle souhaité;
if (Rôle déjà attribué?) then (oui)
  :Afficher "Vous possédez déjà ce rôle";
  stop
else (non)
  if (Auto-distribution activée pour ce rôle?) then (oui)
    :Attribuer le rôle immédiatement;
    :Afficher "Rôle [X] attribué";
    stop
  else (non)
    :Transmettre la demande à l'administrateur;
    :Afficher "Demande envoyée — en attente de validation";
    stop
  endif
endif
@enduml
```