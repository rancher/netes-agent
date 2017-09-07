package sync

import (
	"time"

	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	waitTime = time.Second
)

func reconcilePod(clientset *kubernetes.Clientset, watchClient *watch.Client, desiredPod v1.Pod) (v1.Pod, error) {
	podName := desiredPod.Name
	namespace := desiredPod.Namespace
	desiredRevision := desiredPod.Labels[labels.RevisionLabel]
	for i := 0; i < 5; i++ {
		if err := createPod(clientset, desiredPod); err == nil {
			break
		} else if errors.IsAlreadyExists(err) {
			existingPod, ok := waitForCacheToContainPod(watchClient, namespace, podName)
			if !ok {
				continue
			}
			if existingPod.Labels[labels.RevisionLabel] != desiredRevision {
				log.Infof("Pod %s has old revision", podName)
				if err := deletePod(clientset, watchClient, namespace, podName); err != nil {
					return v1.Pod{}, err
				}
			}
		} else if err != nil {
			return v1.Pod{}, err
		}
	}
	return waitForPodContainersToBeReady(watchClient, namespace, podName)
}

func waitForCacheToContainPod(watchClient *watch.Client, namespace, podName string) (v1.Pod, bool) {
	for i := 0; i < 3; i++ {
		existingPod, ok := watchClient.GetPod(namespace, podName)
		if ok {
			return existingPod, true
		}
		time.Sleep(waitTime)
	}
	return v1.Pod{}, false
}

func waitForPodContainersToBeReady(watchClient *watch.Client, namespace, podName string) (v1.Pod, error) {
	for i := 0; i < 45; i++ {
		if existingPod, ok := watchClient.GetPod(namespace, podName); ok && existingPod.Spec.NodeName != "" {
			if podContainersReady(existingPod) {
				return existingPod, nil
			}
		}
		log.Infof("Waiting for containers of pod %s to be ready", podName)
		time.Sleep(waitTime)
	}
	return v1.Pod{}, fmt.Errorf("Timeout waiting for pod %s", podName)
}

func podContainersReady(pod v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false
		}
	}
	return true
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	if err := ensureNamespaceExists(clientset, pod.Namespace); err != nil {
		return err
	}
	log.Infof("Creating pod %s", pod.Name)
	_, err := clientset.Pods(pod.Namespace).Create(&pod)
	return err
}

func ensureNamespaceExists(clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.Namespaces().Get(namespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Infof("Creating namepace %s", namespace)
		if _, err = clientset.Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}); err != nil {
			return err
		}
	}
	return err
}

func deletePod(clientset *kubernetes.Clientset, watchClient *watch.Client, namespace, podName string) error {
	log.Infof("Deleting pod %s", podName)
	if err := clientset.Pods(namespace).Delete(podName, &metav1.DeleteOptions{}); errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	for {
		if _, ok := watchClient.GetPod(namespace, podName); !ok {
			return nil
		}
		log.Infof("Waiting for pod %s to be deleted", podName)
		time.Sleep(waitTime)
	}
}
