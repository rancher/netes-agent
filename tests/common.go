package integration

import (
	"encoding/json"
	"fmt"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/manager"
	"github.com/rancher/netes-agent/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"path"
	"strings"
	"testing"
)

const (
	kubernetesAddress = "0.0.0.0:8080"
)

var (
	clientset      *kubernetes.Clientset
	testManager    *manager.Manager
	deploymentUUID = 0
)

func init() {
	var err error
	clientset, err = kubernetes.NewForConfig(&rest.Config{
		Host: kubernetesAddress,
	})
	if err != nil {
		panic(err)
	}

	testManager = manager.New(nil)
	testManager.SyncClusters([]client.Cluster{
		{
			Resource: client.Resource{
				Id: "1c1",
			},
			K8sClientConfig: &client.K8sClientConfig{
				Address: kubernetesAddress,
			},
		},
	})
}

func nextDeploymentUUID() string {
	deploymentUUID++
	return fmt.Sprintf("%08d", deploymentUUID)
}

func simulateEvent(t *testing.T, event events.Event, deploymentUUID string) (*client.Publish, v1.Pod) {
	// Save namespace and container UUIDS for later testing
	var namespace string
	var containerUuids []string
	modifyEvent(event, func(request *client.DeploymentSyncRequest, _ *client.Container) {
		namespace = request.Namespace
		request.DeploymentUnitUuid = deploymentUUID
		for _, container := range request.Containers {
			containerUuids = append(containerUuids, container.Uuid)
		}
	})

	response, err := testManager.HandleComputeInstanceActivate(&event)
	assert.NoError(t, err)

	var deploymentSyncResponse client.DeploymentSyncResponse
	assert.NoError(t, utils.ConvertByJSON(response.Data["deploymentSyncResponse"], &deploymentSyncResponse))

	// ExternalId should not be empty
	assert.NotEmpty(t, deploymentSyncResponse.ExternalId)
	// NodeName should be reported correctly
	assert.Equal(t, "testhost", deploymentSyncResponse.NodeName)
	// Ensure instance statuses contain all container UUIDs from the request
	assert.Len(t, deploymentSyncResponse.InstanceStatus, len(containerUuids))
	for _, instanceStatus := range deploymentSyncResponse.InstanceStatus {
		assert.NotEmpty(t, instanceStatus.InstanceUuid)
		assert.Contains(t, containerUuids, instanceStatus.InstanceUuid)
	}

	// Lookup the created pod based on ExternalId
	pod, err := clientset.Pods(namespace).Get(deploymentSyncResponse.ExternalId, metav1.GetOptions{})
	assert.NoError(t, err)

	// Ensure the IP was reported correctly
	for _, instanceStatus := range deploymentSyncResponse.InstanceStatus {
		assert.Equal(t, instanceStatus.PrimaryIpAddress, pod.Status.PodIP)
	}

	return response, *pod
}

func getNonPauseContainer(t *testing.T, pod v1.Pod) v1.Container {
	for _, container := range pod.Spec.Containers {
		if !strings.Contains(container.Image, "pause") {
			return container
		}
	}
	t.Fail()
	return v1.Container{}
}

func getEvent(name string) events.Event {
	contents, err := ioutil.ReadFile(path.Join("./events", name+".json"))
	if err != nil {
		panic(err)
	}

	var event events.Event
	err = json.Unmarshal(contents, &event)
	if err != nil {
		panic(err)
	}
	return event
}

func modifyEvent(event events.Event, f func(request *client.DeploymentSyncRequest, container *client.Container)) events.Event {
	var request client.DeploymentSyncRequest
	if err := utils.ConvertByJSON(event.Data["deploymentSyncRequest"], &request); err != nil {
		panic(err)
	}

	f(&request, &request.Containers[0])
	event.Data["deploymentSyncRequest"] = request

	return event
}
