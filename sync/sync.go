package sync

import (
	"github.com/rancher/go-rancher-metadata/metadata"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func Sync(clientset *kubernetes.Clientset, deploymentUnits []metadata.DeploymentUnit, volumes []metadata.Volume) error {
	var pods []v1.Pod
	for _, deploymentUnit := range deploymentUnits {
		pods = append(pods, PodFromDeploymentUnit(deploymentUnit))
	}

	volumeIds := map[string]bool{}
	for _, deploymentUnit := range deploymentUnits {
		for _, container := range deploymentUnit.Containers {
			for _, mount := range container.Mounts {
				volumeIds[mount.VolumeId] = true
			}
		}
	}

	if err := reconcileVolumes(clientset, volumes, volumeIds); err != nil {
		return err
	}

	return reconcilePods(clientset, pods)
}
