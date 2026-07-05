using '../templates/rhcos-publisher-lookup.bicep'

param msiName = '{{ .rhcosPublisher.managedIdentityName }}'
param imagePullerMsiName = 'image-puller'
param deployRhcosPublisher = {{ .rhcosPublisher.enabled }}
