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

## Pré-conditions

- Être connecté au réseau
- asset disponible à la commande
- Adresse de livraison renseignée

## Scénario

**Étape initiale :** L'utilisateur sélectionne un asset et choisit "Commander en ligne"

### Flux nominal — Commande passée

1. Le système récupère le prix via l'API et l'affiche à l'utilisateur
2. L'utilisateur confirme la commande
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
:Sélectionner un asset et choisir "Commander en ligne";
:Récupérer le prix via l'API et l'afficher;
:Confirmer la commande;
:Enregistrer la transaction sur la blockchain;
:Transmettre la commande à la boutique ou au manufactureur;
:Envoyer une confirmation à l'utilisateur;
stop
@enduml
```
