package sync

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func reconcilePod(clientset *kubernetes.Clientset, watchClient *watch.Client, pod v1.Pod) (v1.Pod, error) {
	revision := pod.Labels[labels.RevisionLabel]
	existingPod, ok := watchClient.GetPod(pod.Namespace, pod.Name)
	if ok {
		if existingRevision, ok := existingPod.Labels[labels.RevisionLabel]; ok {
			if revision != existingRevision {
				log.Infof("Pod %s has old revision", pod.Name)
				if err := deletePod(clientset, watchClient, pod.Namespace, pod.Name); err != nil {
					return v1.Pod{}, err
				}
			}
		} else {
			return existingPod, nil
		}
	}

	if err := createPod(clientset, pod); err != nil {
		return v1.Pod{}, err
	}

	for {
		if existingPod, ok := watchClient.GetPod(pod.Namespace, pod.Name); ok && existingPod.Spec.NodeName != "" {
			allContainersReady := true
			for _, containerStatus := range existingPod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					allContainersReady = false
					break
				}
			}
			if allContainersReady {
				return existingPod, nil
			}
		}
		log.Infof("Waiting for containers of pod %s to be ready", pod.Name)
		time.Sleep(time.Second)
	}
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	_, err := clientset.Namespaces().Get(pod.Namespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Infof("Creating namepace %s", pod.Namespace)
		clientset.Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: pod.Namespace,
			},
		})
	} else if err != nil {
		return err
	}

	log.Infof("Creating pod %s", pod.Name)
	_, err = clientset.Pods(pod.Namespace).Create(&pod)
	return err
}

func deletePod(clientset *kubernetes.Clientset, watchClient *watch.Client, namespace, podName string) error {
	log.Infof("Deleting pod %s", podName)
	err := clientset.Pods(namespace).Delete(podName, &metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	for {
		if _, ok := watchClient.GetPod(namespace, podName); !ok {
			return nil
		}
		log.Infof("Waiting for pod %s to be deleted", podName)
		time.Sleep(time.Second)
	}
}
