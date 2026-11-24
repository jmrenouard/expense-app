# Gérer utilisateurs, groupes et validations

Consultez [API REST et permissions](../03-reference/01-api-rest-et-permissions.md) avant de démarrer cette procédure.

## Prérequis

- Vérifiez qu'`expenseapp` répond sur `http://localhost:8080`.
- Authentifiez un compte du groupe `Administrateurs` puis exportez `$ADMIN_TOKEN`.

## Étapes

1. Listez les groupes initiaux pour lire leurs identifiants (`Administrateurs`, `Validateurs`, `Utilisateurs`) :

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/groups
   ```

2. Créez un compte utilisateur rattaché au groupe `3` (`Utilisateurs`) :

   ```bash
   curl -s -X POST http://localhost:8080/api/admin/users \
     -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"email":"employee@corp.local","password":"change-me-now","groups":[3]}'
   ```

3. Listez les notes soumises en attente de validation :

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/reports
   ```

4. Approuvez ou rejetez une note soumise avec son identifiant :

   ```bash
   curl -s -X POST -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/reports/1/approve
   ```

**NOTE :** Seules les notes au statut `submitted` passent au statut `approved` ou `rejected`.

## Vérification

1. Exécutez `GET /api/admin/users` puis vérifiez la présence de la nouvelle adresse email.
2. Exécutez `GET /api/admin/reports` puis vérifiez que la note `1` affiche `"status":"approved"`.

## Retour arrière

1. Si la validation vise la mauvaise note, examinez immédiatement `GET /api/admin/reports`.
2. Créez une note corrigée ou restaurez la sauvegarde SQLite si la conformité exige un retour d'état.
