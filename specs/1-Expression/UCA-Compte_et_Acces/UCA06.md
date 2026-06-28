---
categorie: Compte et Accès
titre: "Vérifier les possessions"
probabilite: 3
impact: 3
etat: relire
---

# Vérifier les possessions

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Voir ses assets possédés" as UC1
    usecase "Interroger la blockchain" as UC2
}

U --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

L'utilisateur peut consulter la liste de ses assets (composants, modules) enregistrés sur le réseau dont il est propriétaire.

## Pré-conditions

- Être connecté au réseau

## Scénario

**Étape initiale :** L'utilisateur accède à son profil ou à la section "Mes assets"

### Flux nominal — Affichage des possessions

1. Le système interroge la blockchain pour récupérer les assets dont l'utilisateur est propriétaire
2. La liste des composants et modules possédés est affichée

### Flux nominal — Aucune possession

1. Un message indique qu'aucun asset n'est enregistré

## Post-conditions

- L'utilisateur a une vue complète de ses possessions sur le réseau

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Vérifier les possessions
start
:Accéder à son profil ou à la section "Mes assets";
:Interroger la blockchain pour récupérer les assets possédés;
if (Assets trouvés?) then (oui)
  :Afficher la liste des composants et modules possédés;
  stop
else (non)
  :Afficher "Aucun asset enregistré";
  stop
endif
@enduml
```
