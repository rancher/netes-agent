package sync

import (
	"fmt"
	"strings"
)

const (
	kattleLabel   = "io.rancher.kattle"
	revisionLabel = "io.rancher.revision"

	globalLabel               = "io.rancher.scheduler.global"
	hostAffinityLabel         = "io.rancher.scheduler.affinity:host_label"
	hostAntiAffinityLabel     = "io.rancher.scheduler.affinity:host_label_ne"
	hostSoftAffinityLabel     = "io.rancher.scheduler.affinity:host_label_soft"
	hostSoftAntiAffinityLabel = "io.rancher.scheduler.affinity:host_label_soft_ne"
)

func ParseLabel(label interface{}) map[string]string {
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
