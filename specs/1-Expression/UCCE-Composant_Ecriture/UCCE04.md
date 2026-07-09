---
categorie: Composant Ecriture
titre: "Améliorer un Composant"
probabilite: 2
impact: 5
importance: 10
etat: relire
---

# Améliorer un Composant

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Améliorer un composant" as UC1
    usecase "Référencer le composant parent" as UC2
    usecase "Enregistrer l'amélioration" as UC3
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>

@enduml
```

## Contexte

Un composant existant peut être amélioré (mêmes fonctionnalités mais renforcées) par un utilisateur autorisé. Il s'agit d'une AMELIORATION dans la taxonomie MYR.

## Pré-conditions

- Être connecté au réseau
- Avoir les droits d'amélioration sur le composant
- Composant de base existant sur le réseau

## Scénario

**Étape initiale :** `myr model add roue_v2.stl --name "Roue avant v2" --channel greenchannel --category amelioration --parent <id-parent> --license <id-licence>` est exécutée (ou l'appel API équivalent), pour le compte du Concepteur

### Flux nominal — Amélioration réussie

1. Le type AMELIORATION est automatiquement attribué
2. La version améliorée du composant est importée avec les modifications apportées
3. La transaction est soumise avec référence au composant parent — la compatibilité de licence entre le parent et la dérivation est vérifiée

### Flux alternatif — Amélioration avec ajout d'interfaces

1. La version améliorée du composant introduit de nouvelles interfaces non présentes dans la version parente
2. Le système détecte l'ajout de fonctionnalités et reclassifie le type en DERIVATION (le flag `--category` explicite prévaut si fourni)
3. La transaction est soumise avec le type final (DERIVATION) et la référence au composant parent

## Post-conditions

- Une nouvelle version améliorée du composant est enregistrée
- Elle est liée au composant parent via sa dépendance blockchain

## Diagrammes

### Cycle de vie — création, partage et amélioration d'un composant

```plantuml
@startuml
skin rose
title fonctionnement de l'échange
:user1: --> (model1) :create
:user1: --> (model2) :create
:user2: <-- (model2) :get
:user2: --> (model2+) :add
(model2) ..> (model2+) :improved
@enduml
```

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Améliorer un Composant
start
:Transmettre la version améliorée et la référence au parent (myr model add);
if (Nouvelles interfaces ajoutées?) then (oui)
  :Reclassifier automatiquement le type en DERIVATION;
  :Soumettre la transaction de type DERIVATION avec référence au parent;
  stop
else (non)
  :Attribuer automatiquement le type AMELIORATION;
  :Soumettre la transaction avec référence au composant parent;
  stop
endif
@enduml
```
