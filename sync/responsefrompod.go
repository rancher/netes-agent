package sync

import (
	"github.com/rancher/go-rancher/v3"
	"k8s.io/client-go/pkg/api/v1"
)

func responseFromPod(pod v1.Pod) (client.DeploymentSyncResponse, error) {
	var instanceStatuses []client.InstanceStatus
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// Don't report back on Rancher pause container
		if containerStatus.Name == rancherPauseContainerName {
			continue
		}

		// TODO
		/*hostUuid, err := hostname.UuidFromName(pod.Spec.NodeName)
		if err != nil {
			return client.DeploymentSyncResponse{}, err
		}*/

		state := ""
		// TODO: might be the wrong way to tell this
		if containerStatus.Ready {
			state = "running"
		}

		instanceStatuses = append(instanceStatuses, client.InstanceStatus{
			ExternalId: containerStatus.ContainerID,
			//HostUuid:         hostUuid,
			InstanceUuid:     containerStatus.Name,
			PrimaryIpAddress: pod.Status.PodIP,
			State:            state,
		})
	}

	return client.DeploymentSyncResponse{
		InstanceStatus: instanceStatuses,
	}, nil
}
