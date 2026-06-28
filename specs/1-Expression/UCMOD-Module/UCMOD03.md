---
categorie: Module
titre: "Ajouter un lien URL d'un Module existant"
probabilite: 2
impact: 5
importance: 10
etat: relire
---

# Ajouter un lien URL d'un Module existant

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C
actor "Developpeur" as D

rectangle "Application MYR" {
    usecase "Ajouter un lien URL à un module" as UC1
    usecase "Mettre à jour sur la blockchain" as UC2
}

C --> UC1
D --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

Ajouter une ou plusieurs URL de référence (fiche produit, boutique, documentation) à un Module existant sur le réseau.

## Pré-conditions

- Être connecté au réseau
- Avoir les droits d'édition sur le module

## Scénario

**Étape initiale :** L'utilisateur sélectionne un module et accède à la section "Liens"

### Flux nominal — URL ajoutée

1. L'utilisateur clique sur "Ajouter un lien URL"
2. Il renseigne l'URL de référence
3. Il valide l'ajout
4. La transaction de mise à jour est soumise

## Post-conditions

- L'URL est associée au module sur le réseau

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Ajouter un lien URL d'un Module existant
start
:Sélectionner un module et accéder à la section "Liens";
:Cliquer sur "Ajouter un lien URL";
:Renseigner l'URL de référence;
:Valider l'ajout;
:Soumettre la transaction de mise à jour;
stop
@enduml
```
