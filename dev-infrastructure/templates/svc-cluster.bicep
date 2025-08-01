import {
  csvToArray
  determineZoneRedundancy
  determineZoneRedundancyForRegion
  getLocationAvailabilityZonesCSV
} from '../modules/common.bicep'
import * as res from '../modules/resource.bicep'

@description('Azure Region Location')
param location string = resourceGroup().location

@description('Availability Zones to use for the infrastructure, as a CSV string. Defaults to all the zones of the location')
param locationAvailabilityZones string = getLocationAvailabilityZonesCSV(location)
var locationAvailabilityZoneList = csvToArray(locationAvailabilityZones)

@description('AKS cluster name')
param aksClusterName string

@description('Minimum node count for system agent pool')
param systemAgentMinCount int

@description('Maximum node count for system agent pool')
param systemAgentMaxCount int

@description('VM instance type for the system nodes')
param systemAgentVMSize string

@description('Disk size for the AKS system nodes')
param aksSystemOsDiskSizeGB int

@description('Disk size for the AKS user nodes')
param aksUserOsDiskSizeGB int

@description('Network dataplane plugin for the AKS cluster')
param aksNetworkDataplane string

@description('Network policy plugin for the AKS cluster')
param aksNetworkPolicy string

@description('Min replicas for the worker nodes')
param userAgentMinCount int

@description('Max replicas for the worker nodes')
param userAgentMaxCount int

@description('VM instance type for the worker nodes')
param userAgentVMSize string

@description('Number of availability zones to use for the AKS clusters user agent pool')
param userAgentPoolAZCount int

@description('Min replicas for the infra worker nodes')
param infraAgentMinCount int

@description('Max replicas for the infra worker nodes')
param infraAgentMaxCount int

@description('VM instance type for the infra worker nodes')
param infraAgentVMSize string

@description('Number of availability zones to use for the AKS clusters infra user agent pool')
param infraAgentPoolAZCount int

@description('Disk size for the AKS infra nodes')
param aksInfraOsDiskSizeGB int

@description('The resource ID of the OCP ACR')
param ocpAcrResourceId string

@description('The resource ID of the SVC ACR')
param svcAcrResourceId string

@description('Name of the resource group for the AKS nodes')
param aksNodeResourceGroupName string = '${resourceGroup().name}-aks1'

@description('VNET address prefix')
param vnetAddressPrefix string

@description('Subnet address prefix')
param subnetPrefix string

@description('Specifies the address prefix of the subnet hosting the pods of the AKS cluster.')
param podSubnetPrefix string

@description('Kubernetes version to use with AKS')
param kubernetesVersion string

@description('Istio control plane versions to use with AKS. CSV format')
param istioVersions string

@description('The name of the keyvault for AKS.')
@maxLength(24)
param aksKeyVaultName string

@description('The tag key for the AKS keyvault')
param aksKeyVaultTagName string

@description('The tag value for the AKS keyvault')
param aksKeyVaultTagValue string

@description('Manage soft delete setting for AKS etcd key-value store')
param aksEtcdKVEnableSoftDelete bool = true

@description('IPTags to be set on the cluster outbound IP address in the format of ipTagType:tag,ipTagType:tag')
param aksClusterOutboundIPAddressIPTags string = ''

@description('The name of the Istio Ingress Gateway IP address resource')
param istioIngressGatewayIPAddressName string = ''

@description('IPTags to be set on the Istio Ingress Gateway IP address in the format of ipTagType:tag,ipTagType:tag')
param istioIngressGatewayIPAddressIPTags string = ''

// TODO: When the work around workload identity for the RP is finalized, change this to true
@description('disableLocalAuth for the ARO HCP RP CosmosDB')
param disableLocalAuth bool

@description('Deploy ARO HCP RP Azure Cosmos DB if true')
param deployFrontendCosmos bool

@description('The name of the Cosmos DB for the RP')
param rpCosmosDbName string

@description('If true, make the Cosmos DB instance private')
param rpCosmosDbPrivate bool

@description('The zone redundant mode of Cosmos DB instance')
param rpCosmosZoneRedundantMode string

@description('The resourcegroup for regional infrastructure')
param regionalResourceGroup string

