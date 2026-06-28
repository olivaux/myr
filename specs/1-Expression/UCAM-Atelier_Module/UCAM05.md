---
categorie: Atelier Module
titre: "Transformation d'un composant en module"
probabilite: 1
impact: 4
importance: 4
etat: relire
---

# Transformation d'un composant en module

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Convertir composant en module" as UC1
    usecase "Configurer les sous-composants" as UC2
    usecase "Soumettre transaction blockchain" as UC3
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>

@enduml
```

## Contexte

Un composant existant peut être converti en module s'il est redécoupé en sous-systèmes indépendants. Cette opération correspond à la catégorie **découpage** dans la taxonomie des assets.

## Pré-conditions

- Être connecté au réseau
- Être propriétaire du composant à convertir
- Avoir les droits d'édition

## Scénario

**Étape initiale :** L'utilisateur sélectionne un composant existant

### Flux nominal — Conversion réussie

1. L'utilisateur choisit "Convertir en module"
2. Il définit les sous-composants constitutifs
3. Il configure les liaisons entre les sous-composants
4. La transaction "découpage" est soumise sur la blockchain

### Flux alternatif — Composant sans interfaces définies

1. Le composant sélectionné ne possède aucune interface définie
2. Le système avertit : "Ce composant n'a pas d'interfaces — le module résultant ne pourra pas être lié à d'autres composants"
3. L'utilisateur choisit de définir les interfaces avant de poursuivre (voir UCAM03)
4. Une fois les interfaces définies, la transformation en module est relancée normalement

## Post-conditions

- Le composant est transformé en module contenant ses sous-composants
- Les interfaces du module correspondent aux interfaces libres des sous-composants

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Transformation d'un composant en module
start
:Sélectionner un composant existant;
:Choisir "Convertir en module";
if (Composant sans interfaces définies?) then (oui)
  :Avertir "Le module résultant ne pourra pas être lié à d'autres composants";
  :Rediriger vers la définition d'interfaces (UCAM03);
  :Définir les interfaces manquantes;
  :Relancer la transformation;
else (non)
  :Définir les sous-composants constitutifs;
  :Configurer les liaisons entre les sous-composants;
  :Soumettre la transaction "découpage" sur la blockchain;
  stop
endif
@enduml
```
