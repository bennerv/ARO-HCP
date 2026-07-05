@description('The name of the rhcos-publisher managed identity')
param msiName string

@description('The name of the Image Puller MSI')
param imagePullerMsiName string

@description('Whether the rhcos-publisher is deployed in this environment')
param deployRhcosPublisher bool

//
//   I M A G E   P U L L E R   L O O K U P
//

resource imagePullerIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = {
  scope: resourceGroup()
  name: imagePullerMsiName
}

output imagePullerMsiClientId string = imagePullerIdentity.properties.clientId
output imagePullerMsiTenantId string = imagePullerIdentity.properties.tenantId

//
//   R H C O S   P U B L I S H E R   L O O K U P
//

resource managedIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = if (deployRhcosPublisher) {
  scope: resourceGroup()
  name: msiName
}

output tenantId string = tenant().tenantId
// The MSI only exists in environments where the publisher is enabled; the
// Helm chart renders no workloads in other environments, so a placeholder
// client ID is fine there.
output msiClientId string = deployRhcosPublisher ? managedIdentity.properties.clientId : 'disabled'
