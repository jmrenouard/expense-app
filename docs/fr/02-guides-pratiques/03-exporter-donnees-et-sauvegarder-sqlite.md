# Exporter les données et sauvegarder SQLite

Consultez [Stockage, proxy et sécurité](../04-explications/02-stockage-proxy-et-securite.md) avant de démarrer cette procédure.

## Prérequis

- Authentifiez un compte `Administrateurs` doté de `reports:export:all` puis exportez `$ADMIN_TOKEN`.
- Vérifiez l'accès en écriture vers un répertoire de sauvegarde sur l'hôte.

## Consignes de sécurité

**ATTENTION :** Arrêtez les écritures ou utilisez `sqlite3 .backup` avant de copier `expense.db`. Une copie brute pendant une écriture active corrompt la base SQLite.

## Étapes

1. Exportez toutes les notes de frais aux formats CSV, JSON et YAML :

   ```bash
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/csv -o expenses.csv
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/json -o expenses.json
   curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/admin/export/yaml -o expenses.yaml
   ```

2. Créez un instantané SQLite en ligne avec `sqlite3` :

   ```bash
   sqlite3 ./data/expense.db ".backup './data/expense-backup.db'"
   ```

3. Archivez la base sauvegardée avec le répertoire `./data/*/receipts` :

   ```bash
   tar -czf expense-backup.tar.gz ./data/expense-backup.db ./data/*/receipts
   ```

## Vérification

1. Vérifiez que `expenses.csv`, `expenses.json` et `expenses.yaml` contiennent des enregistrements non vides.
2. Exécutez une vérification d'intégrité sur la copie SQLite :

   ```bash
   sqlite3 ./data/expense-backup.db "PRAGMA integrity_check;"
   ```

3. Vérifiez que `PRAGMA integrity_check;` renvoie `ok`.

## Retour arrière

1. Si `expense.db` devient corrompue, arrêtez le conteneur avec `make stop`.
2. Remplacez `./data/expense.db` par `./data/expense-backup.db` puis démarrez le conteneur.
