package sync

import (
	log "github.com/Sirupsen/logrus"
	"github.com/rancherlabs/kattle/labels"
	"github.com/rancherlabs/kattle/watch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func reconcilePods(clientset *kubernetes.Clientset, watchClient *watch.Client, pods []v1.Pod) error {
	for _, pod := range pods {
		go func(pod v1.Pod) {
			revision := pod.Labels[labels.RevisionLabel]
			existingPod, ok := watchClient.Pods()[pod.Name]
			if ok {
				if existingRevision, ok := existingPod.Labels[labels.RevisionLabel]; ok {
					if revision != existingRevision {
						log.Infof("Pod %s has old revision", pod.Name)
						if err := deletePod(clientset, pod); err != nil {
							log.Error(err)
						}
					}
				}
			} else {
				if err := createPod(clientset, pod); err != nil {
					log.Error(err)
				}
			}
		}(pod)
	}

	podNames := map[string]bool{}
	for _, pod := range pods {
		podNames[pod.Name] = true
	}

	for _, pod := range watchClient.Pods() {
		func(pod v1.Pod) {
			if _, ok := podNames[pod.Name]; !ok {
				log.Infof("Pod %s shouldn't exist", pod.Name)
				if err := deletePod(clientset, pod); err != nil {
					log.Error(err)
				}
			}
		}(pod)
	}

	return nil
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Creating pod %s", pod.Name)
	_, err := clientset.Pods(v1.NamespaceDefault).Create(&pod)
	return err
}

func deletePod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Deleting pod %s", pod.Name)
	return clientset.Pods(v1.NamespaceDefault).Delete(pod.Name, &metav1.DeleteOptions{})
}
