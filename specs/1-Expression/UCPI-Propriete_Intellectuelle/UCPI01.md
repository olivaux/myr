---
categorie: Propriété Intellectuelle
titre: "Commander un Module complet"
probabilite: 5
impact: 5
importance: 25
---

# Commander un Module complet

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Consommateur" as CL
actor "Manufactureur" as M
actor "Developpeur" as D

rectangle "Application MYR" {
    usecase "Commander un module" as UC1
    usecase "Identifier boutique partenaire" as UC2
    usecase "Transmettre ordre de fabrication" as UC3
}

CL --> UC1
D --> UC2
M --> UC3
UC1 .> UC2 : <<extend>>
UC1 .> UC3 : <<extend>>

@enduml
```

## Contexte

Commande d'un module complet via le réseau MYR, impliquant soit l'achat sur une boutique existante, soit la fabrication par un manufactureur agréé.

## Pré-conditions

- Être connecté au réseau
- Module disponible à la commande
- Adresse de livraison renseignée

## Scénario

**Étape initiale :** L'utilisateur sélectionne un module et choisit "Commander"

### Flux nominal — Produit en stock

1. Le système identifie la boutique partenaire disposant du produit
2. La commande est transmise à la boutique
3. L'utilisateur reçoit une confirmation de commande

### Flux nominal — Fabrication nécessaire

1. Le système identifie un manufactureur agréé disponible
2. La commande de fabrication est transmise via la blockchain
3. L'utilisateur reçoit une confirmation avec délai de fabrication

## Post-conditions

- La commande est enregistrée sur la blockchain
- L'utilisateur reçoit une confirmation
- La livraison est en cours de traitement

## Diagrammes

### Commande d'un asset unitaire par un consommateur

```plantuml
@startuml
skin rose
:consommateur1: --> (assetD) :order x1
@enduml
```

### Flux complet — commande via site ou interface MYR

```plantuml
@startuml
skin rose
title fonctionnement de l'échange
:Consommateur: --> (website) :order
:Consommateur: --> (myr) :order
(website) --> (myr) : get CAO
(myr) --> (blockchain) : get CAO
(blockchain) ..> (manufacturer) : build product
(blockchain) ..> (shop) : buy product
(shop) ..> :Consommateur: :deliver
(manufacturer) ..> :Consommateur: :deliver
@enduml
```

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Commander un Module complet
start
:Sélectionner un module et choisir "Commander";
if (Produit en stock dans une boutique partenaire?) then (oui)
  :Identifier la boutique partenaire disposant du produit;
  :Transmettre la commande à la boutique;
  :Envoyer une confirmation à l'utilisateur;
  stop
else (non)
  :Identifier un manufactureur agréé disponible;
  :Transmettre la commande de fabrication via la blockchain;
  :Envoyer une confirmation avec délai de fabrication;
  stop
endif
@enduml
```
