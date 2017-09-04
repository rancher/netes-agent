package manager

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
)

func wrapHandler(handler func(event *events.Event) (*client.Publish, error)) func(event *events.Event, apiClient *client.RancherClient) error {
	return func(event *events.Event, apiClient *client.RancherClient) error {
		publish, err := handler(event)
		if err == nil {
			return reply(publish, event, apiClient)
		} else {
			return err
		}
	}
}

func emptyReply(event *events.Event) *client.Publish {
	return &client.Publish{
		PreviousId: event.ID,
		Name:       event.ReplyTo,
	}
}

func createPublish(response client.DeploymentSyncResponse, event *events.Event) *client.Publish {
	reply := emptyReply(event)
	reply.Data = map[string]interface{}{
		"deploymentSyncResponse": response,
	}
	return reply
}

func reply(publish *client.Publish, event *events.Event, apiClient *client.RancherClient) error {
	log.Infof("Reply: %+v", publish)

	_, err := apiClient.Publish.Create(publish)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}

	return nil
}
