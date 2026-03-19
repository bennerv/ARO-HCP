# Internal Go Client SDK for Testing

This document describes how the internal Go client SDK used for end-to-end
testing is generated from the TypeSpec API definitions.

## Generation

The full Go client SDK (including clients, fakes, models, etc.) is generated
directly from TypeSpec using the `@azure-tools/typespec-go` emitter. The
generated code lives in
`test/sdk/resourcemanager/redhatopenshifthcp/armredhatopenshifthcp/` with
`package armredhatopenshifthcp`.

To regenerate the test SDK:

```bash
cd api
make testsdk
```

### API Version

The test SDK API version is controlled by `TESTSDK_VERSION` in the Makefile
(default: `v20240610preview`).

> [!WARNING]
> Before changing the test SDK API version, make sure the new API version has
> been fully deployed to all Azure regions by way of the ARO-HCP
> [ARM manifest](https://msazure.visualstudio.com/AzureRedHatOpenShift/_git/Arm-Manifests).
