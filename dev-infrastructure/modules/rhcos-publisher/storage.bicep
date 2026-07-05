// Storage account for staging RHCOS VHDs on their way to the ARO 1P
// marketplace. The rhcos-publisher controller uploads verified VHDs into the
// $web container; static website hosting (enabled by a Shell pipeline step,
// see dev-infrastructure/scripts/storage.sh) serves them publicly over HTTPS
// so the Partner Center Product Ingestion API can ingest them without SAS
// tokens. Shared key access is disabled: all writes go through Entra ID.

@description('The name of the Azure Storage account to create.')
@minLength(3)
@maxLength(24)
param storageAccountName string

@description('The location into which the Azure Storage resources should be deployed.')
param location string

module rhcosPublisherStorageAccount '../storage/storage.bicep' = {
  name: 'rhcosPublisherStorageAccount'
  params: {
    storageAccountName: storageAccountName
    location: location
    skuName: 'Standard_LRS'
    accessTier: 'Hot'
    allowBlobPublicAccess: true
    allowSharedKeyAccess: false
    publicNetworkAccess: 'Enabled'
    configureNetworkAcls: true
    networkAclsBypass: 'AzureServices'
    networkAclsDefaultAction: 'Allow'
    configureEncryption: true
  }
}

output storageAccountId string = rhcosPublisherStorageAccount.outputs.storageAccountId
output storageAccountName string = rhcosPublisherStorageAccount.outputs.storageAccountName
