---
categorie: Composant Ecriture
titre: "Ajout d'un composant Physique"
probabilite: 3
impact: 5
importance: 15
etat: relire
---

# Ajout d'un composant Physique

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Concepteur" as C

rectangle "Application MYR" {
    usecase "Ajouter un composant physique" as UC1
    usecase "Vérifier le hash" as UC2
    usecase "Analyser similarité SCM" as UC3
    usecase "Enregistrer sur la blockchain" as UC4
}

C --> UC1
UC1 ..> UC2 : <<include>>
UC1 ..> UC3 : <<include>>
UC1 ..> UC4 : <<include>>

@enduml
```

## Contexte

Ajout d'une création de l'utilisateur sous forme de composant physique (pièce CAO, produit matériel...) dans le réseau MYR.

## Pré-conditions

- Être connecté au réseau
- Avoir les droits de création de composants
- Disposer d'un fichier 3D/CAO du composant

## Scénario

**Étape initiale :** `myr model add roue.stl --name "Roue avant" --channel greenchannel --category base` est exécutée (ou l'appel API équivalent), pour le compte du Concepteur

### Flux nominal — Composant nouveau

1. Le fichier 3D est importé
2. Le système vérifie le hash du fichier : aucun doublon détecté
3. L'analyse SCM de similarité est effectuée (seuil 50%)
4. Les métadonnées sont renseignées (nom, licence, auteur)
5. La transaction "fromScratch" est soumise sur la blockchain
6. Un UUID est créé et attribué au composant

### Flux alternatif — Import depuis un format CAO non natif (STL, STEP, OBJ)

1. Le fichier est fourni dans un format de CAO non natif (STL, STEP, OBJ…)
2. Le système accepte le fichier et calcule son hash normalement
3. Les métadonnées géométriques (dimensions, volume) sont extraites automatiquement selon le format
4. Les métadonnées non extractibles automatiquement (description, licence) sont renseignées explicitement
5. La transaction est soumise normalement

### Flux erreur — Hash déjà existant

1. Erreur métier : "Composant déjà existant - risque de plagiat" — invite à contacter l'administration

### Flux erreur — Similarité supérieure à 50%

1. Erreur métier : "Similarité trop élevée avec un composant existant" — invite à contacter l'administration

## Post-conditions

- Le composant physique est enregistré sur la blockchain
- Un UUID unique lui est attribué
- Le droit d'auteur est enregistré

## Diagrammes

### Flux d'ajout d'un asset depuis zéro (fromScratch)

```plantuml
@startuml
skin rose
title Ajout d'un asset sur BASE
start
:Transmettre la demande d'ajout d'asset {type} (myr model add);
if(droits Utilisateur OK?) then (oui)
  :Lecture uuid;
    if (uuid existant?) then (oui)
      :Creation nouvel uuid;
      :Transaction asset {type};
    else (non)
      :Analyse hash asset;
        if (hash existant?) then (oui)
          :Erreur plagiat: Contact administration;
        else (non)
          :Analyse SCM Simulitude;
          if (Similitude > 50%) then (oui)
            :Erreur plagiat: Contact administration;
          else (non)
            :Creation nouvel uuid;
            :Transaction asset 'fromScratch';
          endif
        endif
    endif
else (non)
  :Message : L'utilisateur n'a pas les droits;
endif
stop
@enduml
```

### Taxonomie — Détermination du type d'un nouvel asset

```plantuml
@startuml
skin rose
title définition du type d'un ajout
start
if (Conversion composant en module?) then (oui)
  :est un DÉCOUPAGE;
else (non)
if (Nouvelle piece?) then (oui)
  if(Sur base?) then (oui)
    if(est complementaire?) then (oui)
      :est une EXTENSION;
    elseif(dimension differente?) then (oui)
      :est une ADAPTATION;
    else (non)
      :est une VARIATION;
    endif
  else (non)
    if(est Plagié) then (oui)
      end;
    else (non)
      :est une BASE;
    endif
  endif
else (non)
  if(fonctionnalité ajouté?) then (oui)
    :est une DERIVATION;
  elseif (fonctionnalité supprimée) then (oui)
    :est une REGRESSION;
  else (non)
    :est une AMELIORATION;
  endif
endif
endif

:ajout de la pièce;
stop
@enduml
```
