-include ../setup-env.mk

ARO_HCP_REVISION = $(shell git rev-parse HEAD)
ARO_HCP_IMAGE_TAG ?= $(shell git rev-parse --short=7 HEAD)$(shell git status --porcelain --untracked-files=no | grep --quiet . && echo -dirty)
ARO_HCP_IMAGE_REGISTRY ?= ${ARO_HCP_IMAGE_ACR}.azurecr.io
ARO_HCP_FRONTEND_IMAGE ?= ${ARO_HCP_IMAGE_REGISTRY}/${ARO_HCP_IMAGE_REPOSITORY}:${ARO_HCP_IMAGE_TAG}

.DEFAULT_GOAL := frontend

frontend:
	go build -ldflags="-X github.com/Azure/ARO-HCP/internal/version.CommitSHA=${ARO_HCP_IMAGE_TAG}" -o aro-hcp-frontend .

run:
	DB_URL=$$(az cosmosdb show -n ${DB_NAME} -g ${REGION_RG} --query documentEndpoint -o tsv) && \
	./aro-hcp-frontend --location ${LOCATION} \
		--clusters-service-url http://localhost:8000 \
		--cosmos-name ${DB_NAME} \
		--cosmos-url $${DB_URL}
.PHONY: run

clean:
	rm -f aro-hcp-frontend
.PHONY: clean

build-push: image push

image:
	@bash -c "set -e; pushd .. > /dev/null; \
	          trap 'rm --force image-environment && popd > /dev/null' EXIT; \
	          cp ${ENV_VARS_FILE} image-environment; \
	          docker build . --file frontend/Dockerfile \
	                         --build-arg PLATFORM=linux/amd64 \
	                         --build-arg REVISION=${ARO_HCP_REVISION} \
	                         --build-arg TAG=${ARO_HCP_IMAGE_TAG} \
	                         --tag ${ARO_HCP_FRONTEND_IMAGE}"
.PHONY: image

push: image
	az acr login --name ${ARO_HCP_IMAGE_ACR}
	docker push ${ARO_HCP_FRONTEND_IMAGE}

deploy:
	DIGEST=$$(../get-digest.sh ${ARO_HCP_IMAGE_ACR} arohcpfrontend) \
	FRONTEND_MI_CLIENT_ID=$$(az identity show \
			-g ${RESOURCEGROUP} \
			-n frontend \
			--query clientId -o tsv) && \
	FRONTEND_MI_TENANT_ID=$$(az identity show \
			-g ${RESOURCEGROUP} \
			-n frontend \
			--query tenantId -o tsv) &&\
	IMAGE_PULLER_MI_CLIENT_ID=$$(az identity show \
			-g ${RESOURCEGROUP} \
			-n image-puller \
			--query clientId -o tsv) && \
	IMAGE_PULLER_MI_TENANT_ID=$$(az identity show \
			-g ${RESOURCEGROUP} \
			-n image-puller \
			--query tenantId -o tsv) && \
	SECRET_STORE_MI_CLIENT_ID=$$(az aks show --resource-group ${RESOURCEGROUP} \
			--name ${AKS_NAME} \
			--query addonProfiles.azureKeyvaultSecretsProvider.identity.clientId \
			--output tsv) && \
	TENANT_ID=$(shell az account show --query tenantId --output tsv) && \
	DB_URL=$$(az cosmosdb show -n ${DB_NAME} -g ${REGION_RG} --query documentEndpoint -o tsv) && \
	kubectl create namespace aro-hcp --dry-run=client -o json | kubectl apply -f - && \
	kubectl label namespace aro-hcp "istio.io/rev=${ISTO_TAG}" --overwrite=true && \
	kubectl create namespace mise --dry-run=client -o json | kubectl apply -f - && \
	kubectl label namespace mise "istio.io/rev=${ISTO_TAG}" --overwrite=true && \
	../hack/helm.sh aro-hcp-frontend-dev deploy aro-hcp \
		--set azure.clientId=$${SECRET_STORE_MI_CLIENT_ID} \
		--set azure.tenantId=$${TENANT_ID} \
		--set configMap.databaseName=${DB_NAME} \
		--set configMap.databaseUrl="$${DB_URL}" \
		--set configMap.frontendMiClientId="$${FRONTEND_MI_CLIENT_ID}" \
		--set credsKeyVault.name=${SERVICE_KEY_VAULT} \
		--set credsKeyVault.secret=${CERTIFICATE_NAME} \
		--set serviceAccount.workloadIdentityClientId="$${FRONTEND_MI_CLIENT_ID}" \
		--set serviceAccount.workloadIdentityTenantId="$${FRONTEND_MI_TENANT_ID}" \
		--set pullBinding.workloadIdentityClientId="$${IMAGE_PULLER_MI_CLIENT_ID}" \
		--set pullBinding.workloadIdentityTenantId="$${IMAGE_PULLER_MI_TENANT_ID}" \
		--set configMap.currentVersion=${ARO_HCP_IMAGE_REGISTRY}/${ARO_HCP_IMAGE_REPOSITORY}@$${DIGEST} \
		--set configMap.location=${LOCATION}  \
		--set deployment.imageName=${ARO_HCP_IMAGE_REGISTRY}/${ARO_HCP_IMAGE_REPOSITORY}@$${DIGEST} \
		--set pullBinding.registry=${ARO_HCP_IMAGE_REGISTRY} \
		--set pullBinding.scope=repository:${ARO_HCP_IMAGE_REPOSITORY}:pull \
		--set clustersService.namespace=${CS_NAMESPACE} \
		--set clustersService.serviceAccount=${CS_SERVICE_ACCOUNT_NAME} \
		--set deployMise=${DEPLOY_MISE} \
		--set mise.namespace=mise \
		--set mise.imageRegistry=${ARO_HCP_IMAGE_REGISTRY} \
		--set mise.imageRepository=${MISE_IMAGE_REPOSITORY} \
		--set mise.imageDigest=${MISE_IMAGE_DIGEST} \
		--set mise.tenantId=${MISE_TENANT_ID} \
		--set mise.azureAdInstance=${MISE_AZURE_AD_INSTANCE} \
		--set mise.firstPartyAppId=${MISE_FIRST_PARTY_APP_ID} \
		--set mise.armInstance=${MISE_ARM_INSTANCE} \
		--set mise.armAppId=${MISE_ARM_APP_ID} \
		--set tracing.address=${TRACING_ADDRESS} \
		--set tracing.exporter=${TRACING_EXPORTER} \
		--set audit.tcpAddress=${AUDIT_TCP_ADDRESS} \
		--namespace aro-hcp
.PHONY: deploy

undeploy:
	helm uninstall aro-hcp-frontend-dev --namespace aro-hcp
.PHONY: undeploy

smoke-tests:
	go test -v -count 1 ./utils/frontend_smoke_test.go
