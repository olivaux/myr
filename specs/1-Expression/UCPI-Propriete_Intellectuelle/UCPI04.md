---
categorie: Propriété Intellectuelle
titre: "Définir un prix sur un Composant proprietaire"
probabilite: 3
impact: 5
importance: 15
etat: relire
---

# Définir un prix sur un Composant proprietaire

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Définir un prix sur un composant" as UC1
    usecase "Enregistrer le prix sur la blockchain" as UC2
}

C --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

L'auteur d'un composant peut définir un prix unitaire pour l'utilisation commerciale ou la commande de celui-ci, enregistré sur la blockchain.

Conformément au principe de parité CLI/REST, la définition d'un prix sur un composant doit être exposable en CLI au même titre que via l'interface graphique.

## Pré-conditions

- Être connecté au réseau
- Être propriétaire du composant
- Avoir les droits de tarification

## Scénario

**Étape initiale :** `myr model price set <id> <montant> --currency <devise>` est exécutée (ou l'appel API équivalent), pour le compte du propriétaire du composant

### Flux nominal — Prix défini

1. Le prix unitaire et la devise sont transmis
2. La transaction est validée sur la blockchain

## Post-conditions

- Le prix est enregistré sur le réseau et appliqué lors des commandes

## Diagrammes

### Arbre de dérivation des assets et impact sur la tarification

```plantuml
@startuml
skin rose
:consommateur1: <.. (assetD) :order x1
:consommateur2: <.. (ProductA) :order x10
(assetA) --> (assetB) :Variation
(assetA) --> (assetC) :Extension
(assetB) --> (assetD) :Amelioration
(assetC) --> (assetF) :Derivation
(assetB) --> (assetE) :Adaptation
(assetG) --> (assetH) :Amelioration

(assetB) ..> (ProductA)
(assetF) ..> (ProductA)
(assetH) ..> (ProductA)
@enduml
```

Le prix d'un composant dérivé est fixé librement par son propriétaire (RM31, RM33), indépendamment du prix de ses parents. La compatibilité de licence avec chaque composant parent de la chaîne de dérivation est vérifiée séparément, à la soumission (RM03).

#question La distribution de commission (RM23/RM24) traverse-t-elle la chaîne de dérivation (un auteur reçoit-il une part quand un composant dérivé de son travail est vendu) ou se limite-t-elle à la composition d'un module (seuls les auteurs des composants directement inclus dans le module livré sont rémunérés) ? Voir `specs/1-Expression/Regles_Metier.md` RM23 pour le détail de l'ambiguïté.

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Définir un prix sur un Composant propriétaire
start
:Transmettre prix unitaire et devise (myr model price set);
:Valider la transaction sur la blockchain;
stop
@enduml
```
