using '../templates/rhcos-publisher-storage.bicep'

param deployRhcosPublisher = {{ .rhcosPublisher.enabled }}
param storageAccountName = '{{ .rhcosPublisher.storageAccount.name }}'
param rhcosPublisherMIName = '{{ .rhcosPublisher.managedIdentityName }}'
