package sync

import (
	"testing"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/stretchr/testify/assert"
)

func TestResponseFromPod(t *testing.T) {
	response := responseFromPod(v1.Pod{
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: "docker://id",
					Name:        "00000000-0000-0000-0000-000000000000",
					Ready:       true,
				},
			},
			PodIP: "0.0.0.0",
		},
	})
	assert.Equal(t, response, client.DeploymentSyncResponse{
		InstanceStatus: []client.InstanceStatus{
			{
				ExternalId:       "docker://id",
				InstanceUuid:     "00000000-0000-0000-0000-000000000000",
				PrimaryIpAddress: "0.0.0.0",
				State:            "running",
			},
		},
	})
}
