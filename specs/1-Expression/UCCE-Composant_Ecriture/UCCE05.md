---
categorie: Composant Ecriture
titre: "Créer une extension de Composant"
probabilite: 2
impact: 5
importance: 10
etat: relire
---

# Créer une extension de Composant

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Créer une extension de composant" as UC1
    usecase "Référencer le composant de base" as UC2
    usecase "Définir interfaces complémentaires" as UC3
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>

@enduml
```

## Contexte

Création d'une pièce complémentaire à un composant existant. Il s'agit d'une EXTENSION dans la taxonomie MYR.

## Pré-conditions

- Être connecté au réseau
- Avoir les droits de création
- Composant de base existant sur le réseau

## Scénario

**Étape initiale :** L'utilisateur sélectionne un composant de base et choisit "Créer une extension"

### Flux nominal — Extension créée

1. Le type EXTENSION est attribué
2. Les interfaces du composant de base sont copiées comme référence
3. L'utilisateur définit les interfaces complémentaires de l'extension
4. La transaction est soumise avec référence au composant de base

## Post-conditions

- L'extension est enregistrée et liée au composant de base
- Les interfaces de compatibilité sont documentées

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Créer une extension de Composant
start
:Sélectionner un composant de base;
:Choisir "Créer une extension";
:Attribuer le type EXTENSION;
:Copier les interfaces du composant de base comme référence;
:Définir les interfaces complémentaires de l'extension;
:Soumettre la transaction avec référence au composant de base;
stop
@enduml
```
