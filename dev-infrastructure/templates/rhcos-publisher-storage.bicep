// Deploys the RHCOS VHD staging storage account and grants the
// rhcos-publisher managed identity access to it. Deployed conditionally: the
// rhcos-publisher only runs in environments that can reach the Partner Center
// marketplace account (INT).

@description('Whether to deploy the rhcos-publisher storage account')
param deployRhcosPublisher bool

@description('The name of the Azure Storage account to create.')
param storageAccountName string

@description('The location of the storage account.')
param location string = resourceGroup().location

@description('The name of the rhcos-publisher managed identity')
param rhcosPublisherMIName string

resource rhcosPublisherIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = if (deployRhcosPublisher) {
  scope: resourceGroup()
  name: rhcosPublisherMIName
}

module storage '../modules/rhcos-publisher/storage.bicep' = if (deployRhcosPublisher) {
  name: 'rhcos-publisher-storage'
  params: {
    storageAccountName: storageAccountName
    location: location
  }
}

module storageRbac '../modules/rhcos-publisher/storage-rbac.bicep' = if (deployRhcosPublisher) {
  name: 'rhcos-publisher-storage-rbac'
  params: {
    storageAccountName: storageAccountName
    rhcosPublisherManagedIdentityPrincipalId: rhcosPublisherIdentity.properties.principalId
  }
  dependsOn: [
    storage
  ]
}
