package manager

import (
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/sync"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/client-go/kubernetes"
	"fmt"
)

type deploymentSyncHandler func(*kubernetes.Clientset, *watch.Client, client.DeploymentSyncRequest) (client.DeploymentSyncResponse, error)

func (m *Manager) callDeploymentSyncHandler(ignoreUnknown bool, event *events.Event, handler deploymentSyncHandler) (*client.Publish, error) {
	var request client.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return nil, err
	}

	// Only deal with "Pods"
	if request.DeploymentUnitUuid == "" {
		return emptyReply(event), nil
	}

	errUnknown := fmt.Errorf("unknown cluster %s", request.ClusterId)
	if ignoreUnknown {
		errUnknown = nil
	}

	clientset, ok := m.clientsets[request.ClusterId]
	if !ok {
		return emptyReply(event), errUnknown
	}

	watchClient, ok := m.watchClients[request.ClusterId]
	if !ok {
		return emptyReply(event), errUnknown
	}

	response, err := handler(clientset, watchClient, request)
	return createPublish(response, event), err
}

func (m *Manager) HandleComputeInstanceActivate(event *events.Event) (*client.Publish, error) {
	return m.callDeploymentSyncHandler(false, event, sync.Activate)
}

func (m *Manager) HandleComputeInstanceRemove(event *events.Event) (*client.Publish, error) {
	return m.callDeploymentSyncHandler(true, event, sync.Remove)
}