@description('The domain to use to use for the maestro certificate. Relevant only for environments where OneCert can be used.')
param maestroCertDomain string

@description('The issuer of the maestro certificate.')
param maestroCertIssuer string

@description('The name of the eventgrid namespace for Maestro.')
param maestroEventGridNamespacesName string

@description('Deploy CS Postgres if true')
param csPostgresDeploy bool

@description('The zone redundant mode of the Maestro Postgres Database')
param csPostgresZoneRedundantMode string

@description('The name of the Postgres server for CS')
@maxLength(60)
param csPostgresServerName string

@description('The name of the CS Postgres database')
param csPostgresDatabaseName string

@description('The minimum TLS version for the Postgres server for CS')
param csPostgresServerMinTLSVersion string

@description('The version of the Postgres server for CS')
param csPostgresServerVersion string

@description('The size of the Postgres server for CS')
param csPostgresServerStorageSizeGB int

@description('If true, make the CS Postgres instance private')
param clusterServicePostgresPrivate bool = true

@description('Deploy ARO HCP Maestro Postgres if true')
param deployMaestroPostgres bool = true

@description('The zone redundant mode of the Maestro Postgres Database')
param maestroPostgresZoneRedundantMode string

@description('If true, make the Maestro Postgres instance private')
param maestroPostgresPrivate bool = true

@description('The name of the Postgres server for Maestro')
@maxLength(60)
param maestroPostgresServerName string

@description('The version of the Postgres server for Maestro')
param maestroPostgresServerVersion string

@description('The minimum TLS version for the Postgres server for Maestro')
param maestroPostgresServerMinTLSVersion string

@description('The size of the Postgres server for Maestro')
param maestroPostgresServerStorageSizeGB int

@description('The name of the Maestro Postgres database')
param maestroPostgresDatabaseName string

@description('The name of Maestro Server MQTT client')
param maestroServerMqttClientName string

@description('The name of the maestro managed identity')
param maestroMIName string

@description('The namespace of the maestro managed identity')
param maestroNamespace string

@description('The service account name of the maestro managed identity')
param maestroServiceAccountName string

@description('The name of the service keyvault')
param serviceKeyVaultName string

@description('The name of the resourcegroup for the service keyvault')
param serviceKeyVaultResourceGroup string = resourceGroup().name

@description('OIDC Storage Account name')
param oidcStorageAccountName string

@description('Whether the OIDC storage account is public or private. If private, it can only be accessed via Azure Front Door')
param oidcStorageAccountPublic bool

@description('The zone redundant mode of the OIDC storage account')
param oidcZoneRedundantMode string

@description('The name of the global Azure Front Door profile fronting the OIDC storage account')
param azureFrontDoorResourceId string

@description('The name of the global Azure Front Door parent DNS zone')
param azureFrontDoorParentDnsZoneName string

@description('The regional subdomain for the Azure Front Door')
param azureFrontDoorRegionalSubdomain string

@description('The name of the Azure Front Door global Key Vault')
param azureFrontDoorKeyVaultName string

@description('The tag key for the Azure Front Door Key Vault')
param azureFrontDoorKeyTagKey string

@description('The tag value for the Azure Front Door Key Vault')
param azureFrontDoorKeyTagValue string

@description('Whether to use managed certificates for the Azure Front Door')
param azureFrontDoorUseManagedCertificates bool

@description('MSI that will be used to run the deploymentScript')
param globalMSIId string

@description('The parent SVC DNS zone name')
param svcDNSZoneName string

@description('The regional DNS zone to hold ARO HCP customer cluster DNS records')
param regionalCXDNSZoneName string

@description('This is a regional DNS zone name to hold records for ARO HCP service components, e.g. the RP')
param regionalSvcDNSZoneName string

@description('Frontend Ingress Certificate Name')
param frontendIngressCertName string

@description('Frontend Ingress Certificate Issuer')
param frontendIngressCertIssuer string

@description('The name of the FPA certificate in the SVC keyvault')
param fpaCertificateName string

@description('The issuer of the FPA certificate')
param fpaCertificateIssuer string

@description('Whether to create the FPA certificate in the SVC keyvault')
param manageFpaCertificate bool

