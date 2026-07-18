# Hackathon Guide to `az aro hcp`

This guide walks you through using the `az aro hcp` CLI extension to interact with Azure Red Hat OpenShift Hosted Control Plane (ARO HCP) resources. It covers installation, cluster operations, version queries, and nodepool management.

## Installing the extension

Build and install the extension from the generated wheel:

```bash
cd ~/workspace/azure-cli-extensions/src/aro-hcp
python -m pip install -U build
python -m build --wheel
az extension add --source dist/*.whl -y
```

Verify the extension is installed:

```bash
az aro hcp -h
```

## Clusters

### List clusters

List all ARO HCP clusters in a resource group.

```bash
az aro hcp cluster list --resource-group <resource-group>
```

**Example:**

```bash
az aro hcp cluster list --resource-group private-keyvault-sxf56l
```

**Example output (trimmed):**

```json
[
  {
    "name": "private-kv-cluster",
    "location": "uksouth",
    "resourceGroup": "private-keyvault-sxf56l",
    "properties": {
      "api": {
        "url": "https://api.m4u1n3w1y6c4c0v.xsqu.uksouth.aroapp-hcp.io:443",
        "visibility": "Public"
      },
      "console": {
        "url": "https://console-openshift-console.apps.aro.m4u1n3w1y6c4c0v.xsqu.uksouth.aroapp-hcp.io"
      },
      "network": {
        "hostPrefix": 23,
        "machineCidr": "10.0.0.0/16",
        "networkType": "OVNKubernetes",
        "podCidr": "10.128.0.0/14",
        "serviceCidr": "172.30.0.0/16"
      },
      "provisioningState": "Succeeded",
      "version": {
        "channelGroup": "candidate",
        "id": "4.20.29"
      }
    },
    "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters"
  }
]
```

> **Tip:** Append `-o table` for a compact summary view: `az aro hcp cluster list --resource-group private-keyvault-sxf56l -o table`

### Show a cluster

Show the details of a specific ARO HCP cluster.

```bash
az aro hcp cluster show --resource-group <resource-group> --name <cluster-name>
```

**Example:**

```bash
az aro hcp cluster show --resource-group private-keyvault-sxf56l --name private-kv-cluster
```

**Example output (trimmed):**

```json
{
  "name": "private-kv-cluster",
  "location": "uksouth",
  "resourceGroup": "private-keyvault-sxf56l",
  "properties": {
    "api": {
      "url": "https://api.m4u1n3w1y6c4c0v.xsqu.uksouth.aroapp-hcp.io:443",
      "visibility": "Public"
    },
    "console": {
      "url": "https://console-openshift-console.apps.aro.m4u1n3w1y6c4c0v.xsqu.uksouth.aroapp-hcp.io"
    },
    "dns": {
      "baseDomain": "xsqu.uksouth.aroapp-hcp.io",
      "baseDomainPrefix": "m4u1n3w1y6c4c0v"
    },
    "network": {
      "hostPrefix": 23,
      "machineCidr": "10.0.0.0/16",
      "networkType": "OVNKubernetes",
      "podCidr": "10.128.0.0/14",
      "serviceCidr": "172.30.0.0/16"
    },
    "provisioningState": "Succeeded",
    "version": {
      "channelGroup": "candidate",
      "id": "4.20.29"
    }
  },
  "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters"
}
```

> **Tip:** Use `-o table` for a compact view: `az aro hcp cluster show --resource-group private-keyvault-sxf56l --name private-kv-cluster -o table`

### Request admin credentials

Request a temporary admin kubeconfig for a cluster. The credential includes a client certificate and key and has a limited expiration window.

```bash
az aro hcp cluster request-admin-credential --resource-group <resource-group> --name <cluster-name>
```

By default, the kubeconfig is included in the JSON output. Use `--file` to write it directly to a file:

