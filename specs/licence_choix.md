# Stratégie de licence open-source du projet

## 1. Contexte

### 1.1 Objectif

Ce document définit la stratégie de licence open-source du projet en vue de sa publication sur GitHub.

Le choix de licence doit répondre à la question suivante :

> Comment protéger le cœur du projet tout en permettant un écosystème commercial autour ?

### 1.2 Exigences

Le projet doit satisfaire les exigences suivantes :

- rendre le projet librement consultable, utilisable, modifiable et redistribuable ;
- permettre à la communauté de contribuer aux évolutions ;
- empêcher la création de versions fermées concurrentes du cœur ;
- imposer que les améliorations du cœur technique restent publiques ;
- autoriser la création de logiciels privés ou commerciaux utilisant l'outil via son API ou son CLI ;
- autoriser les services commerciaux autour de l'outil : hébergement, intégration, support, formation, connecteurs, applications métier ;
- conserver une conformité open-source reconnue.

## 2. Analyse des licences

### 2.1 Open-source et activité commerciale

Une restriction non-commerciale n'est pas compatible avec la définition open-source reconnue par l'Open Source Initiative.

Une licence open-source doit permettre l'usage dans tous les domaines, y compris commercial. Une clause interdisant l'usage commercial transforme le projet en logiciel *source-available* ou *visible source*, avec plusieurs conséquences négatives : rejet par certaines entreprises et communautés, frein aux contributions, ambiguïté juridique sur la monétisation.

L'objectif n'est pas d'interdire l'activité commerciale, mais d'empêcher la privatisation du cœur. C'est le rôle des licences copyleft.

### 2.2 Licences permissives vs copyleft

Les licences permissives (MIT, BSD-2, BSD-3, Apache-2.0, 0BSD, Unlicense) accordent beaucoup de libertés avec peu d'obligations. Elles autorisent notamment la modification, la redistribution, l'intégration dans du logiciel propriétaire et la création de forks fermés.

Leur avantage est l'adoption maximale. Leur inconvénient est qu'elles n'imposent pas que les améliorations reviennent au projet : une entreprise peut prendre le code, le modifier, créer une version propriétaire et ne pas publier ses modifications.

Ce modèle n'est pas adapté pour protéger le cœur du projet.

Les licences copyleft (GPL-3.0, AGPL-3.0, LGPL-3.0, MPL-2.0, EUPL-1.2) imposent que les versions dérivées concernées restent sous une licence compatible ou identique.

Elles protègent le logiciel libre tout en autorisant l'usage commercial. Elles sont adaptées lorsque l'objectif est d'empêcher la privatisation des modifications du cœur.

### 2.3 Comparatif des licences

| Licence | Type | Pertinence pour le projet |
| :--- | :---: | ---: |
| Aucune licence | Aucun droit explicite | Non adaptée. Un dépôt public sans licence n'est pas une stratégie open-source. |
| MIT | Permissive | Non adaptée pour protéger le cœur : autorise les forks propriétaires. |
| BSD-2 / BSD-3 | Permissive | Non adaptée : même limitation que MIT. |
| Apache-2.0 | Permissive avec clause brevet | Adaptée pour un SDK ou des exemples, pas pour le cœur. |
| MPL-2.0 | Copyleft faible au niveau fichier | Protection limitée fichier par fichier ; moins robuste pour un cœur applicatif complet. |
| LGPL-3.0 | Copyleft faible orienté bibliothèque | Pertinente pour une bibliothèque technique, pas pour le cœur principal. |
| GPL-3.0 | Copyleft fort | Bonne protection contre les forks distribués ; insuffisante si l'outil est exploité comme service réseau. |
| AGPL-3.0 | Copyleft fort avec clause réseau | **Licence la plus adaptée.** Couvre les forks distribués et les forks exploités en SaaS/API. |
| EUPL-1.2 | Copyleft européen | Pertinente dans un contexte institutionnel européen ; moins standard qu'AGPL pour un outil serveur/API. |
| Unlicense / CC0 / 0BSD | Ultra-permissive | Non adaptée : aucune protection contre la privatisation. |
| Creative Commons | Licence de contenu | Adaptée à la documentation, pas au code. |
| Licence non-commerciale | Source-available | Non recommandée : bloque l'écosystème commercial souhaité et sort du périmètre open-source. |
| SSPL | Copyleft serveur, non OSI | Non recommandée : non reconnue comme open-source ; obligations trop larges. |

