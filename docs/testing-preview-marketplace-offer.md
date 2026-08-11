# Testing a Preview Marketplace Offer

This document describes how to deploy and validate a preview (hidden) Azure Marketplace VM image, such as an RHCOS image published under the `azureopenshift` publisher.

## Background

Azure Marketplace offers can be published in a **preview** state before going live. Preview images are only visible to Azure subscriptions that have been added to the offer's **preview audience** in [Partner Center](https://partner.microsoft.com/dashboard/home). For full details, see:

- [Add a preview audience for a VM offer](https://learn.microsoft.com/en-us/partner-center/marketplace-offers/azure-vm-preview-audience)
- [How do I test a hidden preview image?](https://learn.microsoft.com/en-us/partner-center/marketplace-offers/azure-vm-faq#how-do-i-test-a-hidden-preview-image-)

## Prerequisites

1. Your Azure subscription must be in the offer's **preview audience** (configured in Partner Center). If it is not, the deployment will fail with a `NotFound` error.
2. The ARO marketplace image (`azureopenshift:aro4`) is a **CoreVM** offering, so no `plan` block is required in the ARM/Bicep template. However, the `-preview` suffix on the offer ID **is still required**.

## Deploying a preview image

### 1. Identify the image URN

The URN follows the format `publisher:offer:sku:version`. For example:

```
azureopenshift:aro4:aro_50-x64:10.2.20260423
```

### 2. Deploy via Bicep/ARM template

For preview images, append `-preview` to the offer ID in the `imageReference`. Direct CLI deployment (`az vm create`) does **not** work for hidden preview images — you must use an ARM template deployment.

Example `imageReference` for a preview image:

```bicep
storageProfile: {
  imageReference: {
    publisher: 'azureopenshift'
    offer: 'aro4-preview'       // append '-preview' for hidden preview images
    sku: 'aro_50-x64'
    version: '10.2.20260423'
  }
}
```

For a live (non-preview) image, use the offer ID without the suffix:

```bicep
storageProfile: {
  imageReference: {
    publisher: 'azureopenshift'
    offer: 'aro4'
    sku: 'aro_50-x64'
    version: '10.2.20260423'
  }
}
```

### 3. Deploy

A sample Bicep template that creates a self-contained VM with networking and boot diagnostics is available at `dev-infrastructure/templates/test-marketplace-vm.bicep` (or can be created from this example).

```bash
# Create a resource group
az group create --name aro-marketplace-test --location eastus

# Deploy the preview image
az deployment group create \
  --resource-group aro-marketplace-test \
  --template-file <path-to-bicep> \
  --parameters sshPublicKey="$(cat ~/.ssh/id_rsa.pub)" isPreview=true
```

### 4. Validate the VM booted correctly

Check the boot diagnostics to confirm the RHCOS image provisioned successfully:

```bash
az vm boot-diagnostics get-boot-log \
  --resource-group aro-marketplace-test \
  --name aro-rhcos-vm
```

You can also view the boot screenshot and serial console output in the Azure portal under **VM > Boot diagnostics**.

### 5. Clean up

```bash
az group delete --name aro-marketplace-test --yes --no-wait
```

## Key differences: preview vs. live images

| Aspect | Preview image | Live image |
|--------|--------------|------------|
| Offer ID in `imageReference` | `aro4-preview` | `aro4` |
| Subscription requirement | Must be in preview audience | Publicly available |
| Deployment method | ARM/Bicep template only | ARM/Bicep or `az vm create` |
| Plan block (CoreVM offers) | Not required | Not required |
| Marketplace terms acceptance | Not required (CoreVM) | Not required (CoreVM) |
