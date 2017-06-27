package watch

import (
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	PvCache map[string]v1.PersistentVolume
)

func Pvs(clientset *kubernetes.Clientset) chan struct{} {
	PvCache = map[string]v1.PersistentVolume{}

	watchlist := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "persistentvolumes", "", fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.PersistentVolume{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: pvFilterAddDelete(func(pv v1.PersistentVolume) {
				PvCache[pv.Name] = pv
			}),
			DeleteFunc: pvFilterAddDelete(func(pv v1.PersistentVolume) {
				delete(PvCache, pv.Name)
			}),
			UpdateFunc: pvFilterUpdate(func(pv v1.PersistentVolume) {
				PvCache[pv.Name] = pv
			}),
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	return stop
}

func pvFilterAddDelete(f func(v1.PersistentVolume)) func(interface{}) {
	return func(obj interface{}) {
		pv := obj.(*v1.PersistentVolume)
		f(*pv)
	}
}

func pvFilterUpdate(f func(v1.PersistentVolume)) func(interface{}, interface{}) {
	return func(oldObj, newObj interface{}) {
		pvFilterAddDelete(f)(newObj)
	}
}