@description('Whether to create the MSI refresher certificate in the SVC keyvault')
param manageMsiRefresherCertificate bool

@description('The name of the MSI refresher certificate in the SVC keyvault')
param msiRefresherCertificateName string

@description('The issuer of the MSI refresher certificate')
param msiRefresherCertificateIssuer string

@description('The service tag for Geneva Actions')
param genevaActionsServiceTag string

@description('The Azure Resource ID of the Azure Monitor Workspace (stores prometheus metrics)')
param azureMonitoringWorkspaceId string

@description('The name of the CS managed identity')
param csMIName string

@description('The namespace of the CS managed identity')
param csNamespace string

@description('The service account name of the CS managed identity')
param csServiceAccountName string

@description('The name of the MSI refresher managed identity')
param msiRefresherMIName string

@description('The namespace of the MSI refresher managed identity')
param msiRefresherNamespace string

@description('The service account name of the MSI refresher managed identity')
param msiRefresherServiceAccountName string

// logs
@description('The namespace of the logs')
param logsNamespace string

@description('The managed identity name of the logs')
param logsMSI string

@description('The service account name of the logs managed identity')
param logsServiceAccount string

@description('Tha name of the SVC NSP')
param svcNSPName string

@description('Access mode for this NSP')
param svcNSPAccessMode string

@description('Access mode for this NSP')
param serviceKeyVaultAsignNSP bool = true

@description('Domain used for creation of geneva auth certificates')
param genevaCertificateDomain string

@description('Issuer of certificate for Geneva Authentication')
param genevaCertificateIssuer string = 'Self'

@description('Name of certificate in Keyvault and hostname used in SAN')
param genevaRpLogsName string

@description('Should geneva certificates be managed')
param genevaManageCertificates bool

// Log Analytics Workspace ID will be passed from region pipeline if enabled in config
param logAnalyticsWorkspaceId string = ''

resource serviceKeyVault 'Microsoft.KeyVault/vaults@2024-04-01-preview' existing = {
  name: serviceKeyVaultName
  scope: resourceGroup(serviceKeyVaultResourceGroup)
}

resource svcClusterNSG 'Microsoft.Network/networkSecurityGroups@2023-11-01' = {
  location: location
  name: 'svc-cluster-node-nsg'
  properties: {
    securityRules: [
      {
        name: 'rp-in-arm'
        properties: {
          access: 'Allow'
          destinationAddressPrefix: '*'
          destinationPortRange: '443'
          direction: 'Inbound'
          priority: 120
          protocol: 'Tcp'
          sourceAddressPrefix: 'AzureResourceManager'
          sourcePortRange: '*'
        }
      }
      {
        name: 'admin-in-geneva'
        properties: {
          access: 'Allow'
          destinationAddressPrefix: '*'
          destinationPortRange: '443'
          direction: 'Inbound'
          priority: 130
          protocol: 'Tcp'
          sourceAddressPrefix: genevaActionsServiceTag
          sourcePortRange: '*'
        }
      }
    ]
  }
}

