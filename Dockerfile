# Étape 1: Construire l'application Go
FROM golang:1.20-alpine3.17 AS builder

# Installer git et les outils de construction C
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copier tous les fichiers sources
COPY . .

# Télécharger les dépendances et les vendorer
RUN go mod tidy
RUN go mod vendor

# Construire l'exécutable Go en utilisant les dépendances vendored
RUN CGO_ENABLED=1 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o /app/expenseapp .

# Étape 2: Créer l'image finale avec Nginx
FROM nginx:alpine

# Copier l'exécutable Go depuis l'étape de construction
COPY --from=builder /app/expenseapp /usr/local/bin/expenseapp

# Copier le répertoire frontend
COPY --from=builder /app/frontend /usr/share/nginx/html

# Créer un répertoire pour les données de l'application
RUN mkdir /data

# Définir les variables d'environnement pour l'application Go
ENV PORT=8081
ENV DATADIR=/data

# Supprimer la configuration Nginx par défaut
RUN rm /etc/nginx/conf.d/default.conf

# Créer un script de démarrage
RUN echo '#!/bin/sh' > /entrypoint.sh && \
    echo '/usr/local/bin/expenseapp &' >> /entrypoint.sh && \
    echo 'nginx -g "daemon off;"' >> /entrypoint.sh && \
    chmod +x /entrypoint.sh

# Créer la configuration Nginx
RUN echo 'server {' > /etc/nginx/conf.d/app.conf && \
    echo '    listen 8080;' >> /etc/nginx/conf.d/app.conf && \
    echo '    server_name localhost;' >> /etc/nginx/conf.d/app.conf && \
    echo '    location / {' >> /etc/nginx/conf.d/app.conf && \
    echo '        root   /usr/share/nginx/html;' >> /etc/nginx/conf.d/app.conf && \
    echo '        index  index.html index.htm;' >> /etc/nginx/conf.d/app.conf && \
    echo '        try_files $uri $uri/ /index.html;' >> /etc/nginx/conf.d/app.conf && \
    echo '    }' >> /etc/nginx/conf.d/app.conf && \
    echo '    location /api/ {' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_pass http://127.0.0.1:8081/api/;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header Host $host;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Real-IP $remote_addr;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;' >> /etc/nginx/conf.d/app.conf && \
    echo '        proxy_set_header X-Forwarded-Proto $scheme;' >> /etc/nginx/conf.d/app.conf && \
    echo '    }' >> /etc/nginx/conf.d/app.conf && \
    echo '}' >> /etc/nginx/conf.d/app.conf

# Exposer le port et définir la commande de démarrage
EXPOSE 8080
CMD ["/entrypoint.sh"]
