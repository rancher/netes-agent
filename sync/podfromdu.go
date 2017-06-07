package sync

import (
	"fmt"
	"strings"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/rancher/kattle/types"
)

func PodFromDeploymentUnit(deploymentUnit types.DeploymentUnit) v1.Pod {
	var containers []v1.Container
	for _, container := range deploymentUnit.Containers {
		image := strings.SplitN(container.ImageUuid, ":", 2)[1]
		var environment []v1.EnvVar
		for k, v := range container.Environment {
			environment = append(environment, v1.EnvVar{
				Name:  k,
				Value: fmt.Sprint(v),
			})
		}
		containers = append(containers, v1.Container{
			Name:  container.Uuid,
			Image: image,
			Env:   environment,
			SecurityContext: &v1.SecurityContext{
				Privileged: &container.Privileged,
			},
			// TODO: entrypoint, commands
		})
	}

	primaryConfig := deploymentUnit.RevisionConfig.LaunchConfig

	return v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: deploymentUnit.Uuid,
			Labels: map[string]string{
				"io.rancher.kattle":   "true",
				"io.rancher.revision": deploymentUnit.RevisionId,
			},
		},
		Spec: v1.PodSpec{
			Containers:  containers,
			HostIPC:     primaryConfig.IpcMode == "host",
			HostNetwork: primaryConfig.NetworkMode == "host",
			HostPID:     primaryConfig.PidMode == "host",
			// Handle global service case
			NodeName: deploymentUnit.Host.Name,
		},
	}
}
