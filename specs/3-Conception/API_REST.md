# API REST — Contrat d'interface

> Phase 3 — Arrington | Implémentation : `adapters/in/rest/`

---

## 1. Principes

- **Base URL :** `http(s)://<host>:<port>/api/`
- **Format :** JSON — `Content-Type: application/json`
- **Auth :** token opaque de session — en-tête `X-Myr-Token: <token>` ( voir `specs/3-Conception/DC_D1_Auth_Identity.md`)
- **Erreurs :** `{ "error": "<code>", "message": "<description>" }`
- **Succès :** objet ou tableau JSON directement (pas d'enveloppe `{ data: ... }`)

### Niveaux d'accès

| Niveau | Middleware | Condition |
|--------|-----------|-----------|
| Public | aucun | Toujours accessible |
| Auth | `requireAuth` | Token de session valide requis (`X-Myr-Token`) |
| Write | `requireRole(rbac.PermWrite)` | Session valide + permission `write` (rôle `contributor` par défaut) |
| Admin | `requireRole(rbac.PermAdmin)` | Session valide + permission `admin` uniquement |

> Pour les routes mixtes (lecture libre, écriture protégée), le middleware vérifie la permission uniquement sur `POST`, `PUT`, `PATCH`, `DELETE`.

**Codes d'erreur identité/session :**

| Code HTTP | Cas |
|-----------|-----|
| 400 | Corps invalide, champs requis manquants |
| 401 | Secret d'enrôlement CA invalide, token de session absent/expiré |
| 403 | Permission insuffisante, accès invité refusé (réseau privé) |
| 429 | Trop de tentatives (`authLimiter`, 10/min/IP) |

---

## 2. D1 — Identité & RBAC

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/identity/policy` | Public | Politique du réseau actif (AllowAutoGuest, AllowAutoRegister) |
| POST | `/api/identity/enroll` | Public | Enrôlement Fabric CA avec secret d'enrollment (sans créer de session REST) |
| POST | `/api/identity/session` | Public | (Ré-)enrôlement + création d'une session REST (token opaque) |
| POST | `/api/identity/guest` | Public | Token `reader` automatique (si `AllowAutoGuest=true`) |
| POST | `/api/identity/request` | Public | Demande d'accès (auto-enregistrement si `AllowAutoRegister=true`, sinon mise en attente `pending` jusqu'à `POST /api/identity/requests/{id}/approve`) |
| GET | `/api/identity/requests` | Admin | Liste des demandes de compte en attente — expose des données personnelles (email, message libre), réservée à `identity.admin` |
| POST | `/api/identity/requests/{id}/approve` | Admin | Approuver une demande `pending` — transforme la demande en identité CA active (UCA01) |
| GET | `/api/identity/wallets` | Auth | Wallets locaux — le répertoire de wallets est partagé par tous les utilisateurs enrôlés sur le nœud : une session sans `identity.admin` ne voit que son propre wallet (filtré par pseudo), jamais celui des autres |
| GET | `/api/identity/status` | Auth | Statut CA d'un wallet (`?handle=pseudo@org`) |

> Pas de CRUD REST pour le RBAC (`domain/role`) — gestion des rôles CLI uniquement (`myr role ...`). Le REST ne fait que consulter les permissions via `requireRole`.

---

## 3. D2 — Administration

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/channels` | Public (lecture) | Liste les canaux Fabric disponibles |
| PUT | `/api/channels` | Auth | Mettre à jour la config canal |
| GET | `/api/networks` | Public | Liste des profils réseau configurés |
| GET/PUT | `/api/networks/active` | Auth | Réseau actif de la session |
| GET | `/api/admin/sessions` | Admin | Sessions actives (identifiant permettant de cibler `DELETE /api/admin/sessions/{token}` — voir `specs/roadmap_dev.md` § Écarts Identité & Session, E4 pour le point ouvert sur la forme de cet identifiant) |
| DELETE | `/api/admin/sessions/{token}` | Admin | Invalider une session (token complet requis) |

> **Ajout d'organisation, de nœud (UCADM01/03/04) :** exposés en **CLI uniquement** (`myr org add`, `myr node add/remove`) — l'API REST n'expose jamais ces opérations d'infrastructure (voir `DC_CLI_Admin.md`).

---

## 4. D3/D4 — Composants

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/components` | Public (DC-D1-06) | Liste des composants (filtres: name, category, tags, channel) |
| POST | `/api/components` | Contributor | Créer un composant (`AddFull`) |
| GET | `/api/components/{id}` | Auth | Détail d'un composant |
| PATCH | `/api/components/{id}` | Contributor | Modifier (patch : name, description, tags, links, licenseID) — #incoherence la méthode documentée ici était PUT, mais le handler (`adapters/in/rest/handlers.go`, `patchComponent`) répond uniquement à PATCH ; corrigé pour refléter le comportement réel, à confirmer que PUT n'était pas l'intention initiale |
| DELETE | `/api/components/{id}` | Contributor | Supprimer (local uniquement — Fabric immuable) |
| GET | `/api/components/{id}/interfaces` | Auth | Interfaces physiques d'un composant |
| GET | `/api/components/{id}/compatible` | Auth | Composants compatibles (type d'interface, catégorie, tag, sens — `DC_D8_Recherche.md` §2, UCREC02) |
| GET | `/api/components/{id}/versions` | Auth | Arbre de versions / historique de dérivation (`DC_D8_Recherche.md` §3, UCREC03) |

> **Accès visiteur (EF17, UCCL01) :** seule la liste (`GET /api/components`) est concernée par l'accès public — voir `DC_D1_Auth_Identity.md` DC-D1-06 et l'incohérence relevée avec `GET /api/modules` (§6 ci-dessous) qui reste `Auth`.

---

## 5. D5 — Connexions et Interfaces (Composition de Module)

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/connections` | Auth | Liste des connexions |
| POST | `/api/connections` | Contributor | Créer une connexion simple |
| DELETE | `/api/connections/{id}` | Contributor | Supprimer une connexion |
| POST | `/api/assembly-links` | Contributor | Créer une liaison interface→interface |
| POST | `/api/virtual-connect` | Contributor | Relier interface virtuelle → physique |
| GET | `/api/interfaces/{id}` | Auth | Détail d'une interface |
| PUT | `/api/interfaces/{id}` | Contributor | Modifier une interface |
| DELETE | `/api/interfaces/{id}` | Contributor | Supprimer une interface |
| GET | `/api/refs` | Auth | Vocabulaire de référence (catégories, types, unités) |
| POST | `/api/refs/categories` | Contributor | Ajouter une catégorie |
| POST | `/api/refs/types` | Contributor | Ajouter un type |
| POST | `/api/refs/units` | Contributor | Ajouter une unité |

---

## 6. D6 — Modules

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/modules` | Auth | Liste des modules |
| POST | `/api/modules` | Contributor | Créer un module (draft) |
| GET | `/api/modules/{id}` | Auth | Détail d'un module |
| PUT | `/api/modules/{id}` | Contributor | Modifier un module (patch) — #incoherence aucune méthode de mise à jour n'est câblée pour cette route dans `handleModule` (`adapters/in/rest/handlers.go`) : seuls GET et DELETE y répondent aujourd'hui, PUT/PATCH renvoient 405 |
| DELETE | `/api/modules/{id}` | Contributor | Supprimer un module (draft uniquement) |
| GET | `/api/modules/{id}/instances` | Auth | Instances d'un module |
| POST | `/api/modules/{id}/instances` | Contributor | Ajouter un composant comme instance |
| DELETE | `/api/modules/{id}/instances/{instanceId}` | Contributor | Retirer une instance (+ cascade connexions) |
| POST | `/api/modules/{id}/submit` | Contributor | Soumettre à la blockchain |
| GET | `/api/modules/{id}/interfaces` | Auth | Interfaces exposées du module |
| GET | `/api/modules/{id}/assemblies` | Auth | Connexions internes du module |
| GET | `/api/modules` (`?uses_component={id}`) | Auth | Modules qui intègrent un composant donné (`DC_D8_Recherche.md` §4, UCREC04) |
| GET | `/api/modules/{id}/bom` | Auth | Export BOM (Bill of Materials) du module (`DC_D8_Recherche.md` §5, UCREC05) |

> **Accès visiteur :** `GET /api/modules` reste `Auth`, sous réserve de la configuration réseau visiteur (`UCMOD04.md` ligne 134) — question ouverte pour le PO, non tranchée ici (voir `DC_D1_Auth_Identity.md` DC-D1-06).

---

## 7. Licences

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/licenses` | Auth | Catalogue des licences |
| GET | `/api/licenses/{id}` | Auth | Détail d'une licence |
| POST | `/api/licenses/check` | Auth | Vérifie la compatibilité `{ parent_license_id, proposed_license_id }` (règle 8) — #remarque route implémentée (`handleLicense`) mais absente de ce tableau jusqu'ici |
| POST | `/api/licenses/check-product` | Auth | Vérifie la compatibilité d'un module assemblant plusieurs composants `{ component_license_ids[], proposed_module_license_id }` (règle 8) — #remarque idem |

---

## 8. Miniatures

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| POST | `/api/components/{id}/thumbnail` | Contributor | Sauvegarder une miniature STL (data URL base64) — #incoherence route absente du routeur (`server.go`) et du handler (`handleComponent` ne reconnaît que les suffixes `/interfaces`, `/tree` et `/thumbnail/regenerate`) ; la miniature d'un composant se fixe uniquement au moment de sa création via le champ `thumbnail` du formulaire multipart de `POST /api/components` |
| GET | `/api/components/{id}/thumbnail` | Auth | Récupérer la miniature — #incoherence idem, non routée ; la miniature est aujourd'hui exposée via le champ `thumbnail` de `componentDTO` (`GET /api/components`, `GET /api/components/{id}`) |
| POST | `/api/modules/{id}/thumbnail` | Contributor | Sauvegarder la miniature d'un module (data URL base64) — implémentée, absente jusqu'ici de ce tableau |
| GET | `/api/modules/{id}/thumbnail` | Auth | Récupérer la miniature d'un module — implémentée, absente jusqu'ici de ce tableau |
| POST | `/api/components/{id}/thumbnail/regenerate` | Contributor | Redériver la miniature depuis la seule source durable accessible côté serveur : l'og:image du premier lien externe enregistré (`Links`) — un modèle 3D sans lien n'a pas de source régénérable côté serveur (le rendu 3D est produit par le client GUI, pas par `myr`, voir § Séparation des dépôts) ; échoue en 500 (`ErrNoThumbnailSource`) si l'asset n'a aucun lien |
| POST | `/api/modules/{id}/thumbnail/regenerate` | Contributor | Même comportement que ci-dessus, pour un module |

---

## 9. Infrastructure

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/ping` | Public | Liveness (sans appel Fabric) |
| GET | `/api/status` | Public | Statut serveur |
| GET | `/api/health` | Public | Health check (load balancer) |
| GET | `/metrics` | Public | Métriques Prometheus |

---

## 10. Headers de sécurité (appliqués à toutes les réponses)

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'none'
```

> `myr` sert exclusivement du JSON (pas de GUI embarquée) — voir `specs/3-Conception/Securite.md`.

---

## 11. D7 — Paiement

> Détail des entités (`Order`, `AssetPrice`) et règles métier (RM23, RM24, RM30–RM33) dans `DC_D7_Payment.md`.

| Méthode | Route | Domaine | UC |
|---------|-------|---------|-----|
| POST | `/api/orders` | D7 | UCPI01 |
| GET | `/api/orders/{id}` | D7 | UCPI01 |
| POST | `/api/orders/{id}/deliver` | D7 | UCAUT01 (`DC_D9_Automatisation.md` §2) |
| GET/POST | `/api/assets/{id}/price` | D7 | UCPI04 |
| POST | `/api/assets/{id}/transfer` | D7 | UCPI07 |
| POST | `/api/assets/{id}/clone` | D7 | UCPI08 |

## 12. D9 — Automatisation avancée

> Détail dans `DC_D9_Automatisation.md` §5. Le plugin CAO (UCAUT03) n'a pas de route dédiée — il consomme `POST /api/components` comme tout client API (DC-D9-02).

| Méthode | Route | Domaine | UC |
|---------|-------|---------|-----|
| POST | `/api/components/{id}/versions` | D9 | UCAUT04 |
| GET | `/api/components/{id}/versions/diff` | D9 | UCAUT04 |

---

## 13. Spec OpenAPI générée automatiquement

Les tableaux ci-dessus décrivent le contrat cible de l'API (y compris des routes non encore câblées, ex. §11 D7 Paiement — voir § « CLAUDE.md n'est pas une spec » sur la distinction attendu / avancement). En complément, une **spec OpenAPI 3 est générée automatiquement à partir des commentaires Go** au-dessus de chaque handler REST — elle documente précisément ce que le serveur expose **réellement** à un instant donné (paramètres, schémas de requête/réponse, codes HTTP, exigence d'authentification), pour tout développeur tiers qui construit un logiciel consommant l'API (GUI externe, plugin CAO, script d'intégration).

### Mécanisme

- **Outil :** `swaggo/swag` — parse des commentaires `@Summary`, `@Param`, `@Success`, `@Router`, etc. placés directement au-dessus de chaque fonction handler dans `adapters/in/rest/*.go`, ainsi que le bloc d'annotations générales (`@title`, `@BasePath`, `@securityDefinitions`) au-dessus de `func main()` dans `cmd/api/main.go`.
- **Aucune dépendance ajoutée à `go.mod`/`vendor/` :** le CLI `swag` s'exécute via `go run github.com/swaggo/swag/cmd/swag@<version>` — un outil de développement au même titre que `gofmt`, jamais compilé dans le binaire serveur. La génération produit uniquement `api/swagger.json` et `api/swagger.yaml` (flag `--outputTypes json,yaml` — pas de fichier Go `docs.go`, qui obligerait sinon à importer le package `swaggo/swag` au runtime).
- **Commande :** `make docs-api` (cible du `Makefile`, version de l'outil pinée dans la variable `SWAG_VERSION`).
- **Authentification documentée :** en-tête `X-Myr-Token` (schéma `apiKey`, nommé `MyrToken`) — jamais de JWT (voir règle 23 du domaine identité).

### Convention d'annotation

- Chaque fonction handler qui répond directement à une requête HTTP porte un bloc `// @Summary ... // @Router /chemin [méthode]` juste au-dessus de sa déclaration — sans ligne vide entre le commentaire et le `func`, faute de quoi `swag` ne l'associe pas à la fonction.
- Les corps de requête décrits par une struct Go nommée au niveau paquet (ex. `connectionDTO`, `assemblyLinkRequest`, `walletDTO`) sont référencés par leur type — le schéma JSON exact apparaît dans la spec générée. Les corps décrits par une struct anonyme locale à la fonction (ex. `var body struct{...}` dans `patchComponent`) ne sont pas nommables par `swag` : ils apparaissent dans la spec comme un `object` générique, le détail des champs restant uniquement dans la description textuelle du commentaire — limite connue de l'outil, pas du domaine.
- **Cas des handlers multi-routes** (ex. `handleModule`, qui route une dizaine de sous-ressources d'un module derrière un seul point d'entrée Go) : `swag` associe un commentaire à une fonction, pas à une route — un seul bloc `@Description`/`@Success` couvre alors plusieurs lignes `@Router`, avec une précision par sous-route moindre que pour les handlers dédiés à une seule route. Diviser ces fonctions uniquement pour affiner la documentation n'a pas été fait ici : cela reviendrait à refactorer du code qui fonctionne pour un besoin de documentation, hors du périmètre demandé — un futur découpage de ces handlers (s'il est motivé par autre chose que la doc) affinera la spec générée sans changement de convention.

### Autorité en cas de divergence

En cas d'écart entre les tableaux `§1`–`§12` de ce document et `api/swagger.json`/`api/swagger.yaml` : la spec générée fait foi de l'implémentation réelle (elle est dérivée du code), les tableaux ci-dessus font foi de l'intention cible. Les écarts déjà identifiés lors de la mise en place de la génération automatique sont marqués `#incoherence` / `#remarque` dans ce document (§4, §6, §7, §8) — à trancher par le product owner.

### Régénération

`api/swagger.json`/`.yaml` ne sont pas régénérés automatiquement à chaque modification d'un handler — exécuter `make docs-api` après tout changement de signature de route (nouveau paramètre, nouveau code retour, nouvelle route). Aucune vérification CI ne détecte aujourd'hui une spec désynchronisée du code (`make ci` ne couvre que ENF18/ENF25) ; l'ajout d'un tel contrôle est une évolution possible, non traitée ici faute de validation explicite.