### 2.4 Limite de la GPL-3.0 (faille SaaS)

La GPL-3.0 protège contre les forks propriétaires distribués. Si une entreprise modifie le logiciel et le vend sous forme installable, elle doit fournir le code source correspondant.

En revanche, si une entreprise modifie le logiciel, l'installe sur ses propres serveurs et vend un accès via API ou SaaS sans jamais distribuer le logiciel, la GPL-3.0 peut ne pas l'obliger à publier ses modifications. C'est la faille dite « SaaS » ou « ASP ».

L'AGPL-3.0 comble cette faille par une clause réseau.

### 2.5 AGPL-3.0 : couverture réseau

L'AGPL-3.0 ajoute à la GPL-3.0 une clause spécifique : si des utilisateurs interagissent avec une version modifiée du programme via un réseau, ils doivent pouvoir accéder au code source correspondant.

Ce mécanisme couvre le scénario suivant : une entreprise prend l'outil, le modifie, l'exploite en SaaS ou via API, et vend l'accès au service — elle doit proposer le code source modifié aux utilisateurs du service.

L'AGPL n'interdit pas le commerce. Elle interdit la privatisation des modifications du cœur.

## 3. Analyse approfondie de l'AGPL-3.0

### 3.1 Ce que l'AGPL autorise

| Cas | Autorisé ? | Commentaire |
| :--- | :---: | ---: |
| Usage de l'outil tel quel | Oui | Conforme à la licence. |
| Hébergement commercial de l'outil | Oui | L'usage commercial reste possible. |
| Vente de support ou d'intégration | Oui | Services autour autorisés. |
| Logiciel privé appelant l'API publique | Oui, si séparation claire | Le logiciel tiers reste distinct du cœur AGPL. |
| Connecteur externe séparé | Oui, selon le niveau de couplage | Faible risque si le connecteur communique via API publique. |
| Modification du cœur + redistribution | Oui, mais source obligatoire | Effet copyleft classique. |
| Modification du cœur + exploitation en SaaS | Oui, mais source obligatoire pour les utilisateurs | Effet spécifique de la clause réseau AGPL. |

### 3.2 Frontière API et notion d'œuvre dérivée

Dans les cas courants, un logiciel tiers qui communique avec l'outil uniquement via une API publique, un protocole réseau ou des appels HTTP reste un programme séparé. Il peut conserver sa propre licence, y compris propriétaire.

Cette frontière ne constitue cependant pas une garantie juridique absolue. La qualification d'œuvre dérivée dépend du degré de couplage technique et fonctionnel.

| Type d'intégration | Risque de contrainte AGPL |
| :--- | :---: | ---: |
| Appel HTTP/REST/GraphQL documenté | Faible |
| Communication par protocole public stable | Faible |
| Client externe indépendant | Faible |
| Connecteur externe séparé | Faible à modéré |
| Plugin chargé dans le cœur applicatif | Élevé |
| Module compilé avec le cœur | Élevé |
| Extension manipulant les structures internes | Élevé |
| Fork modifiant directement le cœur | Très élevé |

Formulation recommandée pour le README :

```
Les logiciels tiers qui communiquent avec l'outil uniquement via son API publique documentée
peuvent utiliser leur propre licence, y compris propriétaire.

Cette règle suppose une séparation technique claire entre le logiciel tiers et le cœur AGPL.
Les plugins, modules internes, extensions fortement couplées ou modifications directes du cœur
peuvent être soumis aux obligations de l'AGPL.
```

### 3.3 AGPL-3.0-only ou AGPL-3.0-or-later

| Variante | Sens |
| :--- | :---: | ---: |
| AGPL-3.0-only | Le projet est sous AGPL version 3 uniquement. |
| AGPL-3.0-or-later | Le projet est sous AGPL v3 ou toute version ultérieure publiée par la FSF. |

