package watch

import (
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

type Client struct {
	clientset *kubernetes.Clientset
	pods      map[string]v1.Pod
	podsMutex sync.RWMutex
}

func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset,
		make(map[string]v1.Pod),
		sync.RWMutex{},
	}
}

func (c *Client) Start() {
	c.startPodWatch()
	//c.startPvWatch()
	//c.startPvcWatch()
}

func (c *Client) GetPod(podName string) (v1.Pod, bool) {
	c.podsMutex.RLock()
	pod, ok := c.pods[podName]
	c.podsMutex.RUnlock()
	return pod, ok
}
