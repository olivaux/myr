---
categorie: Recherche
titre: "Exporter BOM Module"
probabilite: 4
impact: 3
importance: 12
---

# Exporter BOM Module

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C
actor "Consommateur" as CL
actor "Manufactureur" as M

rectangle "Application MYR" {
    usecase "Exporter BOM d'un module" as UC1
    usecase "Choisir le format d'export" as UC2
}

C --> UC1
CL --> UC1
M --> UC1
UC1 ..> UC2 : <<include>>

@enduml
```

## Contexte

Exporter une "Bill Of Material" (liste des matériaux) d'un Module, listant tous les composants constitutifs avec leurs quantités et références.

## Pré-conditions

- Être connecté au réseau
- Avoir un module sélectionné

## Scénario

**Étape initiale :** L'utilisateur sélectionne un module et choisit "Exporter BOM"

### Flux nominal — BOM exportée

1. L'utilisateur choisit le format d'export (CSV, XML, PDF...)
2. Le système génère la BOM avec tous les composants et leurs quantités
3. Le fichier BOM est téléchargé

### Flux alternatif — Composants avec données IPFS indisponibles

1. La BOM contient des composants dont les fichiers IPFS ne sont pas accessibles (nœud hors ligne ou données non pinnées)
2. Le système génère la BOM avec un placeholder pour chaque composant concerné (`fichier: "indisponible"`)
3. Un avertissement est affiché : "X composant(s) sans fichier accessible — BOM partielle"
4. L'export est proposé avec les données disponibles

## Post-conditions

- Le fichier BOM est généré et disponible

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Exporter BOM Module
start
:Sélectionner un module et choisir "Exporter BOM";
:Choisir le format d'export (CSV, XML, PDF...);
:Générer la BOM avec tous les composants et leurs quantités;
if (Données IPFS partiellement indisponibles?) then (oui)
  :Générer la BOM avec placeholders pour les composants inaccessibles;
  :Afficher "BOM partielle — X composant(s) sans fichier accessible";
  stop
else (non)
  :Télécharger le fichier BOM complet;
  stop
endif
@enduml
```
