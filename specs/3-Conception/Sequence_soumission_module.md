# Séquence — Composition et soumission d'un module (D5/D6)

> Phase 3 — Arrington | Use cases : UCMOD01, UCMOD02, UCMOD06, UCAM01, UCAM07, UCAM08 | Domaine : `domain/model`

---

## 1. Objectif

Diagramme de séquence couvrant le cycle complet d'un module : création (`draft`, RM16), composition (instances + liaisons, mutable et locale), et soumission (`SubmitModule`, RM17/RM18). Complète `Architecture_Composition.md` §3 (diagramme d'états) et §9 (ADR-05).

---

## 2. Création et composition (état `draft`, entièrement local)

```plantuml
@startuml
actor "Concepteur" as C
participant "adapter in\n(CLI ou REST)" as In
participant "ModelService\n(domain/model)" as Svc
participant "ConnectionStore\n(adapters/out/localstorage)" as ConnS
participant "InterfaceStore" as IfaceS

C -> In : myr module create --name <nom>\nPOST /api/modules
In -> Svc : CreateModule(req ModuleRequest)
Svc -> Svc : Status = draft (RM16)
Svc --> In : *Model3D (module, draft, Assemblies: [])
In --> C : module créé

C -> In : myr model instance add <moduleID> <assetID>\nPOST /api/modules/:id/instances
In -> Svc : AddAssetToWorkspace(moduleID, assetID)
Svc -> Svc : créer WorkspaceInstance\n(indépendante — RM15)
Svc --> In : *Model3D (WorkspaceInstances += 1)
In --> C : instance ajoutée

C -> In : myr model link add\nPOST /api/connections
In -> Svc : AddAssemblyLink(fromIfaceID, toIfaceID, ...)
Svc -> IfaceS : GetInterface(fromIfaceID), GetInterface(toIfaceID)
IfaceS --> Svc : AssetInterface x2
Svc -> Svc : ifacesCompatible()\n(RM10 — vérification automatique,\n5 critères RM11, voir Architecture_Composition.md §5)
alt interfaces compatibles
  Svc -> Svc : vérifier usage unique (RM09)
  Svc -> ConnS : SaveConnection(Connection{Incompatible: false})
else interfaces incompatibles
  Svc -> ConnS : SaveConnection(Connection{Incompatible: true})
  note right: RM12 — jamais supprimée\nautomatiquement
end
ConnS --> Svc : ok
Svc --> In : *Connection
In --> C : liaison créée

C -> In : myr model instance remove <moduleID> <instanceID>\nDELETE /api/modules/:id/instances/:instanceID
In -> Svc : RemoveAssetFromWorkspace(moduleID, instanceID)
Svc -> ConnS : ListConnections() puis suppression en cascade\nde toutes les Connection référençant instanceID
note right: RM14 — cascade obligatoire
ConnS --> Svc : ok
Svc --> In : *Model3D (WorkspaceInstances -= 1)
In --> C : instance retirée
@enduml
```

---

## 3. Soumission (`SubmitModule` — RM17/RM18, transaction unique)

```plantuml
@startuml
actor "Concepteur" as C
participant "adapter in\n(CLI ou REST)" as In
participant "ModelService" as Svc
participant "ConnectionStore" as ConnS
participant "BlockchainPort\n(adapters/out/fabric)" as BC

C -> In : myr module submit <moduleID> [--note]\nPOST /api/modules/:id/submit
In -> Svc : SubmitModule(moduleID, note)

Svc -> Svc : len(Assemblies) > 0 ?\n(RM17)
alt Assemblies vide
  Svc --> In : erreur RM17
  In --> C : 4xx — module vide, soumission refusée
else Assemblies non vide
  Svc -> ConnS : ListConnections() (snapshot des liaisons du module)
  ConnS --> Svc : []Connection
  Svc -> Svc : construire ModuleVersion{\n  Assemblies: [...connID],\n  Hash: hash(snapshot),\n  CreatedAt: now}
  note over Svc
    RM18 — ModuleVersion immuable,
    horodatée, hashée
  end note
  Svc -> Svc : embarquer Interfaces des composants\ndu module (ADR-02, comme un asset standard)
  Svc -> BC : StoreModelRecord(Model3D{\n  Status: submitted,\n  ModuleVersions: [...prev, newVersion]})
  BC --> Svc : BlockID
  Svc --> In : *Model3D (submitted)
  In --> C : module soumis — immuable (RM19)
end
@enduml
```

---

## 4. Notes

- La composition (§2) reste entièrement locale et mutable — aucune transaction Fabric n'est déclenchée avant `SubmitModule` (ADR-05). C'est ce qui garantit les exigences de temps de réponse UCAM (`< 200 ms` / `< 300 ms`, `Conception_intro.md` §6 ADR-02).
- Une fois `submitted`, toute modification du module doit passer par un fork (RM19) — état de cette garde : `specs/roadmap_dev.md` § Écarts structurels — modèle & chaincode, E5. Ce diagramme ne représente pas le chemin de fork, qui reste à concevoir au niveau service.
- Le retrait en cascade (§2, dernier bloc) est la seule opération de composition qui modifie un état déjà persisté localement (les `Connection` liées) — elle reste néanmoins purement locale tant que le module n'est pas soumis.
