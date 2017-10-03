package sync

import (
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
)

func Sync(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest, progressResponder func(string)) (*client.DeploymentSyncResponse, error) {
	if shouldRemove(deploymentUnit) {
		podName := getPodName(deploymentUnit)
		err := deletePod(clientset, watchClient, deploymentUnit.Namespace, podName, false)
		return &client.DeploymentSyncResponse{}, err
	}

	credentialSecrets := getCredentialsFromDeploymentUnit(deploymentUnit)
	if err := reconcileSecrets(clientset, deploymentUnit.Namespace, credentialSecrets); err != nil {
		return nil, err
	}

	pod := podFromDeploymentUnit(deploymentUnit)
	createdPod, err := reconcilePod(clientset, watchClient, pod, deploymentUnit, progressResponder)
	if err != nil {
		return nil, err
	}
	if createdPod == nil {
		return nil, nil
	}

	response := responseFromPod(*createdPod)
	return &response, nil
}
