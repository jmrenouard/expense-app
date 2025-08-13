# **Spécifications Fonctionnelles & Techniques Détaillées (SFTD)**

## **Application de Gestion de Notes de Frais**

### **1\. Introduction et Objectifs**

#### **1.1. Contexte du projet**

Ce document a pour but de définir les spécifications fonctionnelles et techniques pour le développement d'une application web de gestion de notes de frais. L'application doit être simple, performante, sécurisée, et conçue pour être évolutive.

#### **1.2. Objectifs principaux**

* **Simplifier la saisie :** Permettre aux utilisateurs de créer rapidement des notes de frais.  
* **Centraliser les justificatifs :** Associer une photo du justificatif à chaque dépense.  
* **Assurer la traçabilité :** Calculer et stocker les montants TTC, HT et la TVA.  
* **Gérer les permissions finement :** Sécuriser les accès via un système de groupes et de permissions granulaires.  
* **Mettre en place un workflow :** Gérer le cycle de vie d'une note de frais (brouillon, soumission, validation).  
* **Permettre l'interopérabilité :** Offrir la possibilité d'exporter les données dans de multiples formats (PDF, CSV, JSON, YAML).

### **2\. Spécifications Fonctionnelles**

#### **2.1. Parcours Utilisateur (User Stories)**

