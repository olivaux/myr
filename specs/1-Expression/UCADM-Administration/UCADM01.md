---
categorie: Administration
titre: "Ajouter une organisation au réseau"
---

# Ajouter une organisation au réseau

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Administrateur" as ADM

rectangle "Application MYR" {
    usecase "Ajouter une organisation" as UC1
    usecase "Configurer les droits d'accès" as UC2
}

ADM --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

L'administrateur peut ajouter une organisation (Fabricant, vendeur, association...) au réseau avec son rôle et ses droits d'accès.

## Pré-conditions

- Être connecté en tant qu'administrateur du réseau
- Réseau opérationnel

## Scénario

**Étape initiale :** L'administrateur accède à la gestion des organisations

### Flux nominal — Organisation ajoutée

1. Il clique sur "Ajouter une organisation"
2. Il renseigne les informations : nom, MSP ID, rôle par défaut
3. Il configure les droits d'accès de l'organisation
4. Il valide : la configuration du channel est mise à jour
5. L'organisation peut désormais gérer ses propres utilisateurs

### Flux alternatif — Organisation déjà membre du réseau

1. L'administrateur saisit un MSP ID déjà enregistré sur le réseau
2. Le système détecte que l'organisation existe et propose de mettre à jour ses informations (rôle par défaut, politique d'accès)
3. L'administrateur modifie les informations souhaitées et valide
4. La configuration du channel est mise à jour sans recréation de l'organisation

## Post-conditions

- L'organisation est disponible sur le réseau avec les droits définis
- L'administrateur de l'organisation peut créer des comptes pour ses membres

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Ajouter une organisation au réseau
start
:Accéder à la gestion des organisations;
:Cliquer sur "Ajouter une organisation";
:Renseigner les informations (nom, MSP ID, rôle par défaut);
if (MSP ID déjà membre du réseau?) then (oui)
  :Proposer la mise à jour des informations existantes;
  :Modifier le rôle ou la politique d'accès;
  :Valider la mise à jour;
  :Mettre à jour la configuration du channel;
  stop
else (non)
  :Configurer les droits d'accès de l'organisation;
  :Valider la configuration;
  :Mettre à jour la configuration du channel;
  stop
endif
@enduml
```