module svcCluster '../modules/aks-cluster-base.bicep' = {
  name: 'cluster'
  scope: resourceGroup()
  params: {
    location: location
    locationAvailabilityZones: locationAvailabilityZoneList
    ipResourceGroup: regionalResourceGroup
    aksClusterName: aksClusterName
    aksNodeResourceGroupName: aksNodeResourceGroupName
    aksEtcdKVEnableSoftDelete: aksEtcdKVEnableSoftDelete
    aksClusterOutboundIPAddressIPTags: aksClusterOutboundIPAddressIPTags
    kubernetesVersion: kubernetesVersion
    deployIstio: true
    istioVersions: split(istioVersions, ',')
    istioIngressGatewayIPAddressName: istioIngressGatewayIPAddressName
    istioIngressGatewayIPAddressIPTags: istioIngressGatewayIPAddressIPTags
    vnetAddressPrefix: vnetAddressPrefix
    nodeSubnetNSGId: svcClusterNSG.id
    subnetPrefix: subnetPrefix
    podSubnetPrefix: podSubnetPrefix
    clusterType: 'svc-cluster'
    systemOsDiskSizeGB: aksSystemOsDiskSizeGB
    userOsDiskSizeGB: aksUserOsDiskSizeGB
    userAgentMinCount: userAgentMinCount
    userAgentMaxCount: userAgentMaxCount
    userAgentVMSize: userAgentVMSize
    userAgentPoolAZCount: userAgentPoolAZCount
    infraAgentMinCount: infraAgentMinCount
    infraAgentMaxCount: infraAgentMaxCount
    infraAgentVMSize: infraAgentVMSize
    infraAgentPoolAZCount: infraAgentPoolAZCount
    infraOsDiskSizeGB: aksInfraOsDiskSizeGB
    systemAgentMinCount: systemAgentMinCount
    systemAgentMaxCount: systemAgentMaxCount
    systemAgentVMSize: systemAgentVMSize
    networkDataplane: aksNetworkDataplane
    networkPolicy: aksNetworkPolicy
    workloadIdentities: items({
      frontend_wi: {
        uamiName: 'frontend'
        namespace: 'aro-hcp'
        serviceAccountName: 'frontend'
      }
      backend_wi: {
        uamiName: 'backend'
        namespace: 'aro-hcp'
        serviceAccountName: 'backend'
      }
      backplane_wi: {
        uamiName: 'backplane-api'
        namespace: 'aro-hcp'
        serviceAccountName: 'backplane-api'
      }
      maestro_wi: {
        uamiName: maestroMIName
        namespace: maestroNamespace
        serviceAccountName: maestroServiceAccountName
      }
      cs_wi: {
        uamiName: csMIName
        namespace: csNamespace
        serviceAccountName: csServiceAccountName
      }
      image_sync_wi: {
        uamiName: 'image-sync'
        namespace: 'image-sync'
        serviceAccountName: 'image-sync'
      }
      logs_wi: {
        uamiName: logsMSI
        namespace: logsNamespace
        serviceAccountName: logsServiceAccount
      }
      prom_wi: {
        uamiName: 'prometheus'
        namespace: 'prometheus'
        serviceAccountName: 'prometheus'
      }
      msi_refresher_wi: {
        uamiName: msiRefresherMIName
        namespace: msiRefresherNamespace
        serviceAccountName: msiRefresherServiceAccountName
      }
    })
    aksKeyVaultName: aksKeyVaultName
    aksKeyVaultTagName: aksKeyVaultTagName
    aksKeyVaultTagValue: aksKeyVaultTagValue
    logAnalyticsWorkspaceId: logAnalyticsWorkspaceId
    pullAcrResourceIds: [svcAcrResourceId]
    deploymentMsiId: globalMSIId
    enableSwiftV2Vnet: false
    enableSwiftV2Nodepools: false
  }
}

output aksClusterName string = svcCluster.outputs.aksClusterName

//
// L O G S
//

// NOTE: This is only enabled for non-prod environments
module logsCollection '../modules/logs/collection.bicep' = if (logAnalyticsWorkspaceId != '') {
  name: 'logs-collection'
  params: {
    aksClusterName: svcCluster.outputs.aksClusterName
    logAnalyticsWorkspaceId: logAnalyticsWorkspaceId
  }
}

//
// M E T R I C S
//

module dataCollection '../modules/metrics/datacollection.bicep' = {
  name: 'metrics-infra'
  params: {
    azureMonitorWorkspaceLocation: location
    azureMonitoringWorkspaceId: azureMonitoringWorkspaceId
    aksClusterName: aksClusterName
    prometheusPrincipalId: filter(svcCluster.outputs.userAssignedIdentities, id => id.uamiName == 'prometheus')[0].uamiPrincipalID
  }
}

var frontendMI = filter(svcCluster.outputs.userAssignedIdentities, id => id.uamiName == 'frontend')[0]
var backendMI = filter(svcCluster.outputs.userAssignedIdentities, id => id.uamiName == 'backend')[0]

module rpCosmosDb '../modules/rp-cosmos.bicep' = if (deployFrontendCosmos) {
  name: 'rp_cosmos_db'
  scope: resourceGroup(regionalResourceGroup)
  params: {
    name: rpCosmosDbName
    location: location
    zoneRedundant: determineZoneRedundancy(locationAvailabilityZoneList, rpCosmosZoneRedundantMode)
    disableLocalAuth: disableLocalAuth
    userAssignedMIs: [frontendMI, backendMI]
    private: rpCosmosDbPrivate
  }
}