* En tant qu'**utilisateur standard**, je veux gérer l'ensemble de mes notes de frais (créer, modifier, supprimer tant qu'elles sont en brouillon) et les soumettre pour validation.  
* En tant que **validateur**, je veux pouvoir consulter et valider (approuver ou rejeter) les notes de frais soumises par les utilisateurs de mon périmètre.  
* En tant que **Super Administrateur**, je suis le premier utilisateur du système. Je veux pouvoir créer d'autres utilisateurs, créer des groupes, et assigner des permissions précises à ces groupes.  
* En tant qu'**utilisateur**, je veux pouvoir exporter mes notes de frais validées au format PDF.  
* En tant qu'**administrateur**, je veux pouvoir exporter un rapport global des dépenses dans différents formats pour analyse ou intégration.  
* En tant qu'**utilisateur mobile**, je veux pouvoir installer l'application sur mon smartphone pour un accès rapide.

#### **2.2. Description des fonctionnalités**

##### **2.2.1. Workflow de Validation et Statuts**

Le champ status de la table EXPENSE\_REPORTS suivra le cycle de vie suivant :

* **Brouillon (draft)** \-\> **Soumise (submitted)** \-\> **Approuvée (approved)** / **Rejetée (rejected)**

##### **2.2.2. Groupes Prédéfinis**

Pour faciliter la mise en place, l'application sera initialisée avec les groupes suivants :

* **Administrateurs :** Accès à toutes les fonctionnalités d'administration.  
* **Validateurs :** Peuvent approuver et rejeter les notes de frais.  
* **Utilisateurs :** Peuvent créer et gérer leurs propres notes de frais.

##### **2.2.3. Export de Données**

* **Export PDF :** Un utilisateur pourra télécharger une note de frais individuelle au format PDF.  
* **Exports pour l'intégration :** Un administrateur pourra exporter un rapport de toutes les dépenses sur une période donnée dans les formats suivants :  
  * **CSV :** Pour un traitement dans un tableur.  
  * **JSON :** Pour l'intégration avec des applications web modernes.  
  * **YAML :** Pour la configuration ou l'intégration avec des outils d'infrastructure.

### **3\. Spécifications Techniques**

#### **3.1. Architecture et Sécurité**

* **Frontend (Client) :** Une Progressive Web App (PWA) monopage (SPA).  
* **Backend (Serveur) :** Une API REST développée en **Go (Golang)**.  
* **Authentification :** L'API utilisera des **JSON Web Tokens (JWT)** pour les sessions interactives et des **tokens statiques** pour les accès programmatiques (API).  
* **Mots de passe :** Les mots de passe des utilisateurs seront hachés en utilisant l'algorithme **bcrypt**.

##### **3.1.1. Gestion des Droits et Permissions**

Le système de droits sera basé sur les groupes et non sur les utilisateurs individuels.

* **Permissions :** Des permissions atomiques et explicites seront définies dans le code (ex: reports:approve, users:create, groups:manage).  
* **Groupes :** Un groupe est un conteneur de permissions.  
* **Utilisateurs :** Un utilisateur se voit assigner un ou plusieurs groupes. Ses droits sont l'union de toutes les permissions des groupes auxquels il appartient.  
* **Middleware de Vérification :** Chaque route sensible de l'API sera protégée par un middleware qui vérifiera si l'utilisateur authentifié (via son token) possède la permission requise pour effectuer l'action.

##### **3.1.2. Compte Super Administrateur**

Un compte unique **Super Administrateur** sera créé au premier lancement de l'application (ou via une commande de setup). Ce compte est le seul à pouvoir initialement gérer les groupes et assigner des permissions.

##### **3.1.3. Stockage des Justificatifs et datadir**

* **Répertoire de données (datadir) :** Un répertoire principal configurable (ex: /var/data/expense-app) contiendra toutes les données persistantes.  
* **Isolation par utilisateur :** À l'intérieur du datadir, les justificatifs seront isolés dans des sous-répertoires par utilisateur pour une meilleure organisation et sécurité.  
  * Structure : datadir/{user\_id}/receipts/  
* **Nommage des fichiers :** Chaque justificatif téléversé sera renommé en utilisant l'identifiant unique de la dépense (expense\_item\_id) suivi de son extension d'origine (.jpg, .png, etc.).  
  * Exemple : datadir/101/receipts/54321.jpg  
* **Chemin en base de données :** Le champ receipt\_path de la table EXPENSE\_ITEMS stockera uniquement le nom du fichier (ex: 54321.jpg). La logique applicative se chargera de reconstruire le chemin complet.

#### **3.2. Schéma de la Base de Données (SQLite)**

erDiagram  
    USERS {  
        int id PK  
        string email  
        string password\_hash  
        datetime created\_at  
    }  
    GROUPS {  
        int id PK  
        string name Unique  
    }  
    PERMISSIONS {  
        int id PK  
        string action Unique "ex: reports:approve"  
    }  
    USER\_GROUPS {  
        int user\_id PK, FK  
        int group\_id PK, FK  
    }  
    GROUP\_PERMISSIONS {  
        int group\_id PK, FK  
        int permission\_id PK, FK  
    }  
    EXPENSE\_REPORTS {  
        int id PK  
        int user\_id FK  
        string title  
        string status  
        datetime created\_at  
    }  
    EXPENSE\_ITEMS {  
        int id PK  
        int report\_id FK  
        string description  
        date expense\_date  
        float amount\_ht  
        float amount\_ttc  
        float vat\_rate  
        string receipt\_path  
        datetime created\_at  
    }  
    USERS ||--|{ USER\_GROUPS : "appartient à"  
    GROUPS ||--|{ USER\_GROUPS : "contient"  
    GROUPS ||--|{ GROUP\_PERMISSIONS : "possède"  
    PERMISSIONS ||--|{ GROUP\_PERMISSIONS : "est assignée à"  
    USERS ||--o{ EXPENSE\_REPORTS : "crée"  
    EXPENSE\_REPORTS ||--o{ EXPENSE\_ITEMS : "contient"

#### **3.3. API REST (Endpoints principaux)**

| Catégorie | Méthode HTTP | URL | Description | Permission Requise |
| :---- | :---- | :---- | :---- | :---- |
| **Authentification** | POST | /api/auth/login | Connexion et récupération d'un token JWT. | (Publique) |
| **Dépenses** | POST | /api/reports/{report\_id}/items | Ajoute une dépense à une note de frais. | reports:create |
|  | PUT | /api/items/{id} | Met à jour les données d'une dépense. | reports:update:own |
|  | POST | /api/items/{id}/receipt | **Téléverse ou remplace la pièce jointe** d'une dépense. | reports:update:own |
|  | GET | /api/items/{id}/receipt | **Récupère le fichier de la pièce jointe** d'une dépense. | reports:read:own |
| **Administration** | GET | /api/admin/reports | Récupère toutes les notes de frais soumises. | reports:read:all |
|  | POST | /api/admin/reports/{id}/approve | Approuve une note de frais. | reports:approve |
|  | POST | /api/admin/reports/{id}/reject | Rejette une note de frais. | reports:reject |
|  | GET | /api/admin/users | Liste tous les utilisateurs. | users:read |
|  | POST | /api/admin/users | Crée un utilisateur. | users:create |
|  | GET | /api/admin/groups | Liste tous les groupes. | groups:read |
|  | POST | /api/admin/groups | Crée un groupe. | groups:create |
|  | POST | /api/admin/groups/{id}/permissions | Assigne une permission à un groupe. | permissions:assign |
|  | POST | /api/admin/users/{id}/token | Génère un token d'API statique pour un utilisateur. | tokens:create |
| **Exports** | GET | /api/admin/export/csv | Exporte toutes les dépenses en CSV. | reports:export:all |
|  | GET | /api/admin/export/json | Exporte toutes les dépenses en JSON. | reports:export:all |
|  | GET | /api/admin/export/yaml | Exporte toutes les dépenses en YAML. | reports:export:all |

#### **3.4. Détail des Technologies et Outils**

| Domaine | Composant | Outil / Bibliothèque Suggérée | Rôle et Justification |
| :---- | :---- | :---- | :---- |
| **Backend** | Framework Web | **Gin** (github.com/gin-gonic/gin) | Léger, rapide, et dispose d'un écosystème de middlewares mature. |
|  | Accès BDD | **database/sql** \+ **github.com/mattn/go-sqlite3** | Utilisation du package standard Go et d'un driver CGO robuste. |
|  | Auth JWT | **github.com/golang-jwt/jwt** | Implémentation de référence pour la création et la validation des JWT. |
|  | Hachage Mots de Passe | **golang.org/x/crypto/bcrypt** | Package officiel et standard de l'industrie pour le hachage sécurisé. |
|  | Export YAML | **gopkg.in/yaml.v3** | Bibliothèque de référence pour la manipulation de YAML en Go. |
|  | Export PDF | **github.com/jung-kurt/gofpdf** | Bibliothèque native Go pour générer des documents PDF. |
| **Frontend** | Framework JS | **Vanilla JavaScript (ES6+)** | Construire la PWA sans la complexité d'un grand framework. |
|  | PWA & Offline | **Workbox** | Bibliothèque Google simplifiant la gestion des Service Workers. |
|  | Styling CSS | **Tailwind CSS** | Framework "utility-first" pour construire des designs responsives. |
|  | Mise en page | **CSS Flexbox & Grid** | Technologies natives pour créer des mises en page complexes et fluides. |
| **Tests** | Tests Backend | **testing** \+ **github.com/stretchr/testify** | Package natif complété par testify pour des assertions plus lisibles. |
|  | Tests End-to-End | **Playwright** ou **Cypress** | Outils puissants pour automatiser les interactions dans un navigateur. |
| **Outillage** | Conteneurisation | **Docker & Docker Compose** | Créer des environnements de développement et production reproductibles. |
|  | Documentation API | **Swaggo** (github.com/swaggo/swag) | Génère une documentation OpenAPI (Swagger) depuis les commentaires. |
|  | Intégration Continue | **GitHub Actions** | Pour automatiser les builds, les tests et potentiellement le déploiement. |

### **4\. Livrables du Projet**

En plus du code source de l'application, les livrables suivants devront être produits pour garantir la qualité, la maintenabilité et l'adoption du projet.

| Livrable | Description | Contenu Détaillé |
| :---- | :---- | :---- |
| **Programme de Test** | Garantir la qualité et la non-régression de l'application via des tests automatisés. | \- **Tests Unitaires** (logique métier, calculs)\<br\>- **Tests d'Intégration** (endpoints API, base de données)\<br\>- **Tests End-to-End** (scénarios utilisateurs complets) |
| **Documentation Technique** | Permettre aux développeurs de maintenir et faire évoluer le projet facilement. | \- **Documentation de l'API** (générée via OpenAPI/Swagger)\<br\>- **Guide d'Installation** (README.md avec instructions Docker)\<br\>- **Documentation d'Architecture** (ce document SFTD) |
| **Manuel Utilisateur** | Fournir un support clair et accessible pour les utilisateurs finaux de l'application. | \- **Guide pour l'Utilisateur Standard** (création, gestion et soumission des notes de frais)\<br\>- **Guide pour l'Administrateur** (gestion des utilisateurs, groupes, droits et validation) |
