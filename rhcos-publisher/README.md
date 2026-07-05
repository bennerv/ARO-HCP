# rhcos-publisher

A Kubernetes controller that automates downloading, verifying, and hosting
RHCOS (Red Hat Enterprise Linux CoreOS) VHD images for Azure, and configures
them as drafts in the ARO 1P Marketplace offering for human review and
publish.

## Background

ARO HCP boots node VMs from RHCOS images in the Azure Marketplace. When a new
OCP minor version is released, the corresponding RHCOS VHD must be published
to the ARO marketplace offering (publisher `azureopenshift`, offer `aro4`).
Today this is a manual process owned by ARO SRE — downloading VHDs, uploading
them to a storage account, and configuring marketplace offerings through the
Partner Center portal.

This controller automates the workflow end-to-end: detecting new images,
downloading/verifying/hosting VHDs, and configuring marketplace drafts. It
must run within Microsoft IP address space because the marketplace operations
require it (so it cannot run in Prow); it is deployed as a Deployment on the
ARO-HCP INT service cluster and authenticates with Azure Workload Identity.

## Architecture

One process, four controllers, each with its own workqueue:

1. **Installer lister** (1h interval) — polls GitHub for the coreos stream
   metadata (`data/data/coreos/coreos-rhel-<N>.json`) of each configured
   openshift/installer release branch. Caches release versions, download URLs
   and SHA256 hashes. Enqueues a reconcile key when a branch/architecture
   pair's image changes.

2. **Marketplace lister** (2h interval) — polls the Azure Marketplace
   (`VirtualMachineImages` ARM API) for the image versions of the ARO
   offering's SKUs. Enqueues a reconcile key when a version set changes.

3. **Reconciler** (event-driven) — compares the two views:
   - New release, not in marketplace, not staged: download the compressed
     VHD, verify its SHA256, decompress (~16 GiB), verify again, upload to
     the storage account's `$web` container (static website).
   - Staged and marketplace publishing enabled: hand off to the publisher.
   - Release visible in the marketplace: purge the staged VHD.

4. **Marketplace publisher** (event-driven, INT only) — configures the
   marketplace plans and image version as a **draft** through the Partner
   Center [Product Ingestion API](https://learn.microsoft.com/en-us/partner-center/marketplace/product-ingestion-api).
   It never submits the draft to preview/live — a human reviews and publishes
   in Partner Center.

State is tracked in namespaced `RHCOSRelease` custom resources (one per
branch, created by the Helm chart). The controller only writes the status
subresource; per architecture the phase moves `staged` → `draft` →
`published`:

    $ kubectl get rhcosreleases -n rhcos-publisher
    NAME           BRANCH         RHEL   X86_64      AARCH64     AGE
    release-4-22   release-4.22   9      draft       published   12d

Configuration changes (mounted ConfigMap) trigger a graceful process restart.
On startup, staged VHDs of branches that are no longer configured are purged.
Marketplace offerings are never removed — older clusters may still reference
them.

## Configuration

Branches and marketplace settings are defined in the ARO-HCP config system
(`config/config.yaml`, key `rhcosPublisher`) and rendered into Helm values.
The controller derives everything else:

- stream file: `rhelVersion: 9` → `coreos-rhel-9.json`
- SKU names: `release-4.22` → `aro_422` (x86 Gen1), `aro_422-v2` (x86 Gen2),
  `aro_422-arm` (ARM Gen2). The Gen1 SKU only exists when `x86Features`
  contain `HyperVGeneration.V1`.
- image version: release `9.8.20260520-0` on branch 4.22 →
  marketplace version `422.98.20260520`

## Deployment

- **Environment**: INT only (`rhcosPublisher.enabled: true` in the MSFT INT
  overlay). In all other environments the Helm chart renders only the CRD.
- **Type**: Deployment (1 replica) with leader election.
- **Auth**: Azure Workload Identity; the managed identity has Storage Blob
  Data Contributor + Reader on the staging storage account and needs the
  Manager role on the Partner Center marketplace account.
- **Storage**: static website hosting on the staging storage account (`$web`
  container, shared key access disabled, no SAS). Endpoints are discovered at
  runtime from the account properties.
- **Observability**: `/healthz`, `/metrics`; structured JSON logs shipped to
  Kusto (`rhcosPublisherLogs`).

## Local Development

    cd rhcos-publisher
    go build -o rhcos-publisher .

    ./rhcos-publisher controller \
      --config config.yaml \
      --subscription-id <sub-id> \
      --resource-group <rg> \
      --storage-account-name <name> \
      --kube-namespace rhcos-publisher \
      --installer-cooldown 1m \
      --marketplace-cooldown 2m \
      --marketplace-publish-enabled=false

With `--marketplace-publish-enabled=false` the controller stages VHDs and
reports their URLs but never talks to the Partner Center API (which requires
MSIT credentials). Note that the controller expects to run in-cluster (it
reads the in-cluster kubeconfig for leader election and RHCOSRelease status);
run it against a kind/dev cluster with the CRD applied.

## Testing

    go test ./...

The listers, the reconciler decision matrix, hash verification, config
validation and the ingestion draft flow are covered by unit tests with fake
clients.
