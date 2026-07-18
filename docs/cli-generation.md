# ARO HCP CLI Generation (AAZ)

This document captures the current workflow used to generate an Azure CLI extension from ARO HCP swagger.

## Purpose

Generate/update an extension (currently named `aro-hcp`) from swagger tag `package-2025-12-23-preview` using `azdev` + `aaz-dev`.

## Prerequisites

- Python 3.9+ installed
- Git installed

## Setup

This section covers the one-time workspace setup. For full details on the aaz-dev workspace editor and its capabilities, see the [AAZ Dev Tools documentation](https://azure.github.io/aaz-dev-tools/pages/usage/workspace-editor/).

### Clone required repositories

#### Core repos (always required)

```bash
git clone https://github.com/Azure/aaz.git
git clone https://github.com/Azure/azure-cli.git
git clone https://github.com/Azure/azure-cli-extensions.git
```

#### API specs repo

If the API spec you are generating from has already been merged to the public repo:

```bash
git clone https://github.com/Azure/azure-rest-api-specs.git
```

If the API spec is still in a pull request or has not yet been made public, use the PR repo and check out the branch containing your spec:

```bash
git clone https://github.com/Azure/azure-rest-api-specs-pr.git
cd azure-rest-api-specs-pr
git checkout <your-branch-name>
```

> **Note:** When the swagger lives in this ARO-HCP repo under `api/redhatopenshift/`, you can point `aaz-dev` directly at that path instead of cloning a separate specs repo. The generation commands below use `$PWD/api/redhatopenshift/` for this reason.

In the examples below, replace paths like `/path/to/aaz`, `/path/to/azure-cli`, and `/path/to/azure-cli-extensions` with wherever you cloned these repos.

### Python virtual environment and tooling

Create a virtual environment in the ARO-HCP repo root and install the required tools:

```bash
cd /path/to/ARO-HCP
python3 -m venv .venv
source .venv/bin/activate
pip install -U pip
pip install azdev aaz-dev
```

### Set up azdev

`azdev` is the Azure CLI development tool. It needs to know about the `azure-cli` repo (the core CLI) and the `azure-cli-extensions` repo (where extensions like ours live). Setting it up correctly is required before you can lint, test, or run your extension locally.

Activate your virtual environment, then run `azdev setup` pointing at both repos:

```bash
source .venv/bin/activate

azdev setup -c /path/to/azure-cli -r /path/to/azure-cli-extensions/
```

`-c` sets the path to the core `azure-cli` repo. `-r` sets the path to the extensions repo. This configures `azdev` so it can resolve imports, run linters, and discover tests across both.

Next, register the `aro-hcp` extension so `azdev` commands can find it:

```bash
azdev extension add aro-hcp
```

Verify the extension is visible:

```bash
azdev extension list | grep aro-hcp
```

> **Tip:** You need to re-run `azdev extension add aro-hcp` any time you regenerate the extension from scratch or switch to a fresh virtual environment. If `azdev linter` or `azdev test` reports `unrecognized modules`, this is usually why.

## Pruning the command tree with the workspace editor

Before generating code, use the `aaz-dev` workspace editor UI to import swagger resources and prune the command tree. Start the editor:

```bash
source .venv/bin/activate
aaz-dev run
```

This opens a browser-based UI where you can create or open a workspace, add swagger resources, and edit the generated command tree before exporting.

> **Important:** When adding resources to the workspace, use **Swagger** as the source, not TypeSpec. The TypeSpec import path in `aaz-dev` has a bug where generic type names with angle brackets (e.g. `<RECORD>`) are emitted as-is into generated Python, producing invalid identifiers. See [aaz-dev-tools#562](https://github.com/Azure/aaz-dev-tools/issues/562).

When importing a new swagger resource into the command tree, you should remove the following commands at both the cluster and nodepool levels:

- `identity assign`
- `identity remove`
- `identity show`

ARO HCP requires all managed identities to be assigned at cluster creation time. The identity assign/remove commands are not supported post-creation and will fail at the API level, so they should not be exposed in the CLI.

After pruning, click **Export** in the workspace editor to write the command models to the `aaz` repo, then proceed to the generation commands below.

> **Note:** The exported command models are currently staged on a branch in the `aaz` repo, not yet merged to `main`. The CLI code is generated from that branch. Make sure your local `/path/to/aaz` checkout is on the correct branch before running the generation commands.

## Generation commands

Run from this repository root with the virtual environment activated:

```bash
source .venv/bin/activate

aaz-dev command-model generate-from-swagger \
  -a /path/to/aaz \
  --sm "$PWD/api/redhatopenshift/" \
  -m aro-hcp \
  --rp Microsoft.RedHatOpenShift \
  --swagger-tag package-2025-12-23-preview

aaz-dev cli generate-by-swagger-tag \
  -a /path/to/aaz \
  -e /path/to/azure-cli-extensions/ \
  --name aro-hcp \
  --sm "$PWD/api/redhatopenshift/" \
  --rp Microsoft.RedHatOpenShift \
  --tag package-2025-12-23-preview \
  --profile latest
```

## Post-generation customizations

After code generation, customizations are applied in `custom.py` and `commands.py` (which are not overwritten by regeneration). See the [AAZ customization docs](https://azure.github.io/aaz-dev-tools/pages/usage/customization/) for details on the inheritance pattern.

Current customizations:

- **`request-admin-credential`**: exposes the kubeconfig (hidden by default as a secret), replaces literal `\n` with real newlines, and adds `--file` to write the kubeconfig directly to a file.
- **`cluster create`**: injects `identity.type = "UserAssigned"` into the request body. The generated code sets `userAssignedIdentities` but doesn't set the required ARM `identity.type` field.

TODO:

| Area | Item | Details |
|---|---|---|
| Customization | `external-auth create` — flatten `--clients` | The `--clients` argument uses `array<object>` shorthand syntax which is not user-friendly. Flatten into simpler arguments (e.g. `--client-id`, `--client-type`) using `_build_arguments_schema` and `pre_operations` in `custom.py`. |
| Description fix | `request-admin-credential` — typo | "for you Azure Red Hat OpenShift" should be "for your". Fix in the swagger/TypeSpec model. |
| Description fix | `cluster delete` — plural | "Delete ... Clusters" should be singular "Cluster". Fix in the swagger/TypeSpec model. |
| Description fix | `nodepool list` — trailing "by Cluster" | Should be "List ... Node Pools" to match the pattern of other list commands. Fix in the swagger/TypeSpec model. |
| Description fix | `version show --version` | Description says "The name of the HcpOpenShiftVersion" — should be human-readable like "The version of OpenShift". Fix in the swagger/TypeSpec model. |
| Description fix | `nodepool create --disk-encryption-set` — typo | "reosurce" should be "resource". Fix in the swagger/TypeSpec model. |
| UX | `--cluster-name` short alias | Add `-c` as a short alias for `--cluster-name` across nodepool and external-auth commands. |
| UX | `external-auth create/update` — groups args inconsistency | Create uses `--claim`/`--prefix` under "Groups Arguments", update uses `--groups` under "Mappings Arguments". Align parameter names between create and update. |
| Customization | `cluster create` — identity type | `UserAssigned` is the only supported identity type for ARO HCP clusters. Currently hardcoded to `UserAssigned` in `custom.py` by overriding the content builder. Consider exposing it as a validated arg if other types are supported in the future. |
| Investigation | `external-auth create` LRO delay | External auth LRO sat in `Accepted` for an extended period before reaching `Succeeded`. Resource ID: `/subscriptions/1d3378d3-5a3f-4712-85a1-2485495dfc4b/resourceGroups/private-keyvault-sxf56l/providers/Microsoft.RedHatOpenShift/hcpOpenShiftClusters/private-kv-cluster/externalAuths/hackathon-auth` |

## Post-generation lint compatibility patch

Current generated AAZ args can fail `azdev` rule `option_length_too_long` for:

- `--hcp-open-shift-cluster-name`
- `--node-drain-timeout-minutes`

Apply short aliases in generated AAZ files before linting:

```bash
cd /path/to/azure-cli-extensions

find src/aro-hcp/azext_aro_hcp/aaz/latest -type f -name '*.py' -print0 | \
  xargs -0 perl -0777 -pi -e 's/options=\["--hcp-open-shift-cluster-name"\]/options=["-c", "--cluster-name", "--hcp-open-shift-cluster-name"]/g; s/options=\["-n", "--name", "--hcp-open-shift-cluster-name"\]/options=["-n", "--name", "--cluster-name", "--hcp-open-shift-cluster-name"]/g; s/options=\["--node-drain-timeout-minutes"\]/options=["-d", "--drain-timeout", "--node-drain-timeout-minutes"]/g'
```

## Optional command root rewrite (`az arohcp`)

By default, generated commands are rooted at `az red-hat-open-shift`.

If you want `az arohcp`, rewrite command names in generated AAZ files:

```bash
cd /path/to/azure-cli-extensions/src/aro-hcp

find azext_aro_hcp/aaz/latest/red_hat_open_shift -type f -name '*.py' -print0 | \
  xargs -0 sed -i 's/red-hat-open-shift/arohcp/g'

find azext_aro_hcp/aaz/latest/red_hat_open_shift -type f -name '*.py' -print0 | \
  xargs -0 sed -i 's/Manage Red Hat Open Shift/Manage Red Hat OpenShift Hosted Control Plane Resources/g'
```

Then verify:

```bash
az arohcp -h
```

Quick check before/after rewrite:

```bash
# before rewrite (default generated root)
az red-hat-open-shift -h

# after rewrite + reinstall/reload extension
az arohcp -h
```

## How to find the current API version/tag

Use the HCP swagger readme files in this repo:

- `api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/hcpclusters/preview/readme.md`
- `api/readme.md`

Find latest package tag:

```bash
rg -n "package-[0-9]{4}-[0-9]{2}-[0-9]{2}-preview" \
  api/redhatopenshift/resource-manager/Microsoft.RedHatOpenShift/hcpclusters/preview/readme.md
```

Find server model tags:

```bash
rg -n "Tag v20[0-9]{6}preview" api/readme.md
```

As of this update, latest HCP preview package tag is:

- `package-2025-12-23-preview`

## Where to plug in the version/tag

Replace the tag in both generation steps:

```bash
# command model step
--swagger-tag package-2025-12-23-preview

# CLI codegen step
--tag package-2025-12-23-preview
```

If you are generating server models via `api/readme.md`, use the matching `vYYYYMMDDpreview` tag there (for example `v20251223preview`).

## Expected output locations

- Generated extension code:
  - `/path/to/azure-cli-extensions/src/aro-hcp`
- MVP vendored copy in this repo (for standalone testing):
  - `tooling/arohcp-cli`
- Generated/updated AAZ command model artifacts:
  - `/path/to/aaz/Commands/...`
  - `/path/to/aaz/Resources/...`

To refresh the vendored MVP extension in this repo:

```bash
mkdir -p tooling/arohcp-cli
rsync -a --delete \
  /path/to/azure-cli-extensions/src/aro-hcp/ \
  tooling/arohcp-cli/
```

For MVP standalone testing, apply the same rewrite from
`red-hat-open-shift` to `arohcp` described in
Optional command root rewrite (`az arohcp`), but run it against:

- `tooling/arohcp-cli/azext_aro_hcp/aaz/latest/red_hat_open_shift`

## Validation

From `azure-cli-extensions` repo:

```bash
cd /path/to/azure-cli-extensions
azdev linter aro-hcp
azdev test aro-hcp --discover
```

Command root expectations:

- Without the optional rewrite, commands are available under `az red-hat-open-shift`.
- After applying the optional rewrite and reinstalling/reloading the extension, commands are available under `az arohcp`.

Notes from current generation run:

- `azdev linter aro-hcp` passes after generation when the local extension has been added via `azdev extension add aro-hcp`.
- Generated test scaffold currently contains no test methods (`azext_aro_hcp/tests/latest/test_aro_hcp.py` is a TODO template), so `azdev test aro-hcp --discover` may run with `0 items` until real test cases are added.

Manual smoke check after local extension install:

```bash
az extension add --source /path/to/azure-cli-extensions/src/aro-hcp/dist/*.whl -y
az red-hat-open-shift -h
az arohcp -h
```

Interpretation:

- If rewrite is **not** applied: `az red-hat-open-shift -h` should work, `az arohcp -h` is expected to fail.
- If rewrite **is** applied: `az arohcp -h` should work after reinstalling the extension wheel.

If `az arohcp -h` is missing, apply the optional command root rewrite section above and reinstall the extension wheel.

Or build/install from the vendored copy:

```bash
cd tooling/arohcp-cli
python -m pip install -U build
python -m build --wheel
az extension add --source dist/*.whl -y
az arohcp -h
```

## Known warnings seen during generation

- Read-only property requirement warnings (for example: `url`, `issuerUrl`) can appear.
- Wait-command support warnings for non-standard operations (for example credential request/revoke paths) can appear.

These were non-fatal in generation and did not stop artifact creation.

## Troubleshooting

### `azdev setup` crash with `get_env_path() ... NoneType`

Cause: `azdev` expects a virtual environment marker.

Fix:

```bash
source .venv/bin/activate
azdev setup -r /path/to/azure-cli-extensions/
```

### `aaz-dev: command not found`

Install into the active venv:

```bash
pip install aaz-dev
```

### `unrecognized modules: [ aro-hcp ]` during `azdev linter` or `azdev test`

Cause: the extension is generated on disk but is not registered in the current `azdev` dev environment.

Fix:

```bash
cd /path/to/azure-cli-extensions
azdev extension add aro-hcp
```

You can verify visibility with:

```bash
azdev extension list | grep aro-hcp
```

### `extension(s): [ aro-hcp ] installed from a wheel may need --include-whl-extensions option`

Cause: `azdev linter` detected wheel-installed extension state in the current environment.

Fix:

```bash
cd /path/to/azure-cli-extensions
azdev linter aro-hcp --include-whl-extensions
```

### `... is not a valid git repository`

Ensure target repos are cloned and paths are correct:

```bash
ls -ld /path/to/azure-cli-extensions /path/to/aaz
```

> Note: use `/path/to/aaz` (not `~/.workspace/aaz`).

### `Path '/absolute/path/to/ARO-HCP/api/redhatopenshift' does not exist`

Cause: the sample path is a placeholder and not a real directory on your machine.

Fix: run from repo root and use `$PWD` (or your full absolute path):

```bash
source .venv/bin/activate
aaz-dev command-model generate-from-swagger \
  -a /path/to/aaz \
  --sm "$PWD/api/redhatopenshift/" \
  -m aro-hcp \
  --rp Microsoft.RedHatOpenShift \
  --swagger-tag package-2025-12-23-preview
```