```bash
az aro hcp cluster request-admin-credential \
  --resource-group private-keyvault-sxf56l \
  --name private-kv-cluster \
  --file ~/.kube/aro-hcp-config
```

The file will contain a valid kubeconfig that you can use with `kubectl` or `oc`:

```bash
export KUBECONFIG=~/.kube/aro-hcp-config
kubectl get nodes
```

Without `--file`, the full output includes both the expiration and the kubeconfig content:

```json
{
  "expirationTimestamp": "2026-07-19T18:09:30.54144Z",
  "kubeconfig": "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0t...\n    server: https://api.m4u1n3w1y6c4c0v.xsqu.uksouth.aroapp-hcp.io:443\n  name: cluster\n..."
}
```

### Revoke credentials

Revoke all admin credentials for a cluster. This invalidates any kubeconfigs previously obtained via `request-admin-credential`.

```bash
az aro hcp cluster revoke-credential --resource-group <resource-group> --name <cluster-name>
```

**Example:**

```bash
az aro hcp cluster revoke-credential --resource-group private-keyvault-sxf56l --name private-kv-cluster
```

After revoking, any existing admin kubeconfigs will stop working. You can request a new credential with `az aro hcp cluster request-admin-credential`.

### Create a cluster

Creating a cluster requires significant prerequisite infrastructure: a VNet with two subnets, an NSG, a KeyVault with an encryption key, 12 managed identities with specific role assignments, and the cluster itself.

#### Step 1 — Set variables

```bash
LOCATION="uksouth"
CLUSTER_NAME="my-cluster"
RESOURCE_GROUP="my-cluster-rg"
MANAGED_RG="${CLUSTER_NAME}-managed"

VNET_NAME="customer-vnet"
SUBNET_NAME="customer-subnet-1"
VNET_INTEGRATION_SUBNET_NAME="customer-vnet-integration-subnet"
NSG_NAME="customer-nsg"

SUBSCRIPTION_ID=$(az account show --query id -o tsv)
```

#### Step 2 — Register the resource provider

```bash
az provider register --namespace Microsoft.RedHatOpenShift --wait
```

#### Step 3 — Create resource group

```bash
az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"
```

#### Step 4 — Create networking (NSG, VNet, subnets)

```bash
# Create the NSG
az network nsg create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${NSG_NAME}"

# Create the VNet
az network vnet create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${VNET_NAME}" \
  --address-prefix "10.0.0.0/16"

# Create the cluster subnet with the NSG attached and private endpoint policies disabled
az network vnet subnet create \
  --resource-group "${RESOURCE_GROUP}" \
  --vnet-name "${VNET_NAME}" \
  --name "${SUBNET_NAME}" \
  --address-prefix "10.0.0.0/24" \
  --network-security-group "${NSG_NAME}" \
  --disable-private-endpoint-network-policies true

# Create the VNet integration subnet with the ARO HCP delegation
az network vnet subnet create \
  --resource-group "${RESOURCE_GROUP}" \
  --vnet-name "${VNET_NAME}" \
  --name "${VNET_INTEGRATION_SUBNET_NAME}" \
  --address-prefix "10.0.1.0/24" \
  --delegations "Microsoft.RedHatOpenShift/hcpOpenShiftClusters"
```

#### Step 5 — Create KeyVault and etcd encryption key

```bash
# Generate a unique KeyVault name
KV_NAME="cust-kv-$(head -c 6 /dev/urandom | base64 | tr -dc 'a-z0-9' | head -c 13)"

az keyvault create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${KV_NAME}" \
  --enable-rbac-authorization true \
  --public-network-access Enabled

az keyvault key create \
  --vault-name "${KV_NAME}" \
  --name "etcd-data-kms-encryption-key" \
  --kty RSA \
  --size 2048

# Get the key version for the cluster create command
ETCD_KEY_VERSION=$(az keyvault key show \
  --vault-name "${KV_NAME}" \
  --name "etcd-data-kms-encryption-key" \
  --query "key.kid" -o tsv | rev | cut -d'/' -f1 | rev)
```

#### Step 6 — Create managed identities

ARO HCP requires 12 user-assigned managed identities: 9 for control plane operators, 3 for data plane operators. Each needs specific Azure role assignments.

```bash
# Helper to get resource IDs
SUBNET_ID="/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Network/virtualNetworks/${VNET_NAME}/subnets/${SUBNET_NAME}"
VNET_ID="/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Network/virtualNetworks/${VNET_NAME}"
NSG_ID="/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Network/networkSecurityGroups/${NSG_NAME}"
KV_ID=$(az keyvault show --name "${KV_NAME}" --query id -o tsv)

# --- Service Managed Identity ---
SERVICE_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-service" --query id -o tsv)
SERVICE_MI_PRINCIPAL=$(az identity show --ids "${SERVICE_MI}" --query principalId -o tsv)

# Service MI role: Azure Red Hat OpenShift Hosted Control Planes Service Managed Identity
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "c0ff367d-66d8-445e-917c-583feb0ef0d4" --scope "${VNET_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "c0ff367d-66d8-445e-917c-583feb0ef0d4" --scope "${NSG_ID}"

# --- Control Plane: cluster-api-azure ---
CAPI_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-cluster-api-azure" --query id -o tsv)
CAPI_MI_PRINCIPAL=$(az identity show --ids "${CAPI_MI}" --query principalId -o tsv)

# HCP Cluster API Provider role on subnet
az role assignment create --assignee-object-id "${CAPI_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "88366f10-ed47-4cc0-9fab-c8a06148393e" --scope "${SUBNET_ID}"
# Service MI gets Reader on this MI
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${CAPI_MI}"

# --- Control Plane: control-plane ---
CP_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-control-plane" --query id -o tsv)
CP_MI_PRINCIPAL=$(az identity show --ids "${CP_MI}" --query principalId -o tsv)

# HCP Control Plane Operator role on VNet and NSG
az role assignment create --assignee-object-id "${CP_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "fc0c873f-45e9-4d0d-a7d1-585aab30c6ed" --scope "${VNET_ID}"
az role assignment create --assignee-object-id "${CP_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "fc0c873f-45e9-4d0d-a7d1-585aab30c6ed" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${CP_MI}"

# --- Control Plane: cloud-controller-manager ---
CCM_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-cloud-controller-manager" --query id -o tsv)
CCM_MI_PRINCIPAL=$(az identity show --ids "${CCM_MI}" --query principalId -o tsv)

# Cloud Controller Manager role on subnet and NSG
az role assignment create --assignee-object-id "${CCM_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "a1f96423-95ce-4224-ab27-4e3dc72facd4" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${CCM_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "a1f96423-95ce-4224-ab27-4e3dc72facd4" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${CCM_MI}"

# --- Control Plane: ingress ---
INGRESS_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-ingress" --query id -o tsv)
INGRESS_MI_PRINCIPAL=$(az identity show --ids "${INGRESS_MI}" --query principalId -o tsv)

# Cluster Ingress Operator role on subnet
az role assignment create --assignee-object-id "${INGRESS_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "0336e1d3-7a87-462b-b6db-342b63f7802c" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${INGRESS_MI}"

# --- Control Plane: disk-csi-driver ---
DISK_CSI_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-disk-csi-driver" --query id -o tsv)
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${DISK_CSI_MI}"

# --- Control Plane: file-csi-driver ---
FILE_CSI_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-file-csi-driver" --query id -o tsv)
FILE_CSI_MI_PRINCIPAL=$(az identity show --ids "${FILE_CSI_MI}" --query principalId -o tsv)

# File Storage Operator role on subnet and NSG
az role assignment create --assignee-object-id "${FILE_CSI_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "0d7aedc0-15fd-4a67-a412-efad370c947e" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${FILE_CSI_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "0d7aedc0-15fd-4a67-a412-efad370c947e" --scope "${NSG_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${FILE_CSI_MI}"

# --- Control Plane: image-registry ---
IMAGE_REG_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-image-registry" --query id -o tsv)
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${IMAGE_REG_MI}"

# --- Control Plane: cloud-network-config ---
CLOUD_NET_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-cloud-network-config" --query id -o tsv)
CLOUD_NET_MI_PRINCIPAL=$(az identity show --ids "${CLOUD_NET_MI}" --query principalId -o tsv)

# Network Operator role on subnet and VNet
az role assignment create --assignee-object-id "${CLOUD_NET_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "be7a6435-15ae-4171-8f30-4a343eff9e8f" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${CLOUD_NET_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "be7a6435-15ae-4171-8f30-4a343eff9e8f" --scope "${VNET_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${CLOUD_NET_MI}"

# --- Control Plane: kms ---
KMS_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-kms" --query id -o tsv)
KMS_MI_PRINCIPAL=$(az identity show --ids "${KMS_MI}" --query principalId -o tsv)

# Key Vault Crypto User role on the KeyVault
az role assignment create --assignee-object-id "${KMS_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "12338af0-0e69-4776-bea7-57ae8d297424" --scope "${KV_ID}"
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "acdd72a7-3385-48ef-bd42-f606fba81ae7" --scope "${KMS_MI}"

# --- Data Plane: disk-csi-driver ---
DP_DISK_CSI_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-dp-disk-csi-driver" --query id -o tsv)

# Federated Credential role for service MI
az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "ef318e2a-8334-4a05-9e4a-295a196c6a6e" --scope "${DP_DISK_CSI_MI}"

# --- Data Plane: file-csi-driver ---
DP_FILE_CSI_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-dp-file-csi-driver" --query id -o tsv)
DP_FILE_CSI_MI_PRINCIPAL=$(az identity show --ids "${DP_FILE_CSI_MI}" --query principalId -o tsv)

az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "ef318e2a-8334-4a05-9e4a-295a196c6a6e" --scope "${DP_FILE_CSI_MI}"
# File Storage Operator role on subnet and NSG
az role assignment create --assignee-object-id "${DP_FILE_CSI_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "0d7aedc0-15fd-4a67-a412-efad370c947e" --scope "${SUBNET_ID}"
az role assignment create --assignee-object-id "${DP_FILE_CSI_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "0d7aedc0-15fd-4a67-a412-efad370c947e" --scope "${NSG_ID}"

# --- Data Plane: image-registry ---
DP_IMAGE_REG_MI=$(az identity create -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}-dp-image-registry" --query id -o tsv)

az role assignment create --assignee-object-id "${SERVICE_MI_PRINCIPAL}" --assignee-principal-type ServicePrincipal \
  --role "ef318e2a-8334-4a05-9e4a-295a196c6a6e" --scope "${DP_IMAGE_REG_MI}"
```

#### Step 7 — Create the cluster

```bash
VNET_INTEGRATION_SUBNET_ID="/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Network/virtualNetworks/${VNET_NAME}/subnets/${VNET_INTEGRATION_SUBNET_NAME}"

az aro hcp cluster create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${CLUSTER_NAME}" \
  --location "${LOCATION}" \
  --version "4.20" \
  --channel-group candidate \
  --subnet-id "${SUBNET_ID}" \
  --vnet-integration-subnet-id "${VNET_INTEGRATION_SUBNET_ID}" \
  --nsg "${NSG_ID}" \
  --managed-resource-group-name "${MANAGED_RG}" \
  --key-management-mode CustomerManaged \
  --etcd-encryption-type KMS \
  --kms-vault-name "${KV_NAME}" \
  --vault-visibility Public \
  --kms-active-key "{name:etcd-data-kms-encryption-key,version:${ETCD_KEY_VERSION}}" \
  --user-assigned-identities "{${SERVICE_MI}:{},${CAPI_MI}:{},${CP_MI}:{},${CCM_MI}:{},${INGRESS_MI}:{},${DISK_CSI_MI}:{},${FILE_CSI_MI}:{},${IMAGE_REG_MI}:{},${CLOUD_NET_MI}:{},${KMS_MI}:{}}" \
  --operators-authentication "{user-assigned-identities:{control-plane-operators:{cluster-api-azure:${CAPI_MI},control-plane:${CP_MI},cloud-controller-manager:${CCM_MI},ingress:${INGRESS_MI},disk-csi-driver:${DISK_CSI_MI},file-csi-driver:${FILE_CSI_MI},image-registry:${IMAGE_REG_MI},cloud-network-config:${CLOUD_NET_MI},kms:${KMS_MI}},data-plane-operators:{disk-csi-driver:${DP_DISK_CSI_MI},file-csi-driver:${DP_FILE_CSI_MI},image-registry:${DP_IMAGE_REG_MI}},service-managed-identity:${SERVICE_MI}}}"
```

This is a long-running operation. Cluster creation typically takes 15-30 minutes.

<!-- TODO: Evaluate matching the classic ARO `az aro create` behavior where
     infrastructure (VNet, NSG, identities, role assignments) is created
     automatically by the CLI rather than requiring manual pre-creation. -->

### Update a cluster

Update cluster properties such as version, tags, or node drain timeout.

```bash
az aro hcp cluster update --resource-group <resource-group> --name <cluster-name> [options]
```

**Example — add a tag:**

```bash
az aro hcp cluster update \
  --resource-group private-keyvault-sxf56l \
  --name private-kv-cluster \
  --tags env=hackathon
```

**Example — update the cluster version:**

```bash
az aro hcp cluster update \
  --resource-group private-keyvault-sxf56l \
  --name private-kv-cluster \
  --version 4.21
```

### Delete a cluster

Delete a cluster and all its resources.

```bash
az aro hcp cluster delete --resource-group <resource-group> --name <cluster-name> -y
```

**Example:**

```bash
az aro hcp cluster delete --resource-group hackathon-test-rg --name hackathon-test -y
```

This is a long-running operation. Use `--no-wait` to return immediately.

> **Note:** Deleting the cluster does not delete the resource group, networking, KeyVault, or managed identities you created as prerequisites. To fully clean up, delete the resource group: `az group delete --name <resource-group> -y`

## Versions

Versions are scoped to a location (Azure region), not a resource group.

### List versions

List all available ARO HCP OpenShift versions in a given location.

```bash
az aro hcp version list --location <location>
```

**Example:**

```bash
az aro hcp version list --location uksouth
```

**Example output (trimmed to first few entries):**

```json
[
  {
    "name": "4.19.0",
    "properties": {
      "channelGroup": "stable",
      "enabled": true
    },
    "type": "Microsoft.RedHatOpenShift/locations/hcpOpenShiftVersions"
  },
  {
    "name": "4.19.1",
    "properties": {
      "channelGroup": "stable",
      "enabled": true
    },
    "type": "Microsoft.RedHatOpenShift/locations/hcpOpenShiftVersions"
  }
]
```

> **Tip:** Use `-o table` for a quick overview: `az aro hcp version list --location uksouth -o table`

### Show a version

Get the details of a specific OpenShift version. Use `--version` (not `--name`) to specify the version string.

```bash
az aro hcp version show --location <location> --version <version>
```

**Example:**

```bash
az aro hcp version show --location uksouth --version 4.19.0
```

**Example output:**

```json
{
  "name": "4.19.0",
  "properties": {
    "channelGroup": "stable",
    "enabled": true
  },
  "type": "Microsoft.RedHatOpenShift/locations/hcpOpenShiftVersions"
}
```

## Node pools

Node pools are a subresource of a cluster, so all commands are under `az aro hcp cluster nodepool`.

### Create a node pool

Create a new node pool on an existing cluster. At minimum you need to specify the cluster, a name, replica count, VM size, and version. The node pool will use the subnet of the parent cluster by default.

```bash
az aro hcp cluster nodepool create \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <nodepool-name> \
  --replicas <count> \
  --vm-size <vm-size> \
  --version <openshift-version> \
  --channel-group <channel-group>
```

**Example:**

```bash
az aro hcp cluster nodepool create \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name np-3 \
  --replicas 1 \
  --vm-size Standard_D8s_v3 \
  --version 4.20.29 \
  --channel-group candidate
```

> **Note:** The `--version` and `--channel-group` should match the cluster's version or be a supported upgrade target from `az aro hcp version list`.

This is a long-running operation. By default the CLI waits for it to complete. Use `--no-wait` to return immediately and check status later with `az aro hcp cluster nodepool show`.

**Example output:**

```json
{
  "name": "np-3",
  "location": "uksouth",
  "resourceGroup": "private-keyvault-sxf56l",
  "properties": {
    "autoRepair": true,
    "platform": {
      "enableEncryptionAtHost": false,
      "osDisk": {
        "diskStorageAccountType": "Premium_LRS",
        "diskType": "Managed",
        "sizeGiB": 64
      },
      "vmSize": "Standard_D8s_v3"
    },
    "provisioningState": "Succeeded",
    "replicas": 1,
    "version": {
      "channelGroup": "candidate",
      "id": "4.20.29"
    }
  },
  "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters/nodePools"
}
```

### List node pools

List all node pools for a given cluster.

```bash
az aro hcp cluster nodepool list --resource-group <resource-group> --cluster-name <cluster-name>
```

**Example:**

```bash
az aro hcp cluster nodepool list --resource-group private-keyvault-sxf56l --cluster-name private-kv-cluster
```

**Example output (trimmed):**

```json
[
  {
    "name": "np-1",
    "location": "uksouth",
    "resourceGroup": "private-keyvault-sxf56l",
    "properties": {
      "autoRepair": true,
      "platform": {
        "enableEncryptionAtHost": false,
        "osDisk": {
          "diskStorageAccountType": "StandardSSD_LRS",
          "diskType": "Managed",
          "sizeGiB": 64
        },
        "vmSize": "Standard_D8s_v3"
      },
      "provisioningState": "Succeeded",
      "replicas": 2,
      "version": {
        "channelGroup": "candidate",
        "id": "4.20.29"
      }
    },
    "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters/nodePools"
  },
  {
    "name": "np-2",
    "location": "uksouth",
    "resourceGroup": "private-keyvault-sxf56l",
    "properties": {
      "autoRepair": true,
      "platform": {
        "enableEncryptionAtHost": false,
        "osDisk": {
          "diskStorageAccountType": "Premium_LRS",
          "diskType": "Managed",
          "sizeGiB": 64
        },
        "vmSize": "Standard_D8s_v3"
      },
      "provisioningState": "Succeeded",
      "replicas": 1,
      "version": {
        "channelGroup": "candidate",
        "id": "4.20.29"
      }
    },
    "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters/nodePools"
  }
]
```

> **Tip:** Use `-o table` for a compact view: `az aro hcp cluster nodepool list --resource-group private-keyvault-sxf56l --cluster-name private-kv-cluster -o table`

### Show a node pool

Get the details of a specific node pool.

```bash
az aro hcp cluster nodepool show --resource-group <resource-group> --cluster-name <cluster-name> --name <nodepool-name>
```

**Example:**

```bash
az aro hcp cluster nodepool show --resource-group private-keyvault-sxf56l --cluster-name private-kv-cluster --name np-1
```

**Example output:**

```json
{
  "name": "np-1",
  "location": "uksouth",
  "resourceGroup": "private-keyvault-sxf56l",
  "properties": {
    "autoRepair": true,
    "platform": {
      "enableEncryptionAtHost": false,
      "osDisk": {
        "diskStorageAccountType": "StandardSSD_LRS",
        "diskType": "Managed",
        "sizeGiB": 64
      },
      "subnetId": "/subscriptions/.../subnets/customer-subnet-1",
      "vmSize": "Standard_D8s_v3"
    },
    "provisioningState": "Succeeded",
    "replicas": 2,
    "version": {
      "channelGroup": "candidate",
      "id": "4.20.29"
    }
  },
  "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters/nodePools"
}
```

### Update a node pool

Update node pool properties such as replica count, version, labels, or taints.

```bash
az aro hcp cluster nodepool update \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <nodepool-name> \
  [options]
```

**Example — scale replicas:**

```bash
az aro hcp cluster nodepool update \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name np-1 \
  --replicas 3
```

**Example — add labels:**

```bash
az aro hcp cluster nodepool update \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name np-1 \
  --labels "[{key:env,value:hackathon}]"
```

### Delete a node pool

Delete a node pool from a cluster.

```bash
az aro hcp cluster nodepool delete \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <nodepool-name> \
  -y
```

**Example:**

```bash
az aro hcp cluster nodepool delete \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name np-3 \
  -y
```

This is a long-running operation. Use `--no-wait` to return immediately.

## External authentication

External authentication allows you to configure an external identity provider (e.g. Microsoft Entra ID) for cluster authentication. Commands are under `az aro hcp cluster external-auth`.

> **Important:** A cluster can only have a single external auth configuration. Attempting to create a second one will fail.

### Prerequisites

You need a Microsoft Entra ID app registration to use as the OIDC client. The app registration tells Entra ID how to issue tokens that the cluster's kube-apiserver will accept.

```bash
# Create the app registration:
#   --display-name: human-readable name shown in the Azure portal
#   --requested-access-token-version 2: issue v2 tokens whose "iss" claim uses
#     "https://login.microsoftonline.com/{tenant}/v2.0" — this must match the
#     --issuer-url we configure on the external auth
APP_ID=$(az ad app create \
  --display-name "aro-hcp-external-auth" \
  --requested-access-token-version 2 \
  --query appId -o tsv)
TENANT_ID=$(az account show --query tenantId -o tsv)

# Create a service principal for the app. Without this, Entra ID does not
# recognize the app as a resource and will reject token requests.
az ad sp create --id "${APP_ID}"
```

### Create an external auth

```bash
az aro hcp cluster external-auth create \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name> \
  --issuer-url "https://login.microsoftonline.com/${TENANT_ID}/v2.0" \
  --issuer-audience "${APP_ID}" \
  --username-claim oid \
  --username-prefix-policy Prefix \
  --username-prefix "myprefix-" \
  --claim groups \
  --clients "[{client-id:${APP_ID},component:{name:cli,auth-client-namespace:openshift-console},type:Public}]"
```

**Example:**

```bash
az aro hcp cluster external-auth create \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name hackathon-auth \
  --issuer-url "https://login.microsoftonline.com/${TENANT_ID}/v2.0" \
  --issuer-audience "${APP_ID}" \
  --username-claim oid \
  --username-prefix-policy Prefix \
  --username-prefix "hackathon-" \
  --claim groups \
  --clients "[{client-id:${APP_ID},component:{name:cli,auth-client-namespace:openshift-console},type:Public}]"
```

This is a long-running operation. Use `--no-wait` to return immediately. The external auth may sit in `Accepted` state for several minutes before reaching `Succeeded` as the kube-apiserver restarts to pick up the new configuration.

You can check the provisioning state with:

```bash
az aro hcp cluster external-auth show \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name> \
  --query properties.provisioningState -o tsv
```

**Example output on success:**

```json
{
  "name": "hackathon-auth",
  "properties": {
    "claim": {
      "mappings": {
        "groups": {
          "claim": "groups"
        },
        "username": {
          "claim": "oid",
          "prefix": "hackathon-",
          "prefixPolicy": "Prefix"
        }
      }
    },
    "clients": [
      {
        "clientId": "<app-id>",
        "component": {
          "authClientNamespace": "openshift-console",
          "name": "cli"
        },
        "type": "Public"
      }
    ],
    "issuer": {
      "audiences": ["<app-id>"],
      "url": "https://login.microsoftonline.com/<tenant-id>/v2.0"
    },
    "provisioningState": "Succeeded"
  },
  "type": "Microsoft.RedHatOpenShift/hcpOpenShiftClusters/externalAuths"
}
```

### Show an external auth

Get the details of a specific external auth configuration.

```bash
az aro hcp cluster external-auth show \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name>
```

**Example:**

```bash
az aro hcp cluster external-auth show \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name hackathon-auth
```

### List external auth

List all external auth configurations on a cluster. A cluster can only have a single external auth instance.

```bash
az aro hcp cluster external-auth list --resource-group <resource-group> --cluster-name <cluster-name>
```

**Example:**

```bash
az aro hcp cluster external-auth list --resource-group private-keyvault-sxf56l --cluster-name private-kv-cluster
```

### Update an external auth

You can update an existing external auth configuration. For example, to change the username claim from `sub` to `oid`:

```bash
az aro hcp cluster external-auth update \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name> \
  --username-claim oid
```

You can also use `--set` to update arbitrary properties by path:

```bash
az aro hcp cluster external-auth update \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name> \
  --set properties.claim.mappings.username.claim=oid
```

This is a long-running operation — the kube-apiserver restarts to pick up the change.

### Delete an external auth

Delete an external auth configuration from a cluster.

```bash
az aro hcp cluster external-auth delete \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name> \
  -y
```

**Example:**

```bash
az aro hcp cluster external-auth delete \
  --resource-group private-keyvault-sxf56l \
  --cluster-name private-kv-cluster \
  --name hackathon-auth \
  -y
```

### Logging in with Azure authentication

Once external auth is configured, you can authenticate to the cluster using your Azure identity instead of the temporary admin credential.

First, get the admin kubeconfig to find the API server URL:

```bash
az aro hcp cluster request-admin-credential \
  --resource-group <resource-group> \
  --name <cluster-name> \
  --file ~/.kube/aro-hcp-config
```

Before you can log in with your Azure identity, you need to grant it permissions on the cluster. Using the admin kubeconfig, create a ClusterRoleBinding that gives your user `cluster-admin`:

```bash
export KUBECONFIG=~/.kube/aro-hcp-config

# Get your Azure object ID (the "oid" claim — Object ID of the signed-in user in Entra ID)
OBJECT_ID=$(az ad signed-in-user show --query id -o tsv)

# Create a ClusterRoleBinding for your user with the prefix you configured
kubectl create clusterrolebinding hackathon-cluster-admin \
  --clusterrole=cluster-admin \
  --user="hackathon-${OBJECT_ID}"
```

> **Note:** The `--user` value must match the prefix you set in `--username-prefix` during external auth creation, followed by the value of the `--username-claim`. With `--username-claim oid`, this is the Azure object ID returned by `az ad signed-in-user show --query id`.

Then log in using an Azure token. ARO HCP with external auth uses Entra ID directly (not the OpenShift OAuth server), so you need to acquire a token from Azure and pass it to `oc login`:

```bash
# Get the API server URL from the admin kubeconfig
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

# Acquire a token for the app registration
TOKEN=$(az account get-access-token --resource "${APP_ID}" --query accessToken -o tsv)

# Log in with the token
oc login --server="${API_SERVER}" --token="${TOKEN}"
```

You can verify by checking who you are:

```bash
oc whoami
```

The output will show your username with the prefix you configured (e.g. `hackathon-<object-id>`).

### Cleanup

When you are done, delete the external auth and the app registration:

```bash
az aro hcp cluster external-auth delete \
  --resource-group <resource-group> \
  --cluster-name <cluster-name> \
  --name <auth-name>

az ad app delete --id "${APP_ID}"
```
