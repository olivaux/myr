# Séquence — Soumission d'un composant (D3)

> Phase 3 — Arrington | Use cases : UCCE01, UCCE06 | Domaine : `domain/model`

---

## 1. Objectif

Diagramme de séquence détaillant les deux chemins de soumission d'un composant définis par ADR-02 (`Conception_intro.md` §6) et repris dans `Architecture_Composition.md` §3 : la création directe (comportement par défaut, une seule transaction Fabric) et le cycle brouillon → soumission explicite (`draft: true`).

---

## 2. Chemin par défaut — création directe (`AddFull`)

```plantuml
@startuml
actor Client as C
participant "adapter in\n(CLI ou REST)" as In
participant "ModelService\n(domain/model)" as Svc
participant "PlagiarismChecker\n(port out, contrat §6\nArchitecture_Composition.md)" as Plag
participant "FileStoragePort\n(adapters/out/ipfs)" as FS
participant "BlockchainPort\n(adapters/out/fabric)" as BC

C -> In : myr model add <fichier>\nPOST /api/components
In -> Svc : AddFull(req AddRequest)

Svc -> Svc : valider métadonnées\n(RM04 : ID généré serveur,\nRM05 : ParentID si Category != base)
Svc -> Svc : calculer SHA-256 du fichier

alt Category == base
  Svc -> BC : ListModelRecords(channelID)\n(comparaison unicité SHA-256)
  BC --> Svc : []Model3D existants
  Svc -> Svc : comparer Hash (écart E4 —\nnon implémenté actuellement)
  Svc -> Plag : CompareStructural(fileA, fileB)\npour chaque asset existant
  Plag --> Svc : score similarité
  alt score > 0.50 (un seul asset)
    Svc --> In : erreur RM01 (similarité SCM)
    In --> C : 4xx / message erreur
  end
end

alt ParentID != "" && LicenseID != ""
  Svc -> Svc : CheckLicenseCompatibility\n(RM03)
end

Svc -> FS : Upload(filePath)
FS --> Svc : hash (CID)

note over Svc
  Toutes les données (métadonnées,
  licences, interfaces, UUID) validées
  AVANT soumission (RM07)
end note

Svc -> BC : StoreModelRecord(Model3D{Status: submitted})
BC --> Svc : BlockID (Fabric txID)

Svc --> In : *Model3D (submitted)
In --> C : 201 Created / composant créé
@enduml
```

---

## 3. Chemin brouillon — `draft: true` puis `Submit` (E8)

```plantuml
@startuml
actor C as "Concepteur"
participant "adapter in\n(CLI ou REST)" as In
participant "ModelService\n(domain/model)" as Svc
participant "InterfaceStore\n(adapters/out/localstorage)" as IS
participant "FileStoragePort" as FS
participant "BlockchainPort" as BC

C -> In : myr model add <fichier> --draft\nPOST /api/components {draft:true}
In -> Svc : AddFull(req AddRequest{Draft:true})
Svc -> Svc : valider + hash (comme §2, sans anti-plagiat\ntant que Category != base soumis)
Svc -> FS : Upload(filePath)
FS --> Svc : hash
Svc --> In : *Model3D (Status: draft, BlockID: "")
In --> C : composant créé en brouillon

... édition locale, aucun coût blockchain ...

C -> In : myr model interface add <assetID>\nPOST /api/components/:id/interfaces
In -> Svc : AddInterface(iface)
Svc -> IS : SaveInterface(iface)
IS --> Svc : ok
Svc --> In : ok
In --> C : interface ajoutée (locale)

note over C, IS
  Répétable librement (UCCE06, UCAM03) :
  AddInterface / UpdateInterface / RemoveInterface
  / EnsureVirtualSlot — sans transaction Fabric
  tant que Status == draft (ADR-02)
end note

C -> In : myr model submit <assetID>\nPOST /api/components/:id/submit
In -> Svc : Submit(id)\n(cible E8 — généralise SubmitModule)
Svc -> IS : ListInterfacesForAsset(id)
IS --> Svc : []AssetInterface
Svc -> Svc : embarquer Interfaces dans Model3D\nvalidation finale (RM07)
Svc -> BC : StoreModelRecord(Model3D{Status: submitted, Interfaces: [...]})
BC --> Svc : BlockID
Svc --> In : *Model3D (submitted)
In --> C : composant soumis — immuable (RM19)
@enduml
```

---

## 4. Notes

- Les deux chemins convergent : une seule écriture Fabric (`StoreModelRecord`) commet l'intégralité du brouillon, `Interfaces` compris — jamais d'écriture blockchain par édition individuelle (ADR-02, `Conception_intro.md` §6).
- Le contrat `PlagiarismChecker.CompareStructural` (§6 `Architecture_Composition.md`) est représenté ci-dessus pour situer où l'algorithme SCM s'intégrerait une fois choisi — l'algorithme lui-même reste une question ouverte pour le PO, non tranchée par ce diagramme.
- `Submit(id)` (chemin §3) n'a pas de méthode `ModelService` dédiée : ce diagramme documente la cible — état de cet écart : `specs/roadmap_dev.md` § Écarts structurels — modèle & chaincode, E8.
