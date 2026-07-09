---
categorie: Compte et Accès
titre: "Se Déconnecter"
probabilite: 5
impact: 5
importance: 25
etat: relire
---

# Se Déconnecter

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Se déconnecter" as UC1
}

U --> UC1

@enduml
```

## Contexte

Un client (interface graphique tierce, script, plugin...) met fin à une session active en invalidant son token.

## Pré-conditions

- Disposer d'un token de session valide

## Scénario

**Étape initiale :** Le client transmet son token de session (`X-Myr-Token`) pour clôturer la session

### Flux nominal — Déconnexion réussie

1. La session associée au token est invalidée côté serveur
2. Toute requête ultérieure avec ce token est rejetée (`401 Unauthorized`)

## Post-conditions

- La session est terminée et le token n'est plus valide

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Se Déconnecter
start
:Transmettre le token de session à invalider;
:Fermer la session côté serveur;
:Rejeter toute requête ultérieure avec ce token (401);
stop
@enduml
```
