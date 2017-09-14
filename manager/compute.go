package manager

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/sync"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
)

type deploymentSyncHandler func(*kubernetes.Clientset, *watch.Client, client.DeploymentSyncRequest, func(client.DeploymentSyncResponse, string)) (client.DeploymentSyncResponse, error)

func (m *Manager) callDeploymentSyncHandler(ignoreClusterErrors bool, event *events.Event, apiClient *client.RancherClient, handler deploymentSyncHandler) (*client.Publish, error) {
	var request client.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return nil, err
	}

	// Only deal with "Pods"
	if request.DeploymentUnitUuid == "" {
		return emptyReply(event), nil
	}

	clientset, watchClient, err := m.getCluster(request.ClusterId)
	if err != nil {
		err = fmt.Errorf("Failure with cluster %s: %v", request.ClusterId, err)
		if ignoreClusterErrors {
			err = nil
		}
		return emptyReply(event), err
	}

	progressResponder := func(progressResponse client.DeploymentSyncResponse, message string) {
		publish := createPublish(progressResponse, event)
		publish.Transitioning = "yes"
		publish.TransitioningMessage = message
		if err := reply(publish, event, apiClient); err != nil {
			log.Errorf("Failed to publish progress: %v", err)
		}
	}

	response, err := handler(clientset, watchClient, request, progressResponder)
	return createPublish(response, event), err
}

func (m *Manager) HandleComputeInstanceActivate(event *events.Event, apiClient *client.RancherClient) (*client.Publish, error) {
	return m.callDeploymentSyncHandler(false, event, apiClient, sync.Activate)
}

func (m *Manager) HandleComputeInstanceRemove(event *events.Event, apiClient *client.RancherClient) (*client.Publish, error) {
	return m.callDeploymentSyncHandler(true, event, apiClient, sync.Remove)
}
