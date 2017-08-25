package manager

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/structs"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
)

func wrapHandler(handler func(event *events.Event) (*client.Publish, error)) func(event *events.Event, apiClient *client.RancherClient) error {
	return func(event *events.Event, apiClient *client.RancherClient) error {
		publish, err := handler(event)
		if err == nil {
			return reply(publish, event, apiClient)
		} else {
			return reply(publish, event, apiClient)
		}
	}
}

func createPublish(response client.DeploymentSyncResponse, responseErr error, event *events.Event) *client.Publish {
	reply := client.Publish{
		ResourceId: event.ResourceID,
		PreviousIds: []string{
			event.ID,
		},
		ResourceType: event.ResourceType,
		Name:         event.ReplyTo,
		Time:         time.Now().UnixNano() / int64(time.Millisecond),
	}

	if responseErr == nil {
		reply.Data = structs.Map(response)
	} else {
		reply.Transitioning = "true"
		reply.TransitioningMessage = responseErr.Error()
	}

	return &reply
}

func reply(publish *client.Publish, event *events.Event, apiClient *client.RancherClient) error {
	log.Infof("Reply: %+v", publish)

	_, err := apiClient.Publish.Create(publish)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}

	return nil
}
