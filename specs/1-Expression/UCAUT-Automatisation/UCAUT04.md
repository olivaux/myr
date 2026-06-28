---
categorie: Automatisation
titre: "Gestion SCM d'un modèle 3D"
probabilite: 1
impact: 5
importance: 5
etat: relire
---

# Gestion SCM d'un modèle 3D

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Enregistrer une version du modèle 3D" as UC1
    usecase "Exporter les modifications en XML" as UC2
}

C --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

L'utilisateur peut enregistrer par un format XML les modifications liées à un travail sur un modèle 3D (Source Control Management).

## Pré-conditions

- Être connecté au réseau
- Avoir un modèle 3D en cours d'édition

## Scénario

**Étape initiale :** L'utilisateur travaille sur un modèle 3D

### Flux nominal — Version enregistrée

1. L'utilisateur déclenche un enregistrement de version
2. Le système exporte les modifications en format XML
3. La version est enregistrée avec un commentaire et un horodatage
4. L'historique des versions est consultable

## Post-conditions

- Les modifications sont versionnées et exportables en XML
- L'historique des versions est traçable

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Gestion SCM d'un modèle 3D
start
:Travailler sur un modèle 3D;
:Déclencher un enregistrement de version;
:Exporter les modifications en format XML;
:Enregistrer la version avec commentaire et horodatage;
:Rendre l'historique des versions consultable;
stop
@enduml
```