module rpCosmosdbPrivateEndpoint '../modules/private-endpoint.bicep' = {
  name: '${deployment().name}-rp-pe'
  params: {
    location: location
    subnetIds: [svcCluster.outputs.aksNodeSubnetId]
    vnetId: svcCluster.outputs.aksVnetId
    privateLinkServiceId: rpCosmosDb.outputs.cosmosDBAccountId
    serviceType: 'cosmosdb'
    groupId: 'Sql'
  }
}

output cosmosDBName string = deployFrontendCosmos ? rpCosmosDb.outputs.cosmosDBName : ''
output frontend_mi_client_id string = frontendMI.uamiClientID

//
//   M A E S T R O
//

var effectiveMaestroCertDomain = !empty(maestroCertDomain) ? maestroCertDomain : 'maestro.${regionalSvcDNSZoneName}'

module maestroServer '../modules/maestro/maestro-server.bicep' = {
  name: 'maestro-server'
  params: {
    regionalResourceGroup: regionalResourceGroup
    maestroInfraResourceGroup: regionalResourceGroup
    maestroEventGridNamespaceName: maestroEventGridNamespacesName
    mqttClientName: maestroServerMqttClientName
    certKeyVaultName: serviceKeyVaultName
    certKeyVaultResourceGroup: serviceKeyVaultResourceGroup
    keyVaultOfficerManagedIdentityName: globalMSIId
    maestroCertificateDomain: effectiveMaestroCertDomain
    maestroCertificateIssuer: maestroCertIssuer
    deployPostgres: deployMaestroPostgres
    postgresServerName: maestroPostgresServerName
    postgresServerVersion: maestroPostgresServerVersion
    postgresServerMinTLSVersion: maestroPostgresServerMinTLSVersion
    postgresServerStorageSizeGB: maestroPostgresServerStorageSizeGB
    postgresZoneRedundantMode: determineZoneRedundancyForRegion(location, maestroPostgresZoneRedundantMode)
      ? 'ZoneRedundant'
      : 'SameZone'
    privateEndpointSubnetId: svcCluster.outputs.aksNodeSubnetId
    privateEndpointVnetId: svcCluster.outputs.aksVnetId
    privateEndpointResourceGroup: resourceGroup().name
    maestroDatabaseName: maestroPostgresDatabaseName
    postgresServerPrivate: maestroPostgresPrivate
    postgresAdministrationManagedIdentityId: globalMSIId
    maestroServerManagedIdentityPrincipalId: filter(
      svcCluster.outputs.userAssignedIdentities,
      id => id.uamiName == maestroMIName
    )[0].uamiPrincipalID
    maestroServerManagedIdentityName: maestroMIName
    logAnalyticsWorkspaceId: logAnalyticsWorkspaceId
  }
  dependsOn: [
    serviceKeyVault
  ]
}

//
//   K E Y V A U L T S
//

var logsManagedIdentityPrincipalId = filter(svcCluster.outputs.userAssignedIdentities, id => id.uamiName == logsMSI)[0].uamiPrincipalID

module logsServiceKeyVaultAccess '../modules/keyvault/keyvault-secret-access.bicep' = {
  name: guid(serviceKeyVaultName, logsMSI, 'certuser')
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    roleName: 'Key Vault Certificate User'
    managedIdentityPrincipalId: logsManagedIdentityPrincipalId
  }
}

//
//   C L U S T E R   S E R V I C E
//

var csManagedIdentityPrincipalId = filter(svcCluster.outputs.userAssignedIdentities, id => id.uamiName == csMIName)[0].uamiPrincipalID

