---
categorie: Composant Ecriture
titre: "Configurer un Composant"
probabilite: 3
impact: 5
importance: 15
etat: relire
---

# Configurer un Composant

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Configurer un composant" as UC1
    usecase "Définir nom et licence" as UC2
    usecase "Mettre à jour sur la blockchain" as UC3
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>

@enduml
```

## Contexte

Un composant doit pouvoir être configuré selon un nom et une licence par son propriétaire.

## Pré-conditions

- Être connecté au réseau
- Être propriétaire du composant ou avoir les droits d'édition

## Scénario

**Étape initiale :** `myr model update <id> --name <nom> --license <id-licence>` est exécutée (ou l'appel API équivalent), pour le compte du propriétaire du composant

### Flux nominal — Configuration réussie

1. Le nom du composant est défini ou modifié
2. La licence applicable est sélectionnée ou renseignée
3. La transaction de mise à jour est soumise sur la blockchain — les règles de compatibilité de licence s'appliquent

## Post-conditions

- La configuration du composant est mise à jour sur le réseau
- Les informations de nom et licence sont visibles par les autres utilisateurs

## Diagrammes

### Héritage de licences Commercial (C) / Non-Commercial (NC) entre assets

```plantuml
@startuml
skin rose
(Asset1.2 NC*) <-- (Asset1.1 C*)
(Asset2.2 C*)  <-- (Asset2.1 NC*)

(Asset3.3 C*)  <-- (Asset3.2 NC*)
(Asset3.2 NC*) <-- (Asset3.1 C*)

(Asset4.3 NC*) <-- (Asset4.2 C*)
(Asset4.2 C*)  <-- (Asset4.1 NC*)
@enduml
```

- **C** = Commercial — **NC** = Non-Commercial
- Asset 1.1 est Commercial ; son dérivé 1.2 est Non-Commercial : la licence de 1.2 doit être compatible avec 1.1.
- Asset 2.1 est Non-Commercial mais est utilisé comme base d'un asset Commercial 2.2 : la licence de 2.2 doit être compatible avec 2.1.

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Configurer un Composant
start
:Transmettre nom et/ou licence à modifier (myr model update);
:Soumettre la transaction de mise à jour sur la blockchain;
stop
@enduml
```
