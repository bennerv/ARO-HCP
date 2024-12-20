EVENTGRID_NAME ?= {{ .maestro.eventGrid.name }}
REGION_RG ?= {{ .regionRG }}
AKS_NAME ?= {{ .aksName }}
SVC_RG ?= {{ .svc.rg }}
IMAGE_BASE ?= {{ .maestro.imageBase }}
IMAGE_TAG ?= {{ .maestro.imageTag }}
USE_CONTAINERIZED_DB ?= {{ not .maestro.postgres.deploy }}
USE_DATABASE_SSL ?= {{ ternary "enable" "disable" .maestro.postgres.deploy }}
ISTIO_RESTRICT_INGRESS ?= {{ .maestro.restrictIstioIngress }}
KEYVAULT_NAME ?= {{ .serviceKeyVault.name }}
MQTT_CLIENT_NAME ?= {{ .maestro.serverMqttClientName }}
