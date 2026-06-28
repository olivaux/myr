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

**Étape initiale :** L'utilisateur sélectionne son composant et accède à la configuration

### Flux nominal — Configuration réussie

1. L'utilisateur définit ou modifie le nom du composant
2. L'utilisateur sélectionne ou renseigne la licence applicable
3. L'utilisateur valide les modifications
4. La transaction de mise à jour est soumise sur la blockchain

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
:Sélectionner le composant et accéder à la configuration;
:Définir ou modifier le nom du composant;
:Sélectionner ou renseigner la licence applicable;
:Valider les modifications;
:Soumettre la transaction de mise à jour sur la blockchain;
stop
@enduml
```
