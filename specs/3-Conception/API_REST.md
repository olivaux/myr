# API REST — Contrat d'interface

> Phase 3 — Arrington | Implémentation : `adapters/in/rest/`

---

## 1. Principes

- **Base URL :** `http(s)://<host>:<port>/api/`
- **Format :** JSON — `Content-Type: application/json`
- **Auth :** Bearer JWT — `Authorization: Bearer <token>`
- **Erreurs :** `{ "error": "<code>", "message": "<description>" }`
- **Succès :** objet ou tableau JSON directement (pas d'enveloppe `{ data: ... }`)

### Niveaux d'accès

| Niveau | Middleware | Condition |
|--------|-----------|-----------|
| Public | aucun | Toujours accessible |
| Auth | `requireAuth` | JWT valide requis |
| Contributor | `requireRole("contributor")` | JWT + rôle `contributor` ou `admin` |
| Admin | `requireRole("admin")` | JWT + rôle `admin` uniquement |

> Pour les routes mixtes (lecture libre, écriture protégée), le middleware vérifie le rôle uniquement sur `POST`, `PUT`, `PATCH`, `DELETE`.

---

## 2. D1 — Compte & Accès (Auth JWT)

| Méthode | Route | Accès | Corps | Réponse |
|---------|-------|-------|-------|---------|
| POST | `/api/auth/register` | Public | `{email, password, displayName, orgID?}` | `{accessToken, refreshToken, user}` |
| POST | `/api/auth/login` | Public | `{email, password}` | `{accessToken, refreshToken, user}` |
| POST | `/api/auth/refresh` | Public | `{refreshToken}` | `{accessToken}` |
| POST | `/api/auth/logout` | Public | `{refreshToken}` | `204` |
| GET | `/api/auth/me` | Auth | — | `{user}` |
| POST | `/api/auth/enroll-wallet` | Auth | `{name, secret, orgID}` | `{wallet}` |

**Codes d'erreur auth :**

| Code HTTP | Cas |
|-----------|-----|
| 400 | Format invalide, email déjà pris |
| 401 | Identifiants incorrects, token expiré |
| 403 | Compte suspendu |

---

## 3. D1 — Identité Blockchain

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/identity/policy` | Public | Politique du réseau actif (AllowAutoGuest, AllowAutoRegister) |
| POST | `/api/identity/enroll` | Public | Enrôlement Fabric CA avec secret d'enrollment |
| GET/POST | `/api/identity/session` | Public | Session locale active |
| POST | `/api/identity/guest` | Public | Token Lecteur automatique (si AllowAutoGuest=true) |
| POST | `/api/identity/request` | Public | Demande de compte (si AllowAutoRegister=false → mise en attente admin) |
| GET | `/api/identity/requests` | Admin | Liste des demandes de compte en attente |
| GET | `/api/identity/wallets` | Auth | Wallets de l'utilisateur connecté |
| GET | `/api/identity/status` | Auth | Statut de l'identité Fabric CA |

---

## 4. D2 — Administration

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/channels` | Public (lecture) | Liste les canaux Fabric disponibles |
| PUT | `/api/channels` | Auth | Mettre à jour la config canal |
| GET | `/api/networks` | Public | Liste des profils réseau configurés |
| GET/PUT | `/api/networks/active` | Auth | Réseau actif de la session |
| GET/POST | `/api/admin/users` | Admin | Gestion des utilisateurs |
| GET/PUT/DELETE | `/api/admin/users/{id}` | Admin | Utilisateur individuel |
| GET | `/api/admin/sessions` | Admin | Sessions actives |
| DELETE | `/api/admin/sessions/{id}` | Admin | Invalider une session |

> **Non implémentés (UCADM01-03) :** endpoints d'ajout d'organisation, création de canal, ajout de nœud — priorité HAUTE.

---

## 5. D3/D4 — Composants

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/components` | Auth (lecture) | Liste des composants (filtres: name, category, tags, channel) |
| POST | `/api/components` | Contributor | Créer un composant (`AddFull`) |
| GET | `/api/components/{id}` | Auth | Détail d'un composant |
| PUT | `/api/components/{id}` | Contributor | Modifier (patch : name, description, tags, links, licenseID) |
| DELETE | `/api/components/{id}` | Contributor | Supprimer (local uniquement — Fabric immuable) |
| GET | `/api/components/{id}/interfaces` | Auth | Interfaces physiques d'un composant |

> **Non implémentés :** `GET /api/components/{id}/compatible` (UCREC02), `GET /api/components/{id}/versions` (UCREC03)

---

## 6. D5 — Connexions et Interfaces (Atelier)

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

## 7. D6 — Modules

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/modules` | Auth | Liste des modules |
| POST | `/api/modules` | Contributor | Créer un module (draft) |
| GET | `/api/modules/{id}` | Auth | Détail d'un module |
| PUT | `/api/modules/{id}` | Contributor | Modifier un module (patch) |
| DELETE | `/api/modules/{id}` | Contributor | Supprimer un module (draft uniquement) |
| GET | `/api/modules/{id}/workspace` | Auth | Instances dans l'atelier |
| POST | `/api/modules/{id}/workspace` | Contributor | Ajouter un composant à l'atelier |
| DELETE | `/api/modules/{id}/workspace/{instanceId}` | Contributor | Retirer un composant (+ cascade connexions) |
| PATCH | `/api/modules/{id}/workspace/{instanceId}/position` | Contributor | Mettre à jour la position |
| POST | `/api/modules/{id}/submit` | Contributor | Soumettre à la blockchain |
| GET | `/api/modules/{id}/interfaces` | Auth | Interfaces exposées du module |
| GET | `/api/modules/{id}/assemblies` | Auth | Connexions internes du module |

> **Non implémentés :** `GET /api/modules?uses_component={id}` (UCREC04), `GET /api/modules/{id}/bom` (UCREC05)

---

## 8. Licences

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/licenses` | Auth | Catalogue des licences |
| GET | `/api/licenses/{id}` | Auth | Détail d'une licence |

---

## 9. Miniatures

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| POST | `/api/components/{id}/thumbnail` | Contributor | Sauvegarder une miniature STL (data URL base64) |
| GET | `/api/components/{id}/thumbnail` | Auth | Récupérer la miniature |

---

## 10. Infrastructure

| Méthode | Route | Accès | Description |
|---------|-------|-------|-------------|
| GET | `/api/ping` | Public | Liveness (sans appel Fabric) |
| GET | `/api/status` | Public | Statut serveur |
| GET | `/api/health` | Public | Health check (load balancer) |
| GET | `/metrics` | Public | Métriques Prometheus |

---

## 11. Headers de sécurité (appliqués à toutes les réponses)

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
  img-src 'self' data: blob:;
  connect-src 'self';
  font-src 'self' data: https://fonts.gstatic.com;
  worker-src blob:;
  object-src 'none'
```

---

## 12. Routes à implémenter (non exposées)

| Route | Domaine | UC |
|-------|---------|-----|
| `POST /api/orders` | D7 | UCPI01 |
| `GET /api/orders/{id}` | D7 | UCPI01 |
| `POST /api/orders/{id}/deliver` | D7 | UCAUT01 |
| `GET/POST /api/assets/{id}/price` | D7 | UCPI04 |
| `POST /api/assets/{id}/transfer` | D7 | UCPI07 |
| `POST /api/assets/{id}/clone` | D7 | UCPI08 |
| `GET /api/components/{id}/compatible` | D8 | UCREC02 |
| `GET /api/components/{id}/versions` | D8 | UCREC03 |
| `GET /api/modules?uses_component={id}` | D8 | UCREC04 |
| `GET /api/modules/{id}/bom` | D8 | UCREC05 |
| `POST /api/admin/networks` | D2 | UCADM01 |
| `POST /api/admin/organizations` | D2 | UCADM02 |
| `POST /api/admin/peers` | D2 | UCADM03 |
| `POST /api/auth/request-role` | D1 | UCA08 |
