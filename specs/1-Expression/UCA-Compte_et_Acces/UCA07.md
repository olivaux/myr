---
categorie: Compte et Accès
titre: "Vérification du rôle attribué"
probabilite: 1
impact: 1
importance: 1
etat: relire
---

# Vérification du rôle attribué

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Voir son rôle attribué" as UC1
}

U --> UC1

@enduml
```

## Contexte

Le rôle attribué à l'identité connectée est consultable via l'API ou le CLI.

<!-- #remarque : myr identity status / GET /api/identity/status retourne le statut d'enrôlement CA d'un wallet (pending/active/suspended), pas un rôle RBAC (reader/contributor/...). specs/2-Analyse/UCA-Compte_et_Acces/UCA07.md affirme au contraire qu'aucun endpoint ne permet de reconsulter le rôle après la connexion (le rôle n'est communiqué qu'une fois, dans la réponse de POST /api/identity/session). Les deux couches se contredisent sur la commande citée ici — à clarifier avant de considérer ce use case comme couvert. -->

## Pré-conditions

- Être connecté au réseau

## Scénario

**Étape initiale :** Le rôle de l'identité connectée est interrogé (`myr identity status` ou l'appel API équivalent)

### Flux nominal — Affichage du rôle

1. Le rôle attribué est retourné (ex : Concepteur, Consommateur, Manufactureur...)

## Post-conditions

- L'utilisateur connaît son rôle actuel

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Vérification du rôle attribué
start
:Interroger le rôle de l'identité connectée (myr identity status);
:Retourner le rôle attribué;
stop
@enduml
```
