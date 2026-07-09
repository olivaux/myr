---
categorie: Recherche
titre: "Rechercher une référence existante"
probabilite: 4
impact: 5
importance: 20
etat: relire
---

# Rechercher une référence existante

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Utilisateur" as U

rectangle "Application MYR" {
    usecase "Rechercher une référence" as UC1
}

U --> UC1

@enduml
```

## Contexte

La recherche par référence permet de trouver précisément un composant ou un module par son identifiant ou sa référence exacte.

## Pré-conditions

- Être connecté au réseau

## Scénario

**Étape initiale :** `myr model get <id>` est exécutée avec la référence ou l'UUID recherché (ou l'appel API équivalent)

### Flux nominal — Référence trouvée

1. Le composant ou module correspondant est retourné

### Flux nominal — Référence introuvable

1. La réponse indique qu'aucun asset ne correspond à cette référence

### Flux alternatif — Référence exacte inconnue

1. `myr model list [--channel <id>]` permet de parcourir les assets du canal pour retrouver la référence recherchée

## Post-conditions

- L'asset trouvé est disponible pour consultation ou édition (voir UCCE02)

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Rechercher une référence existante
start
:Transmettre la référence ou l'UUID (myr model get);
if (Asset trouvé?) then (oui)
  :Retourner le composant ou module correspondant;
  stop
else (non)
  :Retourner "Aucun asset ne correspond à cette référence";
  stop
endif
@enduml
```
