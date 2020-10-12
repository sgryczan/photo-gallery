VERSION=0.0.2
IMAGE_NAME=sgryczan/photouploader
IMAGE_TAG=${VERSION}
GCLOUD_PROJECT=ivory-vim-222512
GCLOUD_BUILD_NAME=photouploader

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the image
	docker build --build-arg VERSION=${VERSION} -t ${IMAGE_NAME}:${IMAGE_TAG} .

.PHONY: push
push: ## Push the image
	docker push ${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: run
run: ## Run the image
	docker run \
		-p 8080:8080 \
		--env-file creds.ignore \
		${IMAGE_NAME}:${IMAGE_TAG}

.PHONY: test
test: ## Run tests
	go test -v ./... -cover

.PHONY: build_gcloud
build_gcloud: ## Submit a build to gcloud
	gcloud builds submit -t gcr.io/${GCLOUD_PROJECT}/${GCLOUD_BUILD_NAME} .

.PHONY: deploy_gcloud
deploy_gcloud: ## Submit a deployment to gcloud
	gcloud run deploy photouploader --image gcr.io/${GCLOUD_PROJECT}/${GCLOUD_BUILD_NAME} --platform managed --region us-central1