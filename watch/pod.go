package watch

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancherlabs/kattle/labels"
	"github.com/rancherlabs/kattle/publish"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Client) startPodWatch() chan struct{} {
	watchlist := cache.NewListWatchFromClient(c.clientset.Core().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: podFilterAddDelete(c.rancherClient, func(pod v1.Pod) {
				c.pods[pod.Name] = pod
			}),
			DeleteFunc: podFilterAddDelete(c.rancherClient, func(pod v1.Pod) {
				delete(c.pods, pod.Name)
			}),
			UpdateFunc: podFilterUpdate(c.rancherClient, func(pod v1.Pod) {
				c.pods[pod.Name] = pod
			}),
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	return stop
}

func podFilterAddDelete(rancherClient *client.RancherClient, f func(v1.Pod)) func(interface{}) {
	return func(obj interface{}) {
		pod := obj.(*v1.Pod)
		if err := publish.Pod(rancherClient, *pod); err != nil {
			log.Errorf("Failed to publish reply for pod %s: %v", pod.Name, err)
		}
		if _, ok := pod.Labels[labels.RevisionLabel]; ok {
			f(*pod)
		}
	}
}

func podFilterUpdate(rancherClient *client.RancherClient, f func(v1.Pod)) func(interface{}, interface{}) {
	return func(oldObj, newObj interface{}) {
		podFilterAddDelete(rancherClient, f)(newObj)
	}
}
