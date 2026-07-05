@description('Storage account name for RHCOS VHD staging')
param storageAccountName string

@description('Principal ID of the rhcos-publisher managed identity')
param rhcosPublisherManagedIdentityPrincipalId string

// Storage Blob Data Contributor: Grants read, write, and delete access to blob containers and data
// https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#storage-blob-data-contributor
var storageBlobDataContributorRole = 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'

// Reader: Grants permission to read storage account properties, used to
// discover the blob/web endpoints at runtime
// https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#reader
var readerRole = 'acdd72a7-3385-48ef-bd42-f606fba81ae7'

resource rhcosPublisherStorageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' existing = {
  name: storageAccountName
}

resource rhcosPublisherStorageBlobDataContributorAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccountName, 'rhcos-publisher-blob-contributor', storageBlobDataContributorRole)
  scope: rhcosPublisherStorageAccount
  properties: {
    principalId: rhcosPublisherManagedIdentityPrincipalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', storageBlobDataContributorRole)
  }
}

resource rhcosPublisherReaderAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccountName, 'rhcos-publisher-reader', readerRole)
  scope: rhcosPublisherStorageAccount
  properties: {
    principalId: rhcosPublisherManagedIdentityPrincipalId
    principalType: 'ServicePrincipal'
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', readerRole)
  }
}