module cs '../modules/cluster-service.bicep' = {
  name: 'cluster-service'
  params: {
    postgresServerName: csPostgresServerName
    postgresServerMinTLSVersion: csPostgresServerMinTLSVersion
    postgresServerVersion: csPostgresServerVersion
    postgresServerStorageSizeGB: csPostgresServerStorageSizeGB
    csDatabaseName: csPostgresDatabaseName
    privateEndpointSubnetId: svcCluster.outputs.aksNodeSubnetId
    privateEndpointVnetId: svcCluster.outputs.aksVnetId
    privateEndpointResourceGroup: resourceGroup().name
    deployPostgres: csPostgresDeploy
    postgresZoneRedundantMode: determineZoneRedundancyForRegion(location, csPostgresZoneRedundantMode)
      ? 'ZoneRedundant'
      : 'SameZone'
    postgresServerPrivate: clusterServicePostgresPrivate
    clusterServiceManagedIdentityPrincipalId: csManagedIdentityPrincipalId
    clusterServiceManagedIdentityName: csMIName
    serviceKeyVaultName: serviceKeyVault.name
    serviceKeyVaultResourceGroup: serviceKeyVaultResourceGroup
    regionalCXDNSZoneName: regionalCXDNSZoneName
    regionalResourceGroup: regionalResourceGroup
    ocpAcrResourceId: ocpAcrResourceId
    postgresAdministrationManagedIdentityId: globalMSIId
  }
  dependsOn: [
    maestroServer
  ]
}

//
//   O I D C
//

var frontDoorRef = res.frontdoorProfileRefFromId(azureFrontDoorResourceId)
module oidc '../modules/oidc/region/main.bicep' = {
  name: 'oidc-storage'
  scope: resourceGroup(regionalResourceGroup)
  params: {
    gblRgName: frontDoorRef.resourceGroup.name
    gblSubscription: frontDoorRef.resourceGroup.subscriptionId
    location: location
    zoneName: azureFrontDoorParentDnsZoneName
    frontDoorProfileName: frontDoorRef.name
    storageAccountName: oidcStorageAccountName
    customDomainName: azureFrontDoorRegionalSubdomain
    routeName: azureFrontDoorRegionalSubdomain
    originGroupName: azureFrontDoorRegionalSubdomain
    originName: azureFrontDoorRegionalSubdomain
    privateLinkLocation: location
    storageAccountAccessPrincipalId: csManagedIdentityPrincipalId
    skuName: determineZoneRedundancy(locationAvailabilityZoneList, oidcZoneRedundantMode)
      ? 'Standard_ZRS'
      : 'Standard_LRS'
    keyVaultName: azureFrontDoorKeyVaultName
    useManagedCertificates: azureFrontDoorUseManagedCertificates
    globalMSIId: globalMSIId
    deploymentScriptLocation: location
    storageAccountBlobPublicAccess: oidcStorageAccountPublic
  }
}

//
//  E V E N T   G R I D   P R I V A T E   E N D P O I N T   C O N N E C T I O N
//

resource eventGridNamespace 'Microsoft.EventGrid/namespaces@2024-06-01-preview' existing = {
  name: maestroEventGridNamespacesName
  scope: resourceGroup(regionalResourceGroup)
}

// todo manage only if maestro.eventgrid is not set to private
module eventGrindPrivateEndpoint '../modules/private-endpoint.bicep' = {
  name: 'eventGridPrivateEndpoint'
  params: {
    location: location
    subnetIds: [svcCluster.outputs.aksNodeSubnetId]
    privateLinkServiceId: eventGridNamespace.id
    serviceType: 'eventgrid'
    groupId: 'topicspace'
    vnetId: svcCluster.outputs.aksVnetId
  }
}

//
//   M S I   C R E D E N T I A L S   R E F R E S H E R
//

var msiRefresherCertificateSNI = '${msiRefresherCertificateName}.${svcDNSZoneName}'

module msiRefresherCertificate '../modules/keyvault/key-vault-cert.bicep' = if (manageMsiRefresherCertificate) {
  name: 'msi-refresher-certificate-${uniqueString(resourceGroup().name)}'
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    subjectName: 'CN=${msiRefresherCertificateSNI}'
    certName: msiRefresherCertificateName
    keyVaultManagedIdentityId: globalMSIId
    dnsNames: [
      msiRefresherCertificateSNI
    ]
    issuerName: msiRefresherCertificateIssuer
  }
}

//
//   F R O N T E N D
//

var frontendDnsName = 'rp'
var frontendDnsFQDN = '${frontendDnsName}.${regionalSvcDNSZoneName}'

