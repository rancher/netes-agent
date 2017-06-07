package sync

import (
	"fmt"
	"testing"

	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kattle/types"
)

func TestGetVolumes(t *testing.T) {
	fmt.Println(getVolumes(types.DeploymentUnit{
		Containers: []client.Container{
			client.Container{
				DataVolumes: []string{
					"/host/path:/container/path",
				},
			},
		},
	}))
	fmt.Println(getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/host/path:/container/path",
		},
	}))
}
