---
categorie: Automatisation
titre: "Commande en ligne de Asset"
probabilite: 5
impact: 3
importance: 15
etat: relire
---

# Commande en ligne de asset

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Consommateur" as CL
actor "Developpeur" as D
actor "Manufactureur" as M

rectangle "Application MYR" {
    usecase "Commander un asset en ligne" as UC1
    usecase "Récupérer le prix via l'API" as UC2
    usecase "Transmettre la commande" as UC3
}

CL --> UC1
D --> UC2
M --> UC3
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>

@enduml
```

## Contexte

Commande en ligne d'un asset via l'interface MYR ou une boutique partenaire avec récupération du prix via l'API.

Conformément au principe de parité CLI/REST, une commande passée par une boutique partenaire via l'API doit pouvoir être reproduite en CLI pour le compte d'un consommateur, au même titre que via l'interface graphique.

## Pré-conditions

- Être connecté au réseau
- asset disponible à la commande
- Adresse de livraison renseignée

## Scénario

**Étape initiale :** Un client (boutique partenaire, script, interface graphique tierce...) appelle l'API pour commander un asset

### Flux nominal — Commande passée

1. Le prix de l'asset est récupéré via l'API
2. La commande est confirmée par le client
3. La transaction est enregistrée sur la blockchain
4. La commande est transmise à la boutique ou au manufactureur

## Post-conditions

- La commande est enregistrée et en cours de traitement
- L'utilisateur reçoit une confirmation

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Commande en ligne de Asset
start
:Récupérer le prix de l'asset via l'API;
:Confirmer la commande;
:Enregistrer la transaction sur la blockchain;
:Transmettre la commande à la boutique ou au manufactureur;
:Retourner une confirmation au client;
stop
@enduml
```
