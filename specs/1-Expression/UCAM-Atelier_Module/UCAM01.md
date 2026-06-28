---
categorie: Atelier Module
titre: "Liaison entre interfaces"
probabilite: 4
impact: 5
importance: 20
etat: relire
---

# Liaison entre interfaces

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Créer une liaison" as UC1
    usecase "Vérifier compatibilité des interfaces" as UC2
    usecase "Choisir un asset d'accroche" as UC3
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 .> UC3 : <<extend>>

@enduml
```

## Contexte

Création de nouvelles interfaces d'un composant par la création de liaisons entre composants. Le composant doit garder ses interfaces créées par liaisons.

Une liaison peut être **directe** (les interfaces se connectent sans intermédiaire) ou **via un asset d'accroche** (vis, câble, connecteur…). L'asset d'accroche est lui-même un composant existant sur le réseau — son identifiant (`FastenerAssetID`) est enregistré sur la liaison. Voir UCAM07.

## Pré-conditions

- Être connecté au réseau
- Avoir au moins deux composants avec des interfaces dans l'atelier
- Les interfaces à relier doivent être compatibles (type, sens, valeur)

## Scénario

**Étape initiale :** L'utilisateur sélectionne une interface d'un composant dans l'atelier

### Flux nominal — Liaison directe

1. L'utilisateur glisse l'interface vers une interface compatible d'un autre composant
2. Les interfaces non compatibles se grisent pour guider l'utilisateur
3. Le système vérifie la compatibilité de la paire d'interfaces
4. Le système propose optionnellement de choisir un asset d'accroche (voir UCAM07) — l'utilisateur peut ignorer
5. La liaison est enregistrée sans asset d'accroche (`FastenerAssetID` vide)
6. La liaison est représentée par un trait entre les deux interfaces
7. Les interfaces du composant résultant sont créées et conservées

### Flux nominal — Liaison via asset d'accroche

1. Étapes 1 à 3 identiques au flux nominal précédent
2. L'utilisateur choisit de sélectionner un asset d'accroche (voir UCAM07)
3. La liaison est enregistrée avec le `FastenerAssetID` de l'asset choisi
4. La liaison est représentée par un trait incluant l'asset d'accroche

### Comportement à la sélection — Interfaces non compatibles

Les interfaces incompatibles ne constituent pas un flux d'erreur à la création : l'interface graphique les prévient en amont.

1. Dès que l'utilisateur sélectionne une interface source, toutes les interfaces incompatibles se grisent automatiquement
2. Seules les interfaces compatibles restent cliquables
3. Il est donc impossible de créer une liaison incompatible via l'interface graphique

### Flux erreur — Interface déjà utilisée

1. Si l'interface cible est déjà engagée dans une liaison existante, elle est grisée
2. En cas de tentative forcée, le système affiche : "Cette interface est déjà utilisée dans une liaison"

### Flux — Liaison devenue incompatible après modification

Ce scénario survient lorsqu'un asset impliqué dans une liaison est remplacé par une version dont les interfaces ont changé (ex. : passage à une version améliorée avec un type de port différent).

1. La modification de l'asset est enregistrée
2. Le système détecte que la liaison existante n'est plus compatible avec les nouvelles interfaces
3. La liaison n'est **pas supprimée** — elle passe en état `incompatible`
4. Elle reste visible dans l'atelier, représentée en rouge
5. L'utilisateur peut choisir de la supprimer manuellement ou d'adapter les assets

## Post-conditions

- La liaison est enregistrée entre les deux composants
- Les interfaces libres du composant résultant sont visibles
- Une liaison devenue incompatible après modification reste présente, marquée en rouge (`Incompatible: true`) — elle n'est jamais supprimée automatiquement

## Diagrammes

### Principe d'une liaison entre deux interfaces compatibles

```plantuml
@startuml
skin rose
(C2) <-- (L2♀) : interface
(C1) <-- (L2♂) : interface

(L2♂) -- (L2♀) : liaison
@enduml
```

### Exemple concret — Raspberry Pi, caméra, écran, boitier

```plantuml
@startuml
title Exemple de liaison d'interface
skin rose

component rpi {
  port "trou" as T1
  port "trou" as T2
  port "trou" as T3
  port "trou" as T4
  port "port_camera"  as PC
  port "port_ecran"   as PE
}

component camera {
  port "port"   as PCAM
}

component ecran {
  port "port"   as PECR
}

component CableCSI {
  port "port"   as PCSI1
  port "port"   as PCSI2
}

component boitier {
  port "port"   as PB1
  port "port"   as PB2
  port "port"   as PB3
  port "port"   as PB4
}
together {
  component vis1 {
    port "male"   as PM1
  }
  component vis2 {
    port "male"   as PM2
  }
  component vis3 {
    port "male"   as PM3
  }
  component vis4 {
    port "male"   as PM4
  }
}

PM1 --> T1 : MECA
T1 -down-> PB1 : MECA

PM2 --> T2 : MECA
T2 --> PB2 : MECA

PM3 --> T3 : MECA
T3 --> PB3 : MECA

PM4 --> T4 : MECA
T4 --> PB4 : MECA

PE --> PCSI1 : ELEC
PCSI2 --> PECR : ELEC

PC --> PCAM : ELEC

@enduml
```

### Diagramme d'activités

```plantuml
@startuml
skin rose
title Liaison entre interfaces
start
:Sélectionner une interface source d'un composant dans l'atelier;
:Les interfaces incompatibles se grisent automatiquement;
if (Interface cible déjà utilisée?) then (oui)
  :Afficher "Cette interface est déjà utilisée dans une liaison";
  stop
else (non)
  :Glisser vers une interface compatible d'un autre composant;
  :Vérifier la compatibilité de la paire d'interfaces;
  if (Choisir un asset d'accroche?) then (oui)
    :Sélectionner un asset d'accroche (voir UCAM07);
    :Enregistrer la liaison avec FastenerAssetID;
    :Représenter la liaison avec l'asset d'accroche;
    stop
  else (non)
    :Enregistrer la liaison sans asset d'accroche;
    :Représenter la liaison par un trait;
    stop
  endif
endif
@enduml
```