La recommandation est `AGPL-3.0-or-later` : meilleure souplesse à long terme, adaptation plus facile si le cadre juridique évolue, possibilité de bénéficier d'améliorations futures de la licence.



## 4. Architecture de licence du projet

### 4.1 Répartition par composant

| Partie du projet | Licence recommandée | Raison |
| :--- | :---: | ---: |  |
| Cœur principal de l'outil | AGPL-3.0-or-later | Protège contre la privatisation des modifications, y compris en SaaS/API. |
| API publique documentée | Documentation claire | Définit la frontière entre cœur AGPL et logiciels tiers. |
| SDK client officiel | Apache-2.0 ou MIT | Facilite l'intégration dans des logiciels privés. |
| Exemples d'appel API | MIT ou Apache-2.0 | Réduit la friction pour les utilisateurs professionnels. |
| Connecteurs externes simples | MIT, Apache-2.0 ou licence au choix | Pertinent s'ils restent séparés du cœur. |
| Plugins internes fortement couplés | AGPL-3.0 par défaut | Évite de contourner la licence par modules propriétaires. |
| Documentation | CC-BY 4.0 ou équivalent | Adapté aux textes, schémas et guides. |
| Nom, logo, identité visuelle | Protection séparée | La licence du code ne protège pas l'identité du projet. |

### 4.2 Compatibilité avec les dépendances Apache-2.0

Un projet AGPL-3.0 peut intégrer des dépendances Apache-2.0. La compatibilité est directionnelle :

```
Apache-2.0 → AGPL-3.0 : généralement compatible.
AGPL-3.0 → Apache-2.0 : non, on ne peut pas rendre permissif du code AGPL dérivé.
```

Les obligations des dépendances Apache-2.0 doivent être respectées : conservation des notices de copyright, du texte de licence et des fichiers NOTICE, documentation des dépendances, vérification des licences transitives.

Dans le cas d'un projet Go, ce point est important car les dépendances peuvent être compilées dans le binaire final.

**Application directe :** Hyperledger Fabric est sous Apache-2.0. Son utilisation comme dépendance ou infrastructure est compatible avec un projet global AGPL-3.0. Le code propre du cœur reste soumis aux obligations AGPL.

### 4.3 Option : dual licensing

Le dual licensing permet de proposer simultanément :

- une version communautaire sous AGPL-3.0-or-later ;
- une licence commerciale payante pour les entreprises souhaitant s'affranchir des obligations AGPL.

| Public | Licence |
| :--- | :---: | ---: |
| Communauté, associations, chercheurs, utilisateurs libres | AGPL-3.0-or-later |
| Entreprises acceptant les obligations AGPL | AGPL-3.0-or-later |
| Entreprises voulant une intégration propriétaire forte | Licence commerciale payante |
| SDK client ou exemples API | MIT ou Apache-2.0 |
| Documentation | CC-BY 4.0 ou équivalent |

Pour pouvoir proposer une licence commerciale, il faut contrôler les droits sur l'ensemble du code. Cela implique que les contributeurs externes signent un CLA, ou que les contributions externes soient strictement encadrées.

Le dual licensing n'est pas obligatoire au démarrage, mais doit être anticipé tôt si le projet a une ambition commerciale : le mettre en place après l'arrivée de contributions externes est significativement plus complexe.

### 4.4 Protection du nom, du logo et de la marque

La licence open-source protège le code, pas l'identité du projet. Le nom, le logo, l'identité visuelle et la notion de version officielle nécessitent une politique séparée.

Exemple de clause :

```
Le code source est distribué sous licence AGPL-3.0-or-later.
Le nom du projet, le logo et les éléments d'identité visuelle ne peuvent pas être utilisés
pour promouvoir un fork, une version modifiée ou un service commercial sans autorisation explicite.
```

## 5. Mise en œuvre

### 5.1 Structure de dépôt recommandée

