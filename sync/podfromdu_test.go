package sync

import (
	"testing"

	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/utils"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

func TestGetLabels(t *testing.T) {
	assert.Equal(t, getLabels(client.DeploymentSyncRequest{
		Revision:           "revision",
		DeploymentUnitUuid: "00000000-0000-0000-0000-000000000001",
		Containers: []client.Container{
			{
				Name: "test",
				Uuid: "00000000-0000-0000-0000-000000000002",
				Labels: map[string]string{
					"label1":  "value1",
					"label2!": "value2",
					"label3":  "value3!",
				},
			},
		},
	}), map[string]string{
		labels.RevisionLabel:        "revision",
		labels.DeploymentUuidLabel:  "00000000-0000-0000-0000-000000000001",
		labels.PrimaryContainerName: "test-00000000-0000-0000-0000-000000000002",
		"label1":                    "value1",
		utils.Hash("label1"):        utils.Hash("value1"),
		utils.Hash("label2!"):       utils.Hash("value2"),
		utils.Hash("label3"):        utils.Hash("value3!"),
	})
}

func TestGetPodSpec(t *testing.T) {
	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{},
		},
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		DNSPolicy:     v1.DNSDefault,
	})

	assert.Equal(t, getPodSpec(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Labels: map[string]string{
					labels.ServiceLaunchConfig: labels.ServicePrimaryLaunchConfig,
				},
				RestartPolicy: &client.RestartPolicy{
					Name: "always",
				},
				PrimaryNetworkId: "1",
				IpcMode:          "host",
				PidMode:          "host",
			},
		},
		Networks: []client.Network{
			{
				Resource: client.Resource{
					Id: "1",
				},
				Kind: hostNetworkingKind,
			},
		},
		NodeName: "node1",
	}), v1.PodSpec{
		RestartPolicy: v1.RestartPolicyNever,
		HostIPC:       true,
		HostNetwork:   true,
		HostPID:       true,
		DNSPolicy:     v1.DNSDefault,
		NodeName:      "node1",
	})
}

func TestSecurityContext(t *testing.T) {
	securityContext := getSecurityContext(client.Container{
		Privileged: true,
		ReadOnly:   true,
		CapAdd: []string{
			"capadd1",
			"capadd2",
		},
		CapDrop: []string{
			"capdrop1",
			"capdrop2",
		},
	})
	assert.Equal(t, *securityContext.Privileged, true)
	assert.Equal(t, *securityContext.ReadOnlyRootFilesystem, true)
	assert.Equal(t, securityContext.Capabilities.Add, []v1.Capability{
		v1.Capability("capadd1"),
		v1.Capability("capadd2"),
	})
	assert.Equal(t, securityContext.Capabilities.Drop, []v1.Capability{
		v1.Capability("capdrop1"),
		v1.Capability("capdrop2"),
	})
}

func TestGetResources(t *testing.T) {
	assert.Equal(t, v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory: resource.MustParse("100000"),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: resource.MustParse("100000"),
			v1.ResourceCPU:    resource.MustParse("50m"),
		},
	}, getResources(client.Container{
		Memory:              100000,
		MemoryReservation:   100000,
		MilliCpuReservation: 50,
	}))

	emptyResources := getResources(client.Container{})
	assert.Equal(t, v1.ResourceList(nil), emptyResources.Limits)
	assert.Equal(t, v1.ResourceList(nil), emptyResources.Requests)
}

func TestGetHostAliases(t *testing.T) {
	assert.Equal(t, []v1.HostAlias{
		{
			IP: "0.0.0.0",
			Hostnames: []string{
				"hostname",
			},
		},
	}, getHostAliases(client.Container{
		ExtraHosts: []string{
			"hostname:0.0.0.0",
		},
	}))
}

func TestGetVolumes(t *testing.T) {
	assert.Equal(t, getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				DataVolumes: []string{
					"/host/path:/container/path",
				},
			},
		},
	}), []v1.Volume{
		{
			Name: utils.Hash("/host/path"),
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/host/path",
				},
			},
		},
	})
	assert.Equal(t, getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Tmpfs: map[string]string{
					"/dir": "true",
				},
			},
		},
	}), []v1.Volume{
		{
			Name: utils.Hash("/dir"),
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMediumMemory,
				},
			},
		},
	})
	assert.Equal(t, len(getVolumes(client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
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
		{
			Name:      utils.Hash("/host/path"),
			MountPath: "/container/path",
		},
	})
	assert.Equal(t, getVolumeMounts(client.Container{
		Tmpfs: map[string]string{
			"/dir": "true",
		},
	}), []v1.VolumeMount{
		{
			Name:      utils.Hash("/dir"),
			MountPath: "/dir",
		},
	})
	assert.Equal(t, len(getVolumeMounts(client.Container{
		DataVolumes: []string{
			"/anonymous/volume",
		},
	})), 0)
}

