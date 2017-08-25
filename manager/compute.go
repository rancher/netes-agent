package manager

import (
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/sync"
)

func (m *Manager) HandleComputeInstanceActivate(event *events.Event) (*client.Publish, error) {
	var request client.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return nil, err
	}

	clientset, ok := m.clientsets[m.currentClusterId]
	if !ok {
		return nil, nil
	}

	watchClient, ok := m.watchClients[m.currentClusterId]
	if !ok {
		return nil, nil
	}

	response, err := sync.Activate(clientset, watchClient, request)
	return createPublish(response, err, event), nil
}

func (m *Manager) HandleComputeInstanceRemove(event *events.Event) (*client.Publish, error) {
	var request client.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return nil, err
	}

	clientset, ok := m.clientsets[m.currentClusterId]
	if !ok {
		return nil, nil
	}

	watchClient, ok := m.watchClients[m.currentClusterId]
	if !ok {
		return nil, nil
	}

	response, err := sync.Remove(clientset, watchClient, request)
	return createPublish(response, err, event), nil
}