```
.
├── LICENSE           → Texte complet de l'AGPL-3.0-or-later
├── README.md         → Résumé de la stratégie de licence
├── NOTICE            → Notices de copyright, dépendances, mentions
├── TRADEMARKS.md     → Règles d'usage du nom, logo et marque
├── CONTRIBUTING.md   → Conditions de contribution
├── CLA.md            → Accord contributeur (si dual licensing envisagé)
├── docs/LICENSE.md   → Licence de la documentation (CC-BY 4.0)
├── sdk/LICENSE       → Licence MIT ou Apache-2.0 pour le SDK
└── examples/LICENSE  → Licence MIT ou Apache-2.0 pour les exemples
```

### 5.2 Formulation pour le README

```markdown
## Licence

Le cœur de ce projet est distribué sous licence GNU Affero General Public License v3.0 or later
(`AGPL-3.0-or-later`).

Le logiciel peut être utilisé, étudié, modifié et redistribué librement. Toute version modifiée
du cœur, lorsqu'elle est distribuée ou mise à disposition via un service réseau, doit rester
compatible avec les obligations de l'AGPL et fournir le code source correspondant aux utilisateurs.

Les logiciels tiers qui communiquent avec l'outil uniquement via son API publique documentée
peuvent utiliser leur propre licence, y compris propriétaire, à condition de rester techniquement
séparés du cœur AGPL. Les plugins, modules internes, extensions fortement couplées ou modifications
directes du cœur peuvent être soumis aux obligations de l'AGPL.

Le nom du projet, le logo et les éléments d'identité visuelle ne sont pas couverts par la licence
du code et font l'objet de règles d'usage séparées.
```

## 6. Synthèse et conclusion

### 6.1 Synthèse décisionnelle

| Objectif | Licence la plus adaptée |
| :--- | :---: | ---: |
| Maximiser l'adoption sans contrainte sur les modifications | MIT ou Apache-2.0 |
| Protéger seulement les fichiers modifiés | MPL-2.0 |
| Protéger une bibliothèque utilisée par des logiciels propriétaires | LGPL-3.0 |
| Empêcher les forks propriétaires distribués | GPL-3.0 |
| Empêcher les forks propriétaires distribués et les forks SaaS/API | AGPL-3.0 |
| Bloquer fortement les fournisseurs cloud (hors périmètre OSI) | SSPL — non recommandée |
| Protéger documentation et schémas | Creative Commons |
| Protéger le nom et le logo | Marque ou politique d'usage séparée |

### 6.2 Conclusion

Le besoin est de protéger le cœur du logiciel contre la privatisation, tout en autorisant un écosystème commercial autour.

Les licences permissives (MIT, BSD, Apache-2.0) ne répondent pas à cet objectif. La MPL-2.0 offre une protection insuffisante au niveau applicatif. La LGPL-3.0 est adaptée aux bibliothèques, pas au cœur d'un serveur. La GPL-3.0 ne couvre pas les exploitations en mode service réseau. La SSPL va au-delà du périmètre open-source reconnu.

**L'AGPL-3.0-or-later est la licence la plus adaptée pour le cœur du projet.**

Architecture recommandée :

| Partie | Licence |
| :--- | :---: | ---: |
| Cœur du projet | AGPL-3.0-or-later |
| SDK client | MIT ou Apache-2.0 |
| Exemples API | MIT ou Apache-2.0 |
| Documentation | CC-BY 4.0 |
| Nom et logo | Protection séparée |
| Option future | Dual licensing possible |

### 6.3 Sources

- [GitHub Docs — Licensing a repository](https://docs.github.com/articles/licensing-a-repository)
- [Open Source Initiative — Open Source Definition](https://opensource.org/osd)
- [GNU — GNU Affero General Public License v3.0](https://www.gnu.org/licenses/agpl-3.0.html)
- [GNU — GPL FAQ](https://www.gnu.org/licenses/gpl-faq.html)
- [Choose a License](https://choosealicense.com/licenses/)
- [Apache Software Foundation — Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
- [Apache Software Foundation — GPL compatibility](https://www.apache.org/licenses/GPL-compatibility.html)
- [Open Source Initiative — Licenses](https://opensource.org/licenses)
- [Nextcloud — Licensing information](https://nextcloud.com/licensing/)
