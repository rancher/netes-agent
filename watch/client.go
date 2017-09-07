package watch

import (
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

type Client struct {
	clientset     *kubernetes.Clientset
	pods          map[string]map[string]v1.Pod
	podsMutex     sync.RWMutex
	podsWatchChan chan struct{}
}

func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset,
		make(map[string]map[string]v1.Pod),
		sync.RWMutex{},
		nil,
	}
}

func (c *Client) Start() {
	c.podsWatchChan = c.startPodWatch()
}

func (c *Client) Stop() {
	if c.podsWatchChan != nil {
		c.podsWatchChan <- struct{}{}
	}
}

func (c *Client) GetPod(namespace, podName string) (v1.Pod, bool) {
	c.podsMutex.RLock()
	defer c.podsMutex.RUnlock()
	namespacePods, ok := c.pods[namespace]
	if !ok {
		return v1.Pod{}, false
	}
	pod, ok := namespacePods[podName]
	return pod, ok
}

func (c *Client) addPod(pod v1.Pod) {
	c.podsMutex.Lock()
	defer c.podsMutex.Unlock()
	if _, ok := c.pods[pod.Namespace]; !ok {
		c.pods[pod.Namespace] = make(map[string]v1.Pod)
	}
	c.pods[pod.Namespace][pod.Name] = pod
}

func (c *Client) deletePod(pod v1.Pod) {
	c.podsMutex.Lock()
	defer c.podsMutex.Unlock()
	if _, ok := c.pods[pod.Namespace]; !ok {
		return
	}
	delete(c.pods[pod.Namespace], pod.Name)
}
