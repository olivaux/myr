---
categorie: Propriété Intellectuelle
titre: "Cloner un Composant sur un réseau exterieur"
probabilite: 1
impact: 2
importance: 2
etat: relire
---

# Cloner un Composant sur un réseau exterieur

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Cloner un composant sur un réseau externe" as UC1
    usecase "Vérifier compatibilité de licence" as UC2
}

C --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

Un composant peut être cloné vers un réseau MYR externe sous réserve de compatibilité de licence.

## Pré-conditions

- Être connecté au réseau source
- Avoir les droits de clonage (licence compatible)
- Réseau de destination accessible

## Scénario

**Étape initiale :** L'utilisateur sélectionne un composant et choisit "Cloner sur un autre réseau"

### Flux nominal — Clonage autorisé

1. L'utilisateur sélectionne le réseau de destination
2. Le système vérifie la compatibilité de licence
3. La transaction de clonage est soumise sur les deux réseaux

### Flux alternatif — Réseau cible déjà connu (profil de connexion existant)

1. Le réseau cible est déjà référencé dans les profils de connexion locaux
2. L'utilisateur sélectionne directement le réseau cible dans la liste des réseaux connus
3. Le clonage est initié sans saisie manuelle des informations du réseau cible
4. Le composant est publié sur le réseau cible avec le même UUID et les métadonnées originales

### Flux erreur — Licence incompatible

1. Message d'erreur : "La licence du composant ne permet pas le clonage vers ce réseau"

## Post-conditions

- Le composant est disponible sur le réseau de destination
- La traçabilité de l'origine est conservée

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Cloner un Composant sur un réseau extérieur
start
:Sélectionner un composant et choisir "Cloner sur un autre réseau";
if (Réseau cible déjà connu dans les profils locaux?) then (oui)
  :Sélectionner le réseau dans la liste des réseaux connus;
else (non)
  :Saisir manuellement les informations du réseau de destination;
endif
:Vérifier la compatibilité de licence;
if (Licence compatible?) then (oui)
  :Soumettre la transaction de clonage sur les deux réseaux;
  stop
else (non)
  :Afficher "La licence du composant ne permet pas le clonage vers ce réseau";
  stop
endif
@enduml
```
