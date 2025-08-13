# **Detailed Functional & Technical Specifications (DFTS)**

## **Expense Report Management Application**

### **1\. Introduction and Objectives**

#### **1.1. Project Context**

This document aims to define the functional and technical specifications for the development of a web application for managing expense reports. The application must be simple, performant, secure, and designed to be scalable.

#### **1.2. Main Objectives**

* **Simplify Data Entry:** Allow users to quickly create expense reports.  
* **Centralize Receipts:** Link a photo of the receipt to each expense.  
* **Ensure Traceability:** Calculate and store amounts with and without tax, as well as VAT.  
* **Manage Permissions Precisely:** Secure access via a system of groups and granular permissions.  
* **Implement a Workflow:** Manage the lifecycle of an expense report (draft, submission, validation).  
* **Allow Interoperability:** Offer the ability to export data in multiple formats (PDF, CSV, JSON, YAML).

### **2\. Functional Specifications**

#### **2.1. User Journeys (User Stories)**

* **As a standard user,** I want to manage all my expense reports (create, modify, delete as long as they are in draft) and submit them for validation.  
* **As a validator,** I want to be able to view and validate (approve or reject) the expense reports submitted by users within my scope.  
* **As a Super Administrator,** I am the first user of the system. I want to be able to create other users, create groups, and assign specific permissions to those groups.  
* **As a user,** I want to be able to export my validated expense reports in PDF format.  
* **As an administrator,** I want to be able to export a global report of expenses in different formats for analysis or integration.  
* **As a mobile user,** I want to be able to install the application on my smartphone for quick access.

#### **2.2. Feature Description**

#### **2.2.1. Validation Workflow and Statuses**

The status field in the EXPENSE\_REPORTS table will follow this lifecycle:  
Draft \-\> Submitted \-\> Approved / Rejected

#### **2.2.2. Predefined Groups**

To facilitate implementation, the application will be initialized with the following groups:

* **Administrators:** Access to all administration features.  
* **Validators:** Can approve and reject expense reports.  
* **Users:** Can create and manage their own expense reports.

#### **2.2.3. Data Export**

* **PDF Export:** A user will be able to download an individual expense report in PDF format.  
* **Exports for Integration:** An administrator will be able to export a report of all expenses over a given period in the following formats:  
  * **CSV:** For processing in a spreadsheet.  
  * **JSON:** For integration with modern web applications.  
  * **YAML:** For configuration or integration with infrastructure tools.

### **3\. Technical Specifications**

#### **3.1. Architecture and Security**

* **Frontend (Client):** A single-page Progressive Web App (PWA) (SPA).  
* **Backend (Server):** A REST API developed in Go (Golang).  
* **Authentication:** The API will use JSON Web Tokens (JWT) for interactive sessions and static tokens for programmatic (API) access.  
* **Passwords:** User passwords will be hashed using the **bcrypt** algorithm.

#### **3.1.1. Rights and Permissions Management**

The rights system will be based on groups, not individual users.

* **Permissions:** Atomic and explicit permissions will be defined in the code (e.g., reports:approve, users:create, groups:manage).  
* **Groups:** A group is a container for permissions.  
* **Users:** A user is assigned one or more groups. Their rights are the union of all permissions from the groups they belong to.  
* **Verification Middleware:** Each sensitive API route will be protected by middleware that checks if the authenticated user (via their token) has the required permission to perform the action.

#### **3.1.2. Super Administrator Account**

A single Super Administrator account will be created on the application's first launch (or via a setup command). This account is the only one that can initially manage groups and assign permissions.

#### **3.1.3. Receipts Storage and datadir**

