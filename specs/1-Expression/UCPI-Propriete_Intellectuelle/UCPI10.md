---
categorie: Propriété Intellectuelle
titre: "Norme de conception écoconception"
probabilite: 2
impact: 1
importance: 2
---

# Norme de conception écoconception

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Administrateur" as ADM
actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Définir normes d'écoconception" as UC1
    usecase "Vérifier conformité écoconception" as UC2
}

ADM --> UC1
C --> UC2

@enduml
```

## Contexte

Les composants et modules peuvent être soumis à des normes d'écoconception définies par l'administrateur du réseau pour promouvoir l'économie circulaire.

## Pré-conditions

- Normes d'écoconception définies par l'administrateur du réseau
- Composant ou module en cours de création ou de modification

## Scénario

**Étape initiale :** L'utilisateur crée ou modifie un composant

### Flux nominal — Conforme

1. Le système vérifie automatiquement la conformité aux normes d'écoconception
2. Un rapport de conformité positif est affiché

### Flux nominal — Non conforme

1. Le système avertit l'utilisateur des critères non respectés
2. Des recommandations d'amélioration sont proposées

## Post-conditions

- La conformité écoconception est vérifiée et documentée sur le réseau

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Norme de conception écoconception
start
:Créer ou modifier un composant;
:Vérifier automatiquement la conformité aux normes d'écoconception;
if (Conforme?) then (oui)
  :Afficher un rapport de conformité positif;
  stop
else (non)
  :Avertir l'utilisateur des critères non respectés;
  :Proposer des recommandations d'amélioration;
  stop
endif
@enduml
```
