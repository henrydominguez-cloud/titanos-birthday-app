# Titan OS birthday-app — developer entrypoints.
IMAGE       ?= birthday-app:0.1.0
CLUSTER     ?= titanos
APP_NS      ?= birthday
CHART       ?= helm/birthday-app

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ---- application ----------------------------------------------------------
.PHONY: test
test: ## Run Go unit tests (in a container; no local Go needed)
	docker run --rm -v "$(PWD)/app":/src -w /src golang:1.23-alpine \
	  sh -c "go mod tidy && go vet ./... && go test ./... -count=1"

.PHONY: build
build: ## Build the container image
	docker build --provenance=false -t $(IMAGE) ./app

.PHONY: run-local
run-local: ## Run the API locally in memory mode (needs the image)
	docker run --rm -p 8080:8080 -e STORE_BACKEND=memory $(IMAGE)

## ---- kubernetes (kind + kong) --------------------------------------------
.PHONY: up
up: ## Create kind cluster, install Kong, build+load+deploy the app
	bash kind/setup.sh

.PHONY: down
down: ## Delete the kind cluster
	bash kind/teardown.sh

.PHONY: redeploy
redeploy: build ## Rebuild image, reload into kind, roll the deployment
	kind load docker-image $(IMAGE) --name $(CLUSTER)
	kubectl -n $(APP_NS) rollout restart deploy/birthday-app
	kubectl -n $(APP_NS) rollout status deploy/birthday-app

.PHONY: lint
lint: ## helm lint + template render
	helm lint $(CHART)
	helm template birthday-app $(CHART) >/dev/null && echo "template OK"

.PHONY: helm-test
helm-test: ## Run the in-cluster Helm smoke test
	helm test birthday-app -n $(APP_NS)

.PHONY: smoke
smoke: ## Hit the API through Kong from the host
	bash scripts/smoke.sh