* **Data Directory (datadir):** A configurable main directory (e.g., **/var/data/expense-app**) will contain all persistent data.  
* **User Isolation:** Inside the datadir, receipts will be isolated in subdirectories per user for better organization and security.  
* **Structure:** **datadir/{user\_id}/receipts/**  
* **File Naming:** Each uploaded receipt will be renamed using the expense item's unique identifier (**expense\_item\_id**) followed by its original extension (.jpg, .png, etc.).  
* **Example:** **datadir/101/receipts/54321.jpg**  
* **Database Path:** The **receipt\_path** field in the **EXPENSE\_ITEMS** table will only store the filename (e.g., 54321.jpg). The application logic will be responsible for reconstructing the full path.

#### **3.2. Database Schema (SQLite)**

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
    USERS ||--|{ USER\_GROUPS : "belongs to"  
    GROUPS ||--|{ USER\_GROUPS : "contains"  
    GROUPS ||--|{ GROUP\_PERMISSIONS : "has"  
    PERMISSIONS ||--|{ GROUP\_PERMISSIONS : "is assigned to"  
    USERS ||--o{ EXPENSE\_REPORTS : "creates"  
    EXPENSE\_REPORTS ||--o{ EXPENSE\_ITEMS : "contains"

#### **3.3. REST API (Main Endpoints)**

| Category | HTTP Method | URL | Description | Required Permission |
| :---- | :---- | :---- | :---- | :---- |
| **Authentication** | POST | /api/auth/login | Log in and retrieve a JWT token. | (Public) |
| **Expenses** | POST | /api/reports/{report\_id}/items | Add an expense to an expense report. | reports:create |
|  | PUT | /api/items/{id} | Update expense data. | reports:update:own |
|  | POST | /api/items/{id}/receipt | Upload or replace an expense receipt. | reports:update:own |
|  | GET | /api/items/{id}/receipt | Retrieve the expense receipt file. | reports:read:own |
| **Administration** | GET | /api/admin/reports | Retrieve all submitted expense reports. | reports:read:all |
|  | POST | /api/admin/reports/{id}/approve | Approve an expense report. | reports:approve |
|  | POST | /api/admin/reports/{id}/reject | Reject an expense report. | reports:reject |
|  | GET | /api/admin/users | List all users. | users:read |
|  | POST | /api/admin/users | Create a user. | users:create |
|  | GET | /api/admin/groups | List all groups. | groups:read |
|  | POST | /api/admin/groups | Create a group. | groups:create |
|  | POST | /api/admin/groups/{id}/permissions | Assign a permission to a group. | permissions:assign |
|  | POST | /api/admin/users/{id}/token | Generate a static API token for a user. | tokens:create |
| **Exports** | GET | /api/admin/export/csv | Export all expenses as CSV. | reports:export:all |
|  | GET | /api/admin/export/json | Export all expenses as JSON. | reports:export:all |
|  | GET | /api/admin/export/yaml | Export all expenses as YAML. | reports:export:all |

#### **3.4. Details of Technologies and Tools**

| Domain | Component | Suggested Tool / Library | Role and Justification |
| :---- | :---- | :---- | :---- |
| **Backend** | Web Framework | Gin ([github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)) | Lightweight, fast, and has a mature middleware ecosystem. |
|  | DB Access | database/sql \+ [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | Use of the standard Go package and a robust CGO driver. |
|  | JWT Auth | [github.com/golang-jwt/jwt](https://github.com/golang-jwt/jwt) | Reference implementation for creating and validating JWTs. |
|  | Password Hashing | golang.org/x/crypto/bcrypt | Official and industry-standard package for secure hashing. |
|  | YAML Export | gopkg.in/yaml.v3 | Reference library for YAML manipulation in Go. |
|  | PDF Export | [github.com/jung-kurt/gofpdf](https://github.com/jung-kurt/gofpdf) | Native Go library for generating PDF documents. |
| **Frontend** | JS Framework | Vanilla JavaScript (ES6+) | Build the PWA without the complexity of a large framework. |
|  | PWA & Offline | Workbox | Google library simplifying Service Worker management. |
|  | CSS Styling | Tailwind CSS | "Utility-first" framework for building responsive designs. |
|  | Layout | CSS Flexbox & Grid | Native technologies for creating complex and fluid layouts. |
| **Tests** | Backend Tests | testing \+ [github.com/stretchr/testify](https://github.com/stretchr/testify) | Native package supplemented by testify for more readable assertions. |
|  | End-to-End Tests | Playwright or Cypress | Powerful tools for automating browser interactions. |
| **Tooling** | Containerization | Docker & Docker Compose | Create reproducible development and production environments. |
|  | API Documentation | Swaggo ([github.com/swaggo/swag](https://github.com/swaggo/swag)) | Generates OpenAPI (Swagger) documentation from comments. |
|  | Continuous Integration | GitHub Actions | For automating builds, tests, and potentially deployment. |

### **4\. Project Deliverables**

In addition to the application's source code, the following deliverables must be produced to ensure the quality, maintainability, and adoption of the project.  
| Deliverable | Description | Detailed Content |  
| :--- | :--- | :--- |  
| Test Plan | Guarantee the quality and non-regression of the application via automated tests. | \- Unit Tests (business logic, calculations)\<br\>- Integration Tests (API endpoints, database)\<br\>- End-to-End Tests (complete user scenarios) |  
| Technical Documentation | Allow developers to easily maintain and evolve the project. | \- API Documentation (generated via OpenAPI/Swagger)\<br\>- Installation Guide (README.md with Docker instructions)\<br\>- Architecture Documentation (this DFTS document) |  
| User Manual | Provide clear and accessible support for the application's end-users. | \- Standard User Guide (creation, management, and submission of expense reports)\<br\>- Administrator Guide (management of users, groups, rights, and validation) |