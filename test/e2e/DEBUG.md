## Debug.sh Instructions

This script provides step-by-step instructions for debugging clusters across Microsoft and Red Hat tenants using Azure CLI.

### Steps

1. **Access Microsoft Tenant**
    - Navigate to [entra.microsoft.com](https://entra.microsoft.com) and PIM Owner over the HCP subscription.

2. **Login to Azure CLI and Retrieve Kubeconfig Files**
    - Authenticate to Azure CLI using device code login for the following subscription and tenant:

        ```sh
        az login --tenant "72f988bf-86f1-41af-91ab-2d7cd011db47" --use-device-code
        az account set --subscription  "5299e6b7-b23b-46c8-8277-dc1147807117"
        ```

    - Obtain kubeconfig for the mgmt and svc clusters:
        ```sh
        az aks get-credentials -n int-uksouth-svc-1 -g hcp-underlay-int-uksouth-svc -f svc.kubeconfig
        az aks get-credentials -n int-uksouth-mgmt-1 -g hcp-underlay-int-uksouth-mgmt-1 -f mgmt.kubeconfig
        ```

    - Use tokens for auth (short-lived, can expire)
        ```sh
        TOKEN=$(az account get-access-token --scope 6dae42f8-4368-4678-94ff-3960e28e3630/.default -o json | jq -r .accessToken)

        kubectl --kubeconfig svc.kubeconfig config unset users.clusterUser_hcp-underlay-int-uksouth-svc_int-uksouth-svc-1.exec
        kubectl --kubeconfig svc.kubeconfig config set-credentials clusterUser_hcp-underlay-int-uksouth-svc_int-uksouth-svc-1 --token $TOKEN

        kubectl --kubeconfig mgmt.kubeconfig config unset users.clusterUser_hcp-underlay-int-uksouth-mgmt-1_int-uksouth-mgmt-1
        kubectl --kubeconfig mgmt.kubeconfig config set-credentials clusterUser_hcp-underlay-int-uksouth-mgmt-1_int-uksouth-mgmt-1 --token $TOKEN
        ```

3. **Login to Red Hat Tenant**
    - Authenticate to the Red Hat tenant where the cluster is located using Azure CLI.

4. **Run the script**
    - `./debug.sh -s ./svc.kubeconfig -m ./mgmt.kubeconfig -c $CLUSTER_ID -r $RESOURCE_ID`