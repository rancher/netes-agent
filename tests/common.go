package integration

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/manager"
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
	kubernetesAddress = "http://0.0.0.0:8080"
)

var (
	clientset      *kubernetes.Clientset
	testManager    *manager.Manager
	deploymentUuid = 0
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
			K8sClientConfig: &client.K8sClientConfig{
				Address: kubernetesAddress,
			},
		},
	})
}

func nextDeploymentUuid() string {
	deploymentUuid += 1
	return fmt.Sprint(deploymentUuid)
}

func simulateEvent(t *testing.T, event events.Event, deploymentUuid string) (*client.Publish, v1.Pod) {
	modifyEvent(event, func(request *client.DeploymentSyncRequest, _ *client.Container) {
		request.DeploymentUnitUuid = deploymentUuid
	})

	response, err := testManager.HandleComputeInstanceActivate(&event)
	assert.NoError(t, err)

	pod, err := clientset.Pods(v1.NamespaceDefault).Get(deploymentUuid, metav1.GetOptions{})
	assert.NoError(t, err)

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
	if err := mapstructure.Decode(event.Data["deploymentSyncRequest"], &request); err != nil {
		panic(err)
	}

	f(&request, &request.Containers[0])
	event.Data["deploymentSyncRequest"] = request

	return event
}
