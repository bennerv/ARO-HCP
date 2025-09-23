#!/bin/bash

usage() {
    echo "Usage: $0 -r <cluster_resource_id> -c <cluster_id> -s <service_cluster_kubeconfig> -m <mgmt_cluster_kubeconfig>"
    exit 1
}

RH_INT_SUBSCRIPTION="64f0619f-ebc2-4156-9d91-c4c781de7e54"
RH_INT_TENANT="64dc69e4-d083-49fc-9569-ebece1dd1408"
DP_KUBECONFIG="./dp.kubeconfig"

while getopts "r:c:s:m:" opt; do
    case $opt in
        r) CLUSTER_RESOURCE_ID="$OPTARG" ;;
        c) CLUSTER_ID="$OPTARG" ;;
        s) SERVICE_CLUSTER_KUBECONFIG="$OPTARG" ;;
        m) MGMT_CLUSTER_KUBECONFIG="$OPTARG" ;;
        *) usage ;;
    esac
done

IFS='/' read -ra RESOURCE_ID_PARTS <<< "$CLUSTER_RESOURCE_ID"
CLUSTER_NAME=$(echo "${RESOURCE_ID_PARTS[-1]}" | tr '[:upper:]' '[:lower:]')
CLUSTER_NAMESPACE="ocm-arohcpint-${CLUSTER_ID}-${CLUSTER_NAME}"

validate_kubeconfigs() {
    kubectl --kubeconfig="$SERVICE_CLUSTER_KUBECONFIG" get nodes >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: SERVICE_CLUSTER_KUBECONFIG is invalid or cannot access the cluster."
        exit 1
    fi

    kubectl --kubeconfig="$MGMT_CLUSTER_KUBECONFIG" get nodes >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: MGMT_CLUSTER_KUBECONFIG is invalid or cannot access the cluster."
        exit 1
    fi

    echo "Both kubeconfigs are valid and can access their respective clusters."
}

validate_resource_id() {
    az resource show --ids "$CLUSTER_RESOURCE_ID" --resource-type "Microsoft.RedHatOpenShift/hcpopenshiftcluster" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: Resource ID '$CLUSTER_RESOURCE_ID' does not exist or is not of type Microsoft.RedHatOpenShift/hcpopenshiftcluster."
        exit 1
    fi
    echo "Resource ID exists and is of correct type."
}


ensure_azure_auth() {
    CURRENT_TENANT=$(az account show --query tenantId -o tsv 2>/dev/null)
    CURRENT_SUBSCRIPTION=$(az account show --query id -o tsv 2>/dev/null)

    if [ "$CURRENT_TENANT" != "$RH_INT_TENANT" ] || [ "$CURRENT_SUBSCRIPTION" != "$RH_INT_SUBSCRIPTION" ]; then
        echo "Authenticating to Azure tenant $RH_INT_TENANT and subscription $RH_INT_SUBSCRIPTION..."
        az account set --subscription "$RH_INT_SUBSCRIPTION"
        az login --tenant "$RH_INT_TENANT" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            echo "Error: Failed to authenticate to Azure tenant $RH_INT_TENANT and subscription $RH_INT_SUBSCRIPTION."
            exit 1
        fi
    fi

    echo "You are correctly authenticated to Azure tenant $RH_INT_TENANT and subscription $RH_INT_SUBSCRIPTION."
}

request_dataplane_cert() {
    echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Requesting certificate"
    LOCATION_RESULT=$(az rest --method POST --uri "${CLUSTER_RESOURCE_ID}/requestAdminCredential?api-version=2024-06-10-preview" --verbose 2>&1 | grep "'Location': 'https://management.azure.com/" | sed "s/^.*'Location': '\(https:\/\/management\.azure\.com\/.*\)'.*$/\1/")

    while true; do
        CERT_RESULT=$(az rest --method GET --uri "$LOCATION_RESULT" --query "kubeconfig" -o tsv 2>/dev/null)
        if [ -z "$CERT_RESULT" ]; then
            continue
        fi
        echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Certificate obtained"
        az rest --method GET --uri "$LOCATION_RESULT" | jq -r '.kubeconfig' > "$DP_KUBECONFIG"
        break
    done
}

revoke_certificate() {
    echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Revoking certificate starting"
    az resource invoke-action --ids "$CLUSTER_RESOURCE_ID" --action revokecredentials --api-version 2024-06-10-preview
    echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') AZ call returned after revoking cert"
}

wait_for_dp_cert_expiry() {
    echo "Waiting for dataplane certificate to fail kubernetes calls..."
    while true; do
        kubectl --kubeconfig "$DP_KUBECONFIG" get nodes >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Dataplane certificate has expired or is no longer valid"
            break
        fi
        echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Dataplane certificate is still valid"
        sleep 5
    done
}

