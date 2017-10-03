package manager

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
)

func wrapHandler(handler func(event *events.Event, rancherClient Client) (*client.Publish, error)) func(event *events.Event, apiClient *client.RancherClient) error {
	return func(event *events.Event, apiClient *client.RancherClient) error {
		rancherClient := NewRancherClient(apiClient)
		publish, err := handler(event, rancherClient)
		if err != nil {
			return err
		}
		if publish != nil {
			return reply(publish, event, rancherClient)
		}
		return nil
	}
}

func emptyReply(event *events.Event) *client.Publish {
	return &client.Publish{
		PreviousId: event.ID,
		Name:       event.ReplyTo,
	}
}

func createPublish(response *client.DeploymentSyncResponse, event *events.Event) *client.Publish {
	if response == nil {
		return nil
	}
	reply := emptyReply(event)
	reply.Data = map[string]interface{}{
		"deploymentSyncResponse": response,
	}
	return reply
}

func reply(publish *client.Publish, event *events.Event, rancherClient Client) error {
	log.Infof("Reply: %+v", publish)

	if err := rancherClient.Publish(publish); err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}

	return nil
}
