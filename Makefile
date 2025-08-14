# Nom de l'image Docker
IMAGE_NAME = expenseapp

# Commande par défaut
all: build

# Construire l'image Docker
build:
	docker build -t $(IMAGE_NAME) .

# Lancer le conteneur Docker
run:
	docker run -d -p 8080:8080 --name $(IMAGE_NAME) -v $(PWD)/data:/data $(IMAGE_NAME)

# Arrêter et supprimer le conteneur Docker
stop:
	docker stop $(IMAGE_NAME) || true
	docker rm $(IMAGE_NAME) || true

# Afficher les logs du conteneur
logs:
	docker logs -f $(IMAGE_NAME)

# Nettoyer : arrêter le conteneur et supprimer l'image
clean: stop
	docker rmi $(IMAGE_NAME) || true

.PHONY: all build run stop logs clean
