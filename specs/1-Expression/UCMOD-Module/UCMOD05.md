---
categorie: Module
titre: "Ajouter lien URL depuis Plugin Navigateur"
probabilite: 2
impact: 2
importance: 4
etat: relire
---

# Ajouter lien URL depuis Plugin Navigateur

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U
actor "Developpeur" as D

rectangle "Plugin Navigateur MYR" {
    usecase "Créer un module depuis une URL" as UC1
    usecase "Mettre à jour l'URL d'un module existant" as UC2
}

U --> UC1
D --> UC1
UC1 .> UC2 : <<extend>>

@enduml
```

## Contexte

Un Module peut être ajouté depuis le navigateur via un plugin. Si le Module existe déjà, MYR met à jour l'URL du Module existant.

## Pré-conditions

- Être connecté au réseau MYR
- Plugin navigateur MYR installé et activé
- Être sur une page produit dans le navigateur

## Scénario

**Étape initiale :** L'utilisateur navigue sur une page produit dans son navigateur

### Flux nominal — Nouveau module depuis URL

1. L'utilisateur active le plugin MYR
2. Le plugin détecte les informations du produit sur la page
3. Le module n'existe pas encore : un nouveau module est créé avec l'URL

### Flux nominal — Module existant mis à jour

1. Le plugin détecte que le module correspond à un existant sur le réseau
2. L'URL du module existant est mise à jour

## Post-conditions

- Le module est créé ou mis à jour avec l'URL de la page produit

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Ajouter lien URL depuis Plugin Navigateur
start
:Naviguer sur une page produit dans le navigateur;
:Activer le plugin MYR;
:Le plugin détecte les informations du produit sur la page;
if (Module déjà existant sur le réseau?) then (oui)
  :Mettre à jour l'URL du module existant;
  stop
else (non)
  :Créer un nouveau module avec l'URL;
  stop
endif
@enduml
```
