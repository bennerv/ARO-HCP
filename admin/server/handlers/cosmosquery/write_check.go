// Copyright 2026 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cosmosquery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/Azure/ARO-HCP/internal/api/arm"
	"github.com/Azure/ARO-HCP/internal/utils"
)

type WriteCheckResponse struct {
	WriteAllowed bool   `json:"writeAllowed"`
	Details      string `json:"details"`
}

type WriteCheckHandler struct {
	cosmosDatabaseClient *azcosmos.DatabaseClient
}

func NewWriteCheckHandler(cosmosDatabaseClient *azcosmos.DatabaseClient) *WriteCheckHandler {
	return &WriteCheckHandler{cosmosDatabaseClient: cosmosDatabaseClient}
}

func (h *WriteCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if h.cosmosDatabaseClient == nil {
		return arm.NewCloudError(
			http.StatusInternalServerError,
			arm.CloudErrorCodeInternalServerError,
			"",
			"CosmosDB write-check endpoint is not available",
		)
	}

	containerClient, err := h.cosmosDatabaseClient.NewContainer("Resources")
	if err != nil {
		return utils.TrackError(err)
	}

	testDoc := map[string]string{
		"id":           "e2e-write-check-probe",
		"partitionKey": "e2e-write-check-probe",
		"resourceID":   "",
		"resourceType": "",
	}
	testDocBytes, err := json.Marshal(testDoc)
	if err != nil {
		return utils.TrackError(err)
	}

	pk := azcosmos.NewPartitionKeyString("e2e-write-check-probe")
	_, createErr := containerClient.CreateItem(ctx, pk, testDocBytes, nil)

	if createErr != nil {
		var responseErr *azcore.ResponseError
		if errors.As(createErr, &responseErr) && responseErr.StatusCode == http.StatusForbidden {
			resp := WriteCheckResponse{
				WriteAllowed: false,
				Details:      "CosmosDB RBAC correctly denied write access",
			}
			_, err = arm.WriteJSONResponse(w, http.StatusOK, resp)
			return utils.TrackError(err)
		}
		return utils.TrackError(createErr)
	}

	// Write succeeded — clean up and report RBAC misconfiguration
	_, _ = containerClient.DeleteItem(ctx, pk, "e2e-write-check-probe", nil)

	resp := WriteCheckResponse{
		WriteAllowed: true,
		Details:      "CosmosDB RBAC allowed write access — this deployment should be read-only",
	}
	_, err = arm.WriteJSONResponse(w, http.StatusOK, resp)
	return utils.TrackError(err)
}