module frontendIngressCert '../modules/keyvault/key-vault-cert.bicep' = {
  name: 'frontend-cert-${uniqueString(resourceGroup().name)}'
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    subjectName: 'CN=${frontendDnsFQDN}'
    certName: frontendIngressCertName
    keyVaultManagedIdentityId: globalMSIId
    dnsNames: [
      frontendDnsFQDN
    ]
    issuerName: frontendIngressCertIssuer
  }
}

module frontendIngressCertCSIAccess '../modules/keyvault/keyvault-secret-access.bicep' = {
  name: 'aksClusterKeyVaultSecretsProviderMI-${frontendIngressCertName}'
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    roleName: 'Key Vault Secrets User'
    managedIdentityPrincipalId: svcCluster.outputs.aksClusterKeyVaultSecretsProviderPrincipalId
    secretName: frontendIngressCertName
  }
}

module frontendDNS '../modules/dns/a-record.bicep' = {
  name: 'frontend-dns'
  scope: resourceGroup(regionalResourceGroup)
  params: {
    zoneName: regionalSvcDNSZoneName
    recordName: frontendDnsName
    ipAddress: svcCluster.outputs.istioIngressGatewayIPAddress
    ttl: 300
  }
}

//
//   F P A   C E R T I F I C A T E
//

var fpaCertificateSNI = '${fpaCertificateName}.${svcDNSZoneName}'

module fpaCertificate '../modules/keyvault/key-vault-cert.bicep' = if (manageFpaCertificate) {
  name: 'fpa-certificate-${uniqueString(resourceGroup().name)}'
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    subjectName: 'CN=${fpaCertificateSNI}'
    certName: fpaCertificateName
    keyVaultManagedIdentityId: globalMSIId
    dnsNames: [
      fpaCertificateSNI
    ]
    issuerName: fpaCertificateIssuer
  }
}

//
//   G E N E V A   C E R T I F I C A T E
//

module genevaRPCertificate '../modules/keyvault/key-vault-cert-with-access.bicep' = if (genevaManageCertificates) {
  name: 'geneva-rp-certificate-${uniqueString(resourceGroup().name)}'
  scope: resourceGroup(serviceKeyVaultResourceGroup)
  params: {
    keyVaultName: serviceKeyVaultName
    kvCertOfficerManagedIdentityResourceId: globalMSIId
    certDomain: genevaCertificateDomain
    certificateIssuer: genevaCertificateIssuer
    hostName: genevaRpLogsName
    keyVaultCertificateName: genevaRpLogsName
    certificateAccessManagedIdentityPrincipalId: svcCluster.outputs.aksClusterKeyVaultSecretsProviderPrincipalId
  }
}

//
//   N E T W O R K    S E C U R I T Y    P E R I M E T E R
//

module svcNSP '../modules/network/nsp.bicep' = {
  name: 'nsp-${uniqueString(resourceGroup().name)}'
  params: {
    nspName: svcNSPName
    location: location
  }
}

module svcClusterNSPProfile '../modules/network/nsp-profile.bicep' = {
  name: 'profile-${uniqueString(resourceGroup().name)}'
  params: {
    accessMode: svcNSPAccessMode
    nspName: svcNSPName
    profileName: svcNSPName
    location: location
    associatedResources: [
      svcCluster.outputs.etcKeyVaultId
      rpCosmosDb.outputs.cosmosDBAccountId
    ]
    // TODO Add EV2 access here
    subscriptions: [
      subscription().id
    ]
  }
  dependsOn: [
    svcNSP
  ]
}

module svcKVNSPProfile '../modules/network/nsp-profile.bicep' = if (serviceKeyVaultAsignNSP) {
  name: 'profile-svc-kv-${uniqueString(resourceGroup().name)}'
  params: {
    accessMode: svcNSPAccessMode
    nspName: svcNSPName
    profileName: '${svcNSPName}-svc-kv'
    location: location
    associatedResources: [
      serviceKeyVault.id
    ]
    // TODO Add EV2 access here
    subscriptions: [
      subscription().id
    ]
  }
  dependsOn: [
    svcNSP
  ]
}
