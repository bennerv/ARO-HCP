RHCOS_PUBLISHER_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

-include ../setup-env.mk
-include ../setup-templatize-env.mk
-include ../.bingo/Variables.mk

ARO_HCP_REVISION ?= $(shell git rev-parse HEAD)
ARO_HCP_COMMIT_TIME ?= $(shell TZ=UTC git show -s --format=%cd --date=iso-strict-local ${ARO_HCP_REVISION})
ARO_HCP_IMAGE_REGISTRY ?= ${ARO_HCP_IMAGE_ACR}.azurecr.io
RHCOS_PUBLISHER_IMAGE_TAG ?= $(shell DETECT_DIRTY_GIT_WORKTREE=${DETECT_DIRTY_GIT_WORKTREE} DEPLOY_ENV=${DEPLOY_ENV} ../generate-tag.sh)
RHCOS_PUBLISHER_GENERATED_IMAGE_REPOSITORY = $(shell DEPLOY_ENV=${DEPLOY_ENV} BASELINE_REPO=${RHCOS_PUBLISHER_IMAGE_REPOSITORY} ../generate-repo.sh)
RHCOS_PUBLISHER_TAGGED_IMAGE ?= $(ARO_HCP_IMAGE_REGISTRY)/$(RHCOS_PUBLISHER_GENERATED_IMAGE_REPOSITORY):$(RHCOS_PUBLISHER_IMAGE_TAG)

BINARY = rhcos-publisher
LDFLAGS = -ldflags "\
	-X github.com/Azure/ARO-HCP/internal/version.CommitSHA=${ARO_HCP_REVISION}"
RHCOS_PUBLISHER_SOURCES = $(shell find . -name '*.go' -o -name 'go.mod' -o -name 'go.sum')

.DEFAULT_GOAL := build

build: $(BINARY)
.PHONY: build

$(BINARY): $(RHCOS_PUBLISHER_SOURCES)
	go build $(LDFLAGS) -o $(BINARY) .

clean:
	rm -f $(BINARY)
.PHONY: clean

image:
	cd $(RHCOS_PUBLISHER_DIR)/.. && \
	$(CONTAINER_ENGINE) build . --file rhcos-publisher/Dockerfile \
		--build-arg PLATFORM=linux/amd64 \
		--build-arg ARO_HCP_REVISION=${ARO_HCP_REVISION} \
		--build-arg ARO_HCP_COMMIT_TIME=${ARO_HCP_COMMIT_TIME} \
		--tag ${RHCOS_PUBLISHER_TAGGED_IMAGE}
.PHONY: image

build-and-push: image $(ORAS) $(YQ)
	$(acr-login)
	@LOCAL_DIGEST=$$($(CONTAINER_ENGINE) image inspect --format '{{.Id}}' ${RHCOS_PUBLISHER_TAGGED_IMAGE} | sed 's/sha256://'); \
	REMOTE_DIGEST=$$($(ORAS) manifest fetch ${RHCOS_PUBLISHER_TAGGED_IMAGE} 2>/dev/null | $(YQ) .config.digest | sed 's/sha256://'); \
	if [ "$$LOCAL_DIGEST" = "$$REMOTE_DIGEST" ]; then \
		echo "Image ${RHCOS_PUBLISHER_TAGGED_IMAGE} already up to date in registry, skipping push"; \
	else \
		$(CONTAINER_ENGINE) push ${RHCOS_PUBLISHER_TAGGED_IMAGE}; \
	fi
.PHONY: build-and-push

MAKE_RUN_TS := $(shell date +%s)
OVERRIDE_CONFIG_FILE ?= /tmp/rhcos-publisher-override-config-$(MAKE_RUN_TS).yaml

record-override: $(YQ) $(ORAS)
	@az acr login --name ${ARO_HCP_IMAGE_ACR} --expose-token --output tsv --query accessToken | \
		$(ORAS) login ${ARO_HCP_IMAGE_REGISTRY} \
			--username 00000000-0000-0000-0000-000000000000 \
			--password-stdin

	@DIGEST=$$($(ORAS) manifest fetch --descriptor ${RHCOS_PUBLISHER_TAGGED_IMAGE} | $(YQ) .digest); \
	$(YQ) eval -n "\
		.clouds.dev.environments.$(DEPLOY_ENV).defaults.rhcosPublisher.image.registry = \"$(ARO_HCP_IMAGE_REGISTRY)\" | \
		.clouds.dev.environments.$(DEPLOY_ENV).defaults.rhcosPublisher.image.repository = \"$(RHCOS_PUBLISHER_GENERATED_IMAGE_REPOSITORY)\" | \
		.clouds.dev.environments.$(DEPLOY_ENV).defaults.rhcosPublisher.image.digest = \"$$DIGEST\" \
	" > $(OVERRIDE_CONFIG_FILE)
.PHONY: record-override

record-latest-override: $(YQ)
	@../hack/record-latest-image-override.sh \
		"$(ARO_HCP_IMAGE_ACR)" "$(ARO_HCP_IMAGE_REGISTRY)" \
		"$(RHCOS_PUBLISHER_GENERATED_IMAGE_REPOSITORY)" "$(DEPLOY_ENV)" \
		"rhcosPublisher" "$(OVERRIDE_CONFIG_FILE)" "$(YQ)"
.PHONY: record-latest-override
