# GitFlow du dépôt

Ce dépôt utilise un fonctionnement GitFlow adapté afin de séparer clairement le cahier des charges, le développement, la stabilisation et la production.

## Branches principales

### `main`

La branche `main` contient uniquement du code stable, testé et prêt pour la production.

Chaque version livrée doit être taguée depuis cette branche.

Exemples :

```bash
v1.0.0
v1.1.0
v2.0.0
```

### `develop`

La branche `develop` est la branche d’intégration.

Elle contient les développements validés qui seront inclus dans la prochaine version.

### `specs`

La branche `specs` contient le cahier des charges et les documents de spécification du projet.

Elle sert à formaliser les besoins avant le développement.

Elle peut contenir par exemple :

* le cahier des charges fonctionnel ;
* les exigences techniques ;
* les règles métier ;
* les cas d’utilisation ;
* les contraintes du projet ;
* les schémas d’architecture ;
* les documents de conception.

Les modifications importantes du cahier des charges doivent être faites dans `specs`, puis réintégrées dans `develop` lorsque les spécifications sont validées.

## Branches temporaires

### `feature/*`

Les branches `feature/*` servent à développer de nouvelles fonctionnalités.

Elles sont créées depuis `develop`, puis fusionnées dans `develop` une fois terminées.

Une fonctionnalité doit idéalement correspondre à une spécification validée dans `specs`.

Exemples :

```text
feature/login
feature/export-pdf
feature/api-user
```

### `bugfix/*`

Les branches `bugfix/*` servent à corriger des bugs non critiques détectés pendant le développement.

Elles sont créées depuis `develop`, puis fusionnées dans `develop`.

Exemple :

```text
bugfix/correction-affichage-date
```

### `release/*`

Les branches `release/*` servent à préparer une nouvelle version.

Elles sont créées depuis `develop` lorsque les fonctionnalités prévues sont terminées.

Elles permettent uniquement :

* la correction de bugs ;
* les ajustements mineurs ;
* la mise à jour du numéro de version ;
* la documentation de livraison ;
* les tests finaux.

Une fois validée, la branche `release/*` est fusionnée dans `main`, puis dans `develop`.

Exemple :

```text
release/1.2.0
```

### `hotfix/*`

Les branches `hotfix/*` servent à corriger rapidement un problème critique présent en production.

Elles sont créées depuis `main`.

Une fois la correction terminée, la branche est fusionnée dans `main`, puis dans `develop`.

Exemple :

```text
hotfix/1.2.1
```

## Flux standard

```text
specs      -> develop

feature/*  -> develop
bugfix/*   -> develop

develop    -> release/*
release/*  -> main
release/*  -> develop

hotfix/*   -> main
hotfix/*   -> develop
```

## Règles du dépôt

* Aucun commit direct sur `main`.
* Aucun commit direct sur `develop`, sauf exception validée.
* Le cahier des charges est maintenu sur `specs`.
* Une modification majeure des besoins doit d’abord être formalisée dans `specs`.
* Une fonctionnalité part toujours de `develop`.
* Une fonctionnalité doit correspondre à un besoin ou une spécification validée.
* Une release part toujours de `develop`.
* Un hotfix part toujours de `main`.
* Toute fusion vers `main` doit correspondre à une version livrable.
* Toute version livrée sur `main` doit être taguée.
* Les corrections faites dans une `release/*` ou un `hotfix/*` doivent aussi être réintégrées dans `develop`.
* Les documents de spécification validés doivent être synchronisés avec `develop`.

## Exemple de cycle complet

```text
specs
  └── rédaction ou mise à jour du cahier des charges
        └── merge vers develop après validation

develop
  └── feature/ma-fonction
        └── merge vers develop

develop
  └── release/1.0.0
        ├── merge vers main
        ├── tag v1.0.0
        └── merge vers develop

main
  └── hotfix/1.0.1
        ├── merge vers main
        ├── tag v1.0.1
        └── merge vers develop
```

## Convention de nommage

```text
specs
feature/nom-fonctionnalite
bugfix/nom-correction
release/x.y.z
hotfix/x.y.z
```

Exemples :

```text
feature/authentification
bugfix/correction-affichage-date
release/1.0.0
hotfix/1.0.1
```
