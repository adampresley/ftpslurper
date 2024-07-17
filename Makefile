.DEFAULT_GOAL := help
.PHONY: help

VERSION=$(shell cat ./VERSION)

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}'" -mod=mod -o ftpslurper .

run: ## Run the application using CompileDaemon
	CompileDaemon -directory=./ -pattern="(.+\\.go|.+\\.tmpl|.+\\.css|.+\\.js|.+\\.md)" -command="./ftpslurper"

start: ## Run everything locally in Docker
	docker compose up --build

start-backplane: ## Run dependant services in Docker 
	docker compose up -d postgres 

stop-backplane: ## Stop dependant services in Docker 
	docker compose stop postgres

docker-create-builder: ## Create a builder for multi-architecture builds. Only needed once per machine
	docker buildx create --name mybuilder --driver docker-container --bootstrap

docker-tag: ## Builds a docker image and tags a release. It is then pushed up to Docker. GITHUB_TOKEN must be defined as an environment variable. make username="username" docker-tag
	@echo "Creating tag ${VERSION}"
	git tag -a ${VERSION} -m "Release ${VERSION}"
	git push origin ${VERSION}
	@echo "Building ${VERSION}"
	echo $$GITHUB_TOKEN | docker login ghcr.io -u ${username} --password-stdin && docker buildx use mybuilder
	docker buildx build -f Dockerfile --platform linux/amd64 -t ghcr.io/adampresley/ftpslurper:${VERSION} --push .

