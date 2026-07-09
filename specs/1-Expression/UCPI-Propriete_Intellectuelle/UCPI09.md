---
categorie: Propriété Intellectuelle
titre: "Cloner un Module sur un réseau exterieur"
probabilite: 1
impact: 2
importance: 2
etat: relire
---

# Cloner un Module sur un réseau exterieur

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Cloner un module sur un réseau externe" as UC1
    usecase "Vérifier licences de chaque composant" as UC2
}

C --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

Un module peut être cloné vers un réseau MYR externe, ce qui implique la vérification de licence de tous ses composants constitutifs.

Conformément au principe de parité CLI/REST, le clonage d'un module vers un réseau externe doit pouvoir être déclenché en CLI, au même titre que via l'interface graphique.

## Pré-conditions

- Être connecté au réseau source
- Avoir les droits de clonage sur le module et tous ses composants
- Réseau de destination accessible

## Scénario

**Étape initiale :** `myr model clone <id> --target-network <id>` est exécutée (ou l'appel API équivalent)

### Flux nominal — Clonage autorisé

1. Le réseau de destination est transmis
2. Le système vérifie la compatibilité de licence de chaque composant du module
3. La transaction de clonage est soumise pour le module et ses composants sur les deux réseaux

### Flux erreur — Licence incompatible sur un composant

1. Erreur métier : liste des composants dont la licence bloque le clonage

## Post-conditions

- Le module et ses composants sont disponibles sur le réseau de destination

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Cloner un Module sur un réseau extérieur
start
:Transmettre l'identifiant du module et le réseau cible (myr model clone);
:Vérifier la compatibilité de licence de chaque composant du module;
if (Toutes les licences compatibles?) then (oui)
  :Soumettre la transaction de clonage pour le module et ses composants;
  stop
else (non)
  :Retourner la liste des composants dont la licence bloque le clonage;
  stop
endif
@enduml
```
