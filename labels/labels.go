package labels

import (
	"fmt"
	"strings"
)

const (
	RevisionLabel       = "io.rancher.revision"
	DeploymentUuidLabel = "io.rancher.deployment.uuid"
	ContainerUuidLabel  = "io.rancher.container.uuid"

	GlobalLabel                    = "io.rancher.scheduler.global"
	HostAffinityLabel              = "io.rancher.scheduler.affinity:host_label"
	HostAntiAffinityLabel          = "io.rancher.scheduler.affinity:host_label_ne"
	HostSoftAffinityLabel          = "io.rancher.scheduler.affinity:host_label_soft"
	HostSoftAntiAffinityLabel      = "io.rancher.scheduler.affinity:host_label_soft_ne"
	ContainerAffinityLabel         = "io.rancher.scheduler.affinity:container_label"
	ContainerAntiAffinityLabel     = "io.rancher.scheduler.affinity:container_label_ne"
	ContainerSoftAffinityLabel     = "io.rancher.scheduler.affinity:container_label_soft"
	ContainerSoftAntiAffinityLabel = "io.rancher.scheduler.affinity:container_label_soft_ne"

	ServiceLaunchConfig        = "io.rancher.service.launch.config"
	ServicePrimaryLaunchConfig = "io.rancher.service.primary.launch.config"
)

func Parse(label interface{}) map[string]string {
	labelMap := map[string]string{}
	kvPairs := strings.Split(fmt.Sprint(label), ",")
	for _, kvPair := range kvPairs {
		kv := strings.SplitN(kvPair, "=", 2)
		if len(kv) > 1 {
			labelMap[kv[0]] = kv[1]
		}
	}
	return labelMap
}
