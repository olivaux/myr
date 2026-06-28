---
categorie: Développement autour de MYR
titre: "Utilisation du CLI"
probabilite: 1
importance: 0
etat: relire
---

# Utilisation du CLI

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Administrateur" as ADM

rectangle "CLI MYR (Serveur)" {
    usecase "Exécuter une commande CLI" as UC1
}

ADM --> UC1

@enduml
```

## Contexte

L'administrateur peut utiliser le CLI MYR directement sur le serveur pour administrer le réseau. Le Développeur n'a pas accès au CLI — il interagit avec MYR exclusivement via l'API REST.

## Pré-conditions

- Être connecté sur le serveur distant

## Scénario

**Étape initiale :** L'administrateur ouvre un terminal

### Flux nominal — Commande exécutée

1. L'utilisateur saisit une commande MYR CLI (ex : myr query asset --id UUID)
2. La commande est transmise au réseau
3. Le résultat est affiché en sortie texte dans le terminal

### Flux erreur — Commande invalide

1. Un message d'aide est affiché avec la syntaxe correcte

## Post-conditions

- La commande CLI a été exécutée avec succès

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Utilisation du CLI
start
:Ouvrir un terminal sur le serveur distant;
:Saisir une commande MYR CLI;
:Transmettre la commande au réseau;
if (Commande valide?) then (oui)
  :Afficher le résultat en sortie texte;
  stop
else (non)
  :Afficher un message d'aide avec la syntaxe correcte;
  stop
endif
@enduml
```
