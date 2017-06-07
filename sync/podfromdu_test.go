package sync

import (
	"testing"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kattle/types"
	"github.com/stretchr/testify/assert"
)

func TestGetVolumes(t *testing.T) {
	assert.Equal(t, getVolumes(types.DeploymentUnit{
		Containers: []client.Container{
			client.Container{
				DataVolumes: []string{
					"/host/path:/container/path",
				},
			},
		},
	}), []v1.Volume{
		v1.Volume{
			Name: "host-path-volume",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/host/path",
				},
			},
		},
	})
	assert.Equal(t, len(getVolumes(types.DeploymentUnit{
		Containers: []client.Container{
			client.Container{
				DataVolumes: []string{
					"/anonymous/volume",
				},
			},
		},
	})), 0)
}

func TestGetVolumeMounts(t *testing.T) {
	assert.Equal(t, getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/host/path:/container/path",
		},
	}), []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "host-path-volume",
			MountPath: "/container/path",
		},
	})
	assert.Equal(t, len(getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/anonymous/volume",
		},
	})), 0)
}
