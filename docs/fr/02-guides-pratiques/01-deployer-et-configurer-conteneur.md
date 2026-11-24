# Déployer et configurer le conteneur

Consultez [Variables, schéma et Makefile](../03-reference/02-variables-schema-et-makefile.md) avant de démarrer cette procédure.

## Prérequis

- Installez Docker et GNU Make sur l'hôte.
- Préparez un répertoire accessible en écriture pour SQLite.

## Consignes de sécurité

**AVERTISSEMENT :** Définissez des valeurs propres pour `JWT_SECRET` et `ADMIN_PASSWORD` avant le démarrage. Les identifiants par défaut exposent l'API d'administration aux accès non autorisés.

## Étapes

1. Construisez l'image Docker depuis la racine du dépôt :

   ```bash
   make build
   ```

2. Créez le répertoire de données local sur l'hôte :

   ```bash
   mkdir -p ./data
   ```

3. Lancez le conteneur avec vos paramètres `JWT_SECRET`, `ADMIN_EMAIL` et `ADMIN_PASSWORD` :

   ```bash
   docker run -d \
     -p 8080:8080 \
     -e PORT=8081 \
     -e DATADIR=/data \
     -e JWT_SECRET="replace-with-random-secret" \
     -e ADMIN_EMAIL="admin@corp.local" \
     -e ADMIN_PASSWORD="replace-with-strong-password" \
     -v "$(pwd)/data:/data" \
     --name expenseapp \
     expenseapp
   ```

4. Attendez que le serveur Go et le proxy Nginx écoutent sur le port `8080`.

## Vérification

1. Examinez les journaux du conteneur pour vérifier le message d'écoute :

   ```bash
   docker logs expenseapp
   ```

2. Envoyez une requête de connexion sur `/api/auth/login` :

   ```bash
   curl -s -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@corp.local","password":"replace-with-strong-password"}'
   ```

3. Vérifiez que la réponse JSON contient un champ `token` non vide.

## Retour arrière

1. Si les vérifications échouent, arrêtez et supprimez le conteneur :

   ```bash
   make stop
   ```

2. Restaurez la sauvegarde précédente `./data/expense.db` avant de démarrer un conteneur corrigé.
