# quota-request

A CLI tool for managing Azure quota requests.

## Overview

`quota-request` automates the process of requesting quota increases for Azure resources across subscriptions and regions. It uses Azure DefaultAzureCredential for authentication and interacts with the Azure Quota API.

## Installation

```bash
# Build the binary
make build

# Install to $GOPATH/bin
make install
```

## Usage

```bash
# Request quotas using a configuration file
quota-request request --subscription-id <subscription-id> --region <region> [flags]
```

### Flags

- `--subscription-id, -s`: Azure subscription ID (required)
- `--region, -r`: Azure region (required, e.g., eastus, westus2)
- `--config, -c`: Path to quota configuration file (default: quota-config.yaml)
- `--tenant-id, -t`: Azure tenant ID (optional, uses default credential if not specified)

### Example

```bash
# Request quotas for a subscription in East US
quota-request request \
  --subscription-id 12345678-1234-1234-1234-123456789012 \
  --region eastus \
  --config my-quotas.yaml
```

## Configuration File

The configuration file is a YAML file that defines the quota requests:

```yaml
quotas:
  - provider: Microsoft.Compute
    resourceName: standardDSv3Family
    limit: 100
  - provider: Microsoft.Compute
    resourceName: standardDSv4Family
    limit: 200
  - provider: Microsoft.Compute
    resourceName: cores
    limit: 500
```

See `quota-config.yaml.example` for a complete example.

## Authentication

The tool uses Azure DefaultAzureCredential, which supports multiple authentication methods in order:

1. Environment variables
2. Managed Identity
3. Azure CLI credentials
4. Azure PowerShell credentials

Make sure you're authenticated with `az login` or have appropriate environment variables set.

## Development

```bash
# Format code
make fmt

# Run vet
make vet

# Run tests
make test

# Build
make build

# All of the above
make all
```

## License

Licensed under the Apache License, Version 2.0. See LICENSE for details.
