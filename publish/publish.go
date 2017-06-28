package publish

import (
	"github.com/rancher/go-rancher/v2"
	"github.com/rancherlabs/kattle/utils"

	"k8s.io/client-go/pkg/api/v1"
)

func Pod(rancherClient *client.RancherClient, pod v1.Pod) error {
	data := map[string]interface{}{}
	if err := utils.ConvertByJSON(pod, &data); err != nil {
		return err
	}
	_, err := rancherClient.Publish.Create(&client.Publish{
		Data: data,
	})
	return err
}
