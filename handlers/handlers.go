package handlers

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	v3 "github.com/rancher/go-rancher/v3"
	"github.com/rancherlabs/kattle/sync"
	"github.com/rancherlabs/kattle/watch"
)

// TODO
var (
	WatchClient *watch.Client
	Clientset   *kubernetes.Clientset
)

func GetHandlers() map[string]events.EventHandler {
	return map[string]events.EventHandler{
		"compute.instance.activate": handleComputeInstanceActivate,
		"compute.instance.remove":   handleComputeInstanceRemove,
		// TODO
		"ping": func(event *events.Event, apiClient *client.RancherClient) error {
			return nil
		},
	}
}

func handleComputeInstanceActivate(event *events.Event, apiClient *client.RancherClient) error {
	var request v3.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return err
	}

	response, err := sync.Activate(Clientset, WatchClient, request)
	if err != nil {
		return err
	}

	return reply(response, event, apiClient)
}

func handleComputeInstanceRemove(event *events.Event, apiClient *client.RancherClient) error {
	var request v3.DeploymentSyncRequest
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		return err
	}
	return sync.Remove(Clientset, WatchClient, request)
}

func reply(response v3.DeploymentSyncResponse, event *events.Event, apiClient *client.RancherClient) error {
	reply := &client.Publish{
		ResourceId: event.ResourceID,
		PreviousIds: []string{
			event.ID,
		},
		ResourceType: event.ResourceType,
		Name:         event.ReplyTo,
		Data:         structs.Map(response),
		Time:         time.Now().UnixNano() / int64(time.Millisecond),
	}

	log.Infof("Reply: %+v", reply)

	_, err := apiClient.Publish.Create(reply)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}

	return nil
}
