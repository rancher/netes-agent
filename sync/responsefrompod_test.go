package sync

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/stretchr/testify/assert"
)

func TestResponseFromPod(t *testing.T) {
	response := responseFromPod(v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod1",
			Annotations: map[string]string{
				"c1/io.rancher.container.uuid": "00000000-0000-0000-0000-000000000000",
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: "docker://id",
					Name:        "c1",
					Ready:       true,
				},
			},
			PodIP: "0.0.0.0",
		},
	})
	assert.Equal(t, response, client.DeploymentSyncResponse{
		ExternalId: "pod1",
		InstanceStatus: []client.InstanceStatus{
			{
				ExternalId:       "id",
				InstanceUuid:     "00000000-0000-0000-0000-000000000000",
				PrimaryIpAddress: "0.0.0.0",
			},
		},
	})
}
