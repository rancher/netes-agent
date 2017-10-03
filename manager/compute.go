package manager

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/sync"
	"github.com/rancher/netes-agent/utils"
)

type deploymentSyncHandler func(request client.DeploymentSyncRequest) (*client.Publish, error)

func callDeploymentSyncHandler(event *events.Event, handler deploymentSyncHandler) (*client.Publish, error) {
	var request client.DeploymentSyncRequest
	if err := utils.ConvertByJSON(event.Data["deploymentSyncRequest"], &request); err != nil {
		return nil, err
	}

	// Only deal with "Pods"
	if request.DeploymentUnitUuid == "" {
		return emptyReply(event), nil
	}

	return handler(request)
}

func (m *Manager) HandleComputeSync(event *events.Event, rancherClient Client) (*client.Publish, error) {
	return callDeploymentSyncHandler(event, func(request client.DeploymentSyncRequest) (*client.Publish, error) {
		clientset, watchClient, err := m.getCluster(rancherClient, request.ClusterId)
		if err != nil {
			return nil, fmt.Errorf("Failure with cluster %s: %v", request.ClusterId, err)
		}

		progressResponder := func(message string) {
			publish := emptyReply(event)
			publish.Transitioning = "yes"
			publish.TransitioningMessage = message
			if err := reply(publish, event, rancherClient); err != nil {
				log.Errorf("Failed to publish progress: %v", err)
			}
		}

		response, err := sync.Sync(clientset, watchClient, request, progressResponder)
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, nil
		}
		return createPublish(response, event), nil
	})
}
