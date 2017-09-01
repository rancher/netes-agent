package sync

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
)

func Activate(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest) (client.DeploymentSyncResponse, error) {
	/*volumeIds := map[string]bool{}
	for _, deploymentUnit := range deploymentUnits {
		for _, container := range deploymentUnit.Containers {
			for _, mount := range container.Mounts {
				volumeIds[mount.VolumeId] = true
			}
		}
	}

	if err := reconcileVolumes(clientset, watchClient, volumes, volumeIds); err != nil {
		return err
	}*/

	pod := podFromDeploymentUnit(deploymentUnit)
	createdPod, err := reconcilePod(clientset, watchClient, pod)
	if err != nil {
		return client.DeploymentSyncResponse{}, err
	}

	return responseFromPod(createdPod), nil
}

// TODO
func Remove(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest) (client.DeploymentSyncResponse, error) {
	log.Info("Remove")
	return client.DeploymentSyncResponse{}, nil
}
