package sync

import (
	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
)

func primary(d client.DeploymentSyncRequest) client.Container {
	if len(d.Containers) == 1 {
		return d.Containers[0]
	}
	for _, container := range d.Containers {
		value, ok := container.Labels[labels.ServiceLaunchConfig]
		if ok && value == labels.ServicePrimaryLaunchConfig {
			return container
		}
	}
	return client.Container{}
}

func primaryContainerNameFromPod(pod v1.Pod) string {
	return pod.Labels[labels.PrimaryContainerName]
}
