---
categorie: Propriété Intellectuelle
titre: "Transfert de propriété intellectuelle"
probabilite: 1
impact: 3
importance: 3
etat: relire
---

# Transfert de propriété intellectuelle

## Diagramme d'acteurs

```plantuml
@startuml
left to right direction

actor "Propriétaire" as P
actor "Destinataire" as DEST

rectangle "Application MYR" {
    usecase "Transférer la propriété d'un asset" as UC1
    usecase "Accepter le transfert" as UC2
    usecase "Enregistrer sur la blockchain" as UC3
}

P --> UC1
DEST --> UC2
UC1 ..> UC2 : <<include>>
UC2 ..> UC3 : <<include>>

@enduml
```

## Contexte

Un propriétaire peut transférer la propriété intellectuelle d'un composant ou module à un autre utilisateur ou organisation.

## Pré-conditions

- Être connecté au réseau
- Être propriétaire du composant/module à transférer

## Scénario

**Étape initiale :** Le propriétaire accède à son asset et choisit "Transférer la propriété"

### Flux nominal — Transfert réussi

1. Le propriétaire renseigne l'identifiant du destinataire
2. Il confirme le transfert
3. La transaction de transfert est soumise sur la blockchain
4. Le destinataire reçoit une notification et accepte le transfert

### Flux alternatif — Transfert vers un réseau externe

1. L'identifiant du destinataire appartient à un utilisateur sur un réseau externe
2. Le système génère un token de transfert signé cryptographiquement
3. Le destinataire est notifié sur son réseau et accepte le transfert via le token
4. La propriété est transférée avec enregistrement de la transaction sur les deux réseaux

## Post-conditions

- La propriété est transférée et enregistrée de manière immuable sur la blockchain
- L'ancien propriétaire perd les droits d'édition

## Diagramme d'activités

```plantuml
@startuml
skin rose
title Transfert de propriété intellectuelle
start
:Accéder à l'asset et choisir "Transférer la propriété";
:Renseigner l'identifiant du destinataire;
:Confirmer le transfert;
if (Destinataire sur un réseau externe?) then (oui)
  :Générer un token de transfert signé cryptographiquement;
  :Notifier le destinataire sur son réseau avec le token;
  :Le destinataire accepte via le token;
  :Enregistrer la transaction sur les deux réseaux;
  stop
else (non)
  :Soumettre la transaction de transfert sur la blockchain;
  :Notifier le destinataire;
  :Le destinataire accepte le transfert;
  stop
endif
@enduml
```
