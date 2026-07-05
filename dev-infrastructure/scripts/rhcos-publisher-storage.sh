#!/bin/bash
set -euo pipefail

# Enables static website hosting on the rhcos-publisher staging storage
# account. The account only exists in environments where the publisher is
# deployed (see rhcosPublisher.enabled), so this is a no-op elsewhere.

if [[ "${DeployRhcosPublisher:-false}" != "true" ]] && [[ "${DeployRhcosPublisher:-false}" != "True" ]]; then
  echo "rhcos-publisher is not enabled in this environment, skipping static website setup"
  exit 0
fi

exec ./storage.sh
