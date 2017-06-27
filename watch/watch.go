package watch

import (
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	PodCache map[string]v1.Pod
)

func Pods(clientset *kubernetes.Clientset) chan struct{} {
	PodCache = map[string]v1.Pod{}

	watchlist := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "pods", v1.NamespaceDefault,
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				PodCache[pod.Name] = *pod
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				delete(PodCache, pod.Name)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod := newObj.(*v1.Pod)
				PodCache[pod.Name] = *pod
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	return stop
}
