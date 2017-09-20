package sync

import (
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
)

func Activate(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest, progressResponder func(*client.DeploymentSyncResponse, string)) (*client.DeploymentSyncResponse, error) {
	credentialSecrets := getCredentialsFromDeploymentUnit(deploymentUnit)
	if err := reconcileSecrets(clientset, deploymentUnit.Namespace, credentialSecrets); err != nil {
		return nil, err
	}

	pod := podFromDeploymentUnit(deploymentUnit)
	createdPod, err := reconcilePod(clientset, watchClient, pod, progressResponder)
	if err != nil {
		return nil, err
	}
	if createdPod == nil {
		return nil, nil
	}

	response := responseFromPod(*createdPod)
	return &response, nil
}

func Remove(clientset *kubernetes.Clientset, watchClient *watch.Client, deploymentUnit client.DeploymentSyncRequest) error {
	podName := deploymentUnit.ExternalId
	if podName == "" {
		podName = getPodName(deploymentUnit)
	}
	return deletePod(clientset, watchClient, deploymentUnit.Namespace, podName)
}
