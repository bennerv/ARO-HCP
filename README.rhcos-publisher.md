# rhcos-publisher — What Was Built

The `rhcos-publisher` controller is fully implemented, wired into the deployment
system, and all repo verification passes.

## Go module `rhcos-publisher/`

In `go.work`, mirroring `fleet/`'s structure: cobra `main.go`, Raw → Validate →
Complete → Run options pipeline, and a manager running four controllers under
one leader-election lease with healthz/metrics servers.

- **Installer lister** — polls the openshift/installer coreos stream JSON per
  branch (1h).
- **Marketplace lister** — polls the ARM `VirtualMachineImages` API for the
  `azureopenshift/aro4` SKUs (2h).
- **Reconciler** (event-driven) — implements the decision matrix: download →
  SHA256-verify → gunzip → re-verify → block-blob upload to `$web`, and purge
  once the marketplace serves the version.
- **Publisher** — configures Partner Center **drafts** via the Product
  Ingestion API and never submits to preview/live.

State lives in a namespaced `RHCOSRelease` CRD
(`rhcos.aro.openshift.io/v1alpha1`) — the controller only writes `/status`
(per-arch `staged → draft → published`). The ingestion client manipulates tech
configs as raw JSON maps so unmodeled draft fields survive
fetch-modify-configure round trips. Config changes restart the process via the
shared `FSWatcher`; startup purges staged VHDs of removed branches.

## Deployment

Helm chart (CRD, ConfigMap, one CR per branch, Deployment with workload
identity + 40Gi work emptyDir, ServiceAccount, ACR pull binding, RBAC —
everything except the CRD gated on `enabled`), `pipeline.yaml` (image mirror →
MSI/subscription lookups → Helm deploy with Kusto log shipping to
`rhcosPublisherLogs`), topology entry `Microsoft.Azure.ARO.HCP.RhcosPublisher`,
and Dockerfile/Makefile/Env.mk matching fleet's.

## Infrastructure

`rhcos_publisher_wi` workload identity in `svc-cluster.bicep`; storage account
module (`allowSharedKeyAccess: false`, public blob read for the static
website) + Storage Blob Data Contributor/Reader RBAC, deployed from
`svc-pipeline.yaml` behind an `if (deployRhcosPublisher)` guard, with a Shell
step reusing `scripts/storage.sh` (skipped when disabled) to enable
static-website hosting.

## Configuration

`rhcosPublisher` block in `config/config.yaml` (disabled by default, branches
4.20–4.23, RHEL 9, per-arch features), full schema in `config.schema.json`,
and INT-only enablement (`enabled` + `marketplace.enabled`) in the MSFT
overlay. Rendered configs and the new helm fixture are regenerated.

## Verification status

- `go build`/`go test` green — 30+ unit tests cover the reconciler decision
  matrix, both listers' change detection, hash verification (including cleanup
  on mismatch), SKU/version derivation (`release-4.22` + `9.8.20260520-0` →
  `aro_422-v2` / `422.98.20260520`), config validation, and the publisher
  draft flow with a fake ingestion API.
- `make lint` (whole workspace): 0 issues. `make licenses`, `fmt`, `tidy`,
  `json-format`, `yamlfmt`, `deepcopy`, `verify-schema`: all clean/stable.
- `make validate-config-pipelines` passes; materialize is stable and the helm
  fixture tests pass in verify mode.
- All touched bicep compiles; `helm template` verified in both modes
  (enabled=false renders only the CRD; enabled=true renders all 9 resources
  with a parseable embedded config).

## Deviations from the handoff plan

- Cooldowns are Go durations (`--installer-cooldown=1h`) rather than minute
  integers — cleaner flags, same config semantics.
- The CRD lives in `deploy/templates/` (sessiongate/acrpull convention) rather
  than `crds/`, so schema updates apply on upgrade.
- CR status access uses a dynamic client with typed conversion instead of a
  generated clientset — avoids adding codegen infrastructure for one small
  resource.
- The x86 Gen1 SKU is only created when `x86Features` contains
  `HyperVGeneration.V1`, making the eventual Gen1 sunset a config-only change.

## Next steps

- The Product Ingestion API models were written from the API docs and
  cloudpub's patterns but haven't been exercised against the live endpoint —
  watch the first INT run with `marketplace.enabled: true`, and grant the
  managed identity the **Manager** role on the Partner Center account
  (manual, partner.microsoft.com).
- For INT rollout, config/pipeline changes need an `sdp-pipelines` PR and a
  manual pipeline run — see aka.ms/arohcp-pipelines.
- An image build/push (`make build-rhcos-publisher`) plus a digest in config
  is needed before the INT deployment can pull anything.
