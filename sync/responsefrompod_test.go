package sync

import (
	"testing"

	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/stretchr/testify/assert"
)

func TestResponseFromPod(t *testing.T) {
	response := responseFromPod(v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "podname",
		},
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
		ExternalId: "podname",
		InstanceStatus: []client.InstanceStatus{
			{
				ExternalId:       "docker://id",
				InstanceUuid:     "00000000-0000-0000-0000-000000000000",
				PrimaryIpAddress: "0.0.0.0",
			},
		},
	})
}