wait_for_dp_cert_validity() {
    echo "Waiting for dataplane certificate to be valid..."
    while true; do
        kubectl --kubeconfig "$DP_KUBECONFIG" get nodes >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Dataplane certificate is now valid"
            break
        fi
        echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Dataplane certificate is not yet valid"
        sleep 5
    done
}

cleanup() {
    rm -f "$DP_KUBECONFIG"
}

collect_logs() {
    echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Collecting logs..."
    local iteration=$1
    local start_time=$2

    # Kube-apiserver
    for pod in $(kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} get pods -l app=kube-apiserver -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} logs $pod -c kube-apiserver --since-time "${start_time}" > logs/$i-kube-apiserver-$pod.log
        pids+=($!)
    done

    # Maestro Agent
    for pod in $(kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n maestro get pods -l app=maestro-agent -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n maestro logs $pod -c maestro-agent --since-time "${start_time}"| grep ${CLUSTER_ID} > logs/$i-maestro-agent-$pod.log
    done

    # Backend
    kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n aro-hcp logs -l app=aro-hcp-backend -c aro-hcp-backend --since-time "${start_time}"> logs/$i-aro-hcp-backend.log

    # Frontend
    for pod in $(kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n aro-hcp get pods -l app=aro-hcp-frontend -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n aro-hcp logs $pod -c aro-hcp-frontend --since-time "${start_time}" > logs/$i-aro-hcp-frontend-$pod.log
    done

    # Cluster Service
    for pod in $(kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n clusters-service get pods -l app=clusters-service -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n clusters-service logs $pod -c service --since-time "${start_time}" | grep ${CLUSTER_ID} > logs/$i-clusters-service-$pod.log
    done

    # Maestro
    for pod in $(kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n maestro get pods -l app=maestro -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${SERVICE_CLUSTER_KUBECONFIG} -n maestro logs $pod -c service --since-time "${start_time}" > logs/$i-maestro-$pod.log
    done

    # control-plane-pki-operator CertRevocation logs for all pods
    for pod in $(kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} get pods -l app=control-plane-pki-operator -o jsonpath='{.items[*].metadata.name}'); do
        kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} logs $pod --since-time "${start_time}" | grep "CertificateRevocation" > logs/$i-control-plane-pki-operator-$pod.log
    done

}

main() {
    if [ -z "$CLUSTER_RESOURCE_ID" ] || [ -z "$CLUSTER_NAMESPACE" ] || [ -z "$SERVICE_CLUSTER_KUBECONFIG" ] || [ -z "$MGMT_CLUSTER_KUBECONFIG" ]; then
        echo "CLUSTER_RESOURCE_ID: $CLUSTER_RESOURCE_ID"
        echo "CLUSTER_NAMESPACE: $CLUSTER_NAMESPACE"
        echo "SERVICE_CLUSTER_KUBECONFIG: $SERVICE_CLUSTER_KUBECONFIG"
        echo "MGMT_CLUSTER_KUBECONFIG: $MGMT_CLUSTER_KUBECONFIG"
        usage
    fi

    validate_kubeconfigs
    validate_resource_id
    ensure_azure_auth

    mkdir -p logs

    for i in {1..5}; do
        echo "Iteration $i"
        request_dataplane_cert
        pids=()
        mkdir -p logs/

        start_time=$(date -u +'%Y-%m-%dT%H:%M:%S%:z')
        # Monitor PreviousCertificatesRevoked status in background
        (
            STATUS=""
            while true; do
                NEW_STATUS=$(kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} get crrs -o json | jq -r '.items[] | select(.status != null) | .status | select(.conditions != null) | .conditions[] | [.type, .status]')
                if [ "$NEW_STATUS" != "$STATUS" ]; then
                    STATUS="$NEW_STATUS"
                    echo "$(date -u +'%Y-%m-%dT%H:%M:%S%:z') Conditions status changed to $NEW_STATUS" >> logs/$i-crr-status.log
                fi
            done
        ) &
        pids+=($!)
        kubectl --kubeconfig ${MGMT_CLUSTER_KUBECONFIG} -n ${CLUSTER_NAMESPACE} get crrs -o yaml -w > logs/$i-crr-watch.log &
        pids+=($!)

        revoke_certificate
        request_dataplane_cert
        wait_for_dp_cert_validity
        collect_logs $i ${start_time}

        for pid in "${pids[@]}"; do
            kill "$pid"
        done
        cleanup

        echo "Sleeping for 60 seconds before next iteration..."
        sleep 60
    done
    echo "All iterations completed."
}

main