func TestGetImagePullSecretReferences(t *testing.T) {
	assert.Equal(t, getImagePullSecretReferences(oneRegistryRequest), []v1.LocalObjectReference{
		{
			Name: utils.Hash("https://index.docker.io/v1/" + "username" + "password"),
		},
	})
	assert.Equal(t, getImagePullSecretReferences(twoRegistriesRequest), []v1.LocalObjectReference{
		{
			Name: utils.Hash("https://index.docker.io/v1/" + "username1" + "password1"),
		},
		{
			Name: utils.Hash("https://quay.io" + "username2" + "password2"),
		},
	})
}

func TestGetAffinity(t *testing.T) {
	matchExpressions := getAffinity(client.Container{
		Labels: map[string]string{
			labels.HostAffinityLabel:     "key1=val1,key2=val2",
			labels.HostAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}, "default").NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
	assert.Len(t, matchExpressions, 4)
	for _, nodeSelectorRequirement := range []v1.NodeSelectorRequirement{
		{
			Key:      "key1",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val1",
			},
		},
		{
			Key:      "key2",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val2",
			},
		},
		{
			Key:      "key3",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val3",
			},
		},
		{
			Key:      "key4",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val4",
			},
		},
	} {
		assert.Contains(t, matchExpressions, nodeSelectorRequirement)
	}

	matchExpressions = getAffinity(client.Container{
		Labels: map[string]string{
			labels.HostSoftAffinityLabel:     "key1=val1,key2=val2",
			labels.HostSoftAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}, "default").NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Preference.MatchExpressions
	assert.Len(t, matchExpressions, 4)
	for _, nodeSelectorRequirement := range []v1.NodeSelectorRequirement{
		{
			Key:      "key1",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val1",
			},
		},
		{
			Key:      "key2",
			Operator: v1.NodeSelectorOpIn,
			Values: []string{
				"val2",
			},
		},
		{
			Key:      "key3",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val3",
			},
		},
		{
			Key:      "key4",
			Operator: v1.NodeSelectorOpNotIn,
			Values: []string{
				"val4",
			},
		},
	} {
		assert.Contains(t, matchExpressions, nodeSelectorRequirement)
	}

	podAffinityTerms := getAffinity(client.Container{
		Labels: map[string]string{
			labels.ContainerAffinityLabel:     "key1=val1,key2=val2",
			labels.ContainerAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}, "default").PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	assert.Len(t, podAffinityTerms, 1)
	assert.Equal(t, podAffinityTerms[0].Namespaces, []string{"default"})
	assert.Equal(t, podAffinityTerms[0].TopologyKey, hostnameTopologyKey)
	labelMatchExpressions := podAffinityTerms[0].LabelSelector.MatchExpressions
	assert.Len(t, labelMatchExpressions, 4)
	for _, labelSelectorRequirement := range []metav1.LabelSelectorRequirement{
		{
			Key:      utils.Hash("key1"),
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				utils.Hash("val1"),
			},
		},
		{
			Key:      utils.Hash("key2"),
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				utils.Hash("val2"),
			},
		},
		{
			Key:      utils.Hash("key3"),
			Operator: metav1.LabelSelectorOpNotIn,
			Values: []string{
				utils.Hash("val3"),
			},
		},
		{
			Key:      utils.Hash("key4"),
			Operator: metav1.LabelSelectorOpNotIn,
			Values: []string{
				utils.Hash("val4"),
			},
		},
	} {
		assert.Contains(t, labelMatchExpressions, labelSelectorRequirement)
	}

	weightedPodAffinityTerms := getAffinity(client.Container{
		Labels: map[string]string{
			labels.ContainerSoftAffinityLabel:     "key1=val1,key2=val2",
			labels.ContainerSoftAntiAffinityLabel: "key3=val3,key4=val4",
		},
	}, "default").PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	assert.Len(t, weightedPodAffinityTerms, 1)
	assert.Equal(t, weightedPodAffinityTerms[0].PodAffinityTerm.Namespaces, []string{"default"})
	assert.Equal(t, weightedPodAffinityTerms[0].PodAffinityTerm.TopologyKey, hostnameTopologyKey)
	labelMatchExpressions = weightedPodAffinityTerms[0].PodAffinityTerm.LabelSelector.MatchExpressions
	assert.Len(t, labelMatchExpressions, 4)
	for _, labelSelectorRequirement := range []metav1.LabelSelectorRequirement{
		{
			Key:      utils.Hash("key1"),
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				utils.Hash("val1"),
			},
		},
		{
			Key:      utils.Hash("key2"),
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				utils.Hash("val2"),
			},
		},
		{
			Key:      utils.Hash("key3"),
			Operator: metav1.LabelSelectorOpNotIn,
			Values: []string{
				utils.Hash("val3"),
			},
		},
		{
			Key:      utils.Hash("key4"),
			Operator: metav1.LabelSelectorOpNotIn,
			Values: []string{
				utils.Hash("val4"),
			},
		},
	} {
		assert.Contains(t, labelMatchExpressions, labelSelectorRequirement)
	}
}
