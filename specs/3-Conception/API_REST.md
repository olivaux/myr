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
| GET | `/api/identity/requests` | Auth | Liste des demandes de compte en attente |
| POST | `/api/identity/requests/{id}/approve` | Admin | Approuver une demande `pending` — transforme la demande en identité CA active (UCA01) |
| GET | `/api/identity/wallets` | Auth | Wallets locaux |
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
| PUT | `/api/components/{id}` | Contributor | Modifier (patch : name, description, tags, links, licenseID) |
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
| PUT | `/api/modules/{id}` | Contributor | Modifier un module (patch) |
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

---

## 8. Miniatures

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| POST | `/api/components/{id}/thumbnail` | Contributor | Sauvegarder une miniature STL (data URL base64) |
| GET | `/api/components/{id}/thumbnail` | Auth | Récupérer la miniature |

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
