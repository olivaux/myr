---
categorie: Automatisation
titre: "Fabrication/Livraison d'un Composant"
probabilite: 5
impact: 4
importance: 20
etat: relire
---

# Fabrication/Livraison d'un Composant

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Consommateur" as CL
actor "Manufactureur" as M
actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Commander une fabrication" as UC1
    usecase "Transmettre CAO au manufactureur" as UC2
    usecase "Livrer le composant" as UC3
    usecase "Distribuer les commissions" as UC4
}

CL --> UC1
UC1 ..> UC2 : <<include>>
M --> UC3
UC3 ..> UC4 : <<include>>
C --> UC4

@enduml
```

## Contexte

Passage par un fabricant externe agréé par le réseau qui se chargera de créer et livrer le(s) composant(s) à l'adresse définie.

Conformément au principe de parité CLI/REST (CLAUDE.md), la confirmation de livraison par un manufactureur (et la distribution automatique des commissions qui s'ensuit) doit être déclenchable en CLI pour son compte, au même titre que via l'interface graphique ou l'API REST.

## Pré-conditions

- Être connecté au réseau
- Composant commandé avec fichier CAO disponible sur la blockchain
- Manufactureur agréé disponible sur le réseau
- Adresse de livraison renseignée

## Scénario

**Étape initiale :** Une commande de fabrication est déclenchée

### Flux nominal — Fabrication et livraison réussies

1. Le système identifie le manufactureur agréé disponible
2. Le fichier CAO et les spécifications sont transmis via la blockchain
3. Le manufactureur produit le composant
4. Le composant est livré à l'adresse définie
5. Le statut de commande est mis à jour sur le réseau

### Flux alternatif — Commande en lot (quantité > 1)

1. Le consommateur saisit une quantité supérieure à 1 dans sa commande
2. Le système vérifie si la capacité de fabrication disponible peut absorber le lot
3. Si la capacité est suffisante chez un seul manufactureur : la commande est acceptée comme lot unique
4. Si la capacité est partielle : le système propose de répartir la commande entre plusieurs manufactureurs
5. L'utilisateur accepte la répartition
6. Les ordres de fabrication sont transmis en parallèle sur la blockchain à chaque manufactureur concerné

### Flux erreur — Aucun manufactureur disponible

1. L'utilisateur est notifié et mis en liste d'attente

## Post-conditions

- Le composant est fabriqué et livré
- La commande est enregistrée sur la blockchain
- Les commissions des auteurs sont distribuées automatiquement

## Diagrammes

### Flux de commande — boutique partenaire ou API MYR vers manufactureur

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
title Fabrication/Livraison d'un Composant
start
:Déclencher une commande de fabrication;
if (Manufactureur agréé disponible?) then (oui)
  if (Quantité > 1 et capacité partielle?) then (oui)
    :Proposer une répartition entre plusieurs manufactureurs;
    :L'utilisateur accepte la répartition;
    :Transmettre les ordres de fabrication en parallèle sur la blockchain;
  else (non)
    :Identifier le manufactureur disponible;
    :Transmettre le fichier CAO et les spécifications via la blockchain;
  endif
  :Produire le(s) composant(s);
  :Livrer à l'adresse définie;
  :Mettre à jour le statut de commande sur le réseau;
  :Distribuer automatiquement les commissions des auteurs;
  stop
else (non)
  :Notifier l'utilisateur;
  :Mettre l'utilisateur en liste d'attente;
  stop
endif
@enduml
```
