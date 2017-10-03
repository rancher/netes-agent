package sync

import (
	errs "errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/watch"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	waitTime                 = time.Second
	timeoutDuration          = 5 * time.Minute
	imagePullBackOffReason   = "ImagePullBackOff"
	containerCannotRunReason = "ContainerCannotRun"
)

var (
	errTimeoutDelete = errs.New("Timeout deleting pod")
)

func shouldRemove(deploymentUnit client.DeploymentSyncRequest) bool {
	if len(deploymentUnit.Containers) == 0 {
		return true
	}

	container := deploymentUnit.Containers[0]
	if container.State == "removing" || container.Removed != "" {
		return true
	}

	return false
}

func reconcilePod(clientset *kubernetes.Clientset, watchClient *watch.Client, desiredPod v1.Pod, deploymentUnit client.DeploymentSyncRequest, progressResponder func(string)) (*v1.Pod, error) {
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
				if err := deletePod(clientset, watchClient, namespace, podName, true); err != nil {
					if err == errTimeoutDelete {
						return nil, nil
					}
					return nil, err
				}
			} else {
				break
			}
		} else if err != nil {
			return nil, err
		}
	}
	return waitForPodContainersToBeReady(watchClient, namespace, podName, deploymentUnit, progressResponder)
}

func waitForCacheToContainPod(watchClient *watch.Client, namespace, podName string) (*v1.Pod, bool) {
	for i := 0; i < 3; i++ {
		existingPod, ok := watchClient.GetPod(namespace, podName)
		if ok {
			return &existingPod, true
		}
		time.Sleep(waitTime)
	}
	return nil, false
}

func waitForPodContainersToBeReady(watchClient *watch.Client, namespace, podName string, deploymentUnit client.DeploymentSyncRequest, progressResponder func(string)) (*v1.Pod, error) {
	var statusMessage string
	for i := 0; i < 15; i++ {
		if existingPod, ok := watchClient.GetPod(namespace, podName); ok {
			firstContainer := deploymentUnit.Containers[0]
			for _, container := range existingPod.Status.ContainerStatuses {
				if strings.HasSuffix(container.Name, firstContainer.Uuid) && container.ContainerID != "" {
					return &existingPod, nil
				}
			}

			currentStatusMessage, err := getPodStatusMessage(existingPod, firstContainer.Uuid)
			if err != nil {
				return nil, err
			}

			if currentStatusMessage != statusMessage {
				progressResponder(currentStatusMessage)
				statusMessage = currentStatusMessage
			}
		}
		log.Infof("Waiting for containers of pod %s to be ready", podName)
		time.Sleep(waitTime)
	}
	return nil, nil
}

func getPodStatusMessage(pod v1.Pod, uuid string) (string, error) {
	for _, condition := range pod.Status.Conditions {
		if condition.Status != "False" {
			continue
		}
		message := condition.Message
		if condition.Type == v1.PodReady {
			statusMessage, err := getAllContainerStatusMessage(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses)
			if err != nil {
				return "", err
			}
			if statusMessage != "" {
				message = fmt.Sprintf("%s (%s)", message, statusMessage)
			}
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil {
			msg := status.State.Waiting.Reason
			if status.State.Waiting.Message != "" {
				msg = fmt.Sprintf("%s: %s", msg, status.State.Waiting.Message)
			}
			return msg, nil
		}

	}
	return "Waiting on pod " + pod.Name, nil
}

func getAllContainerStatusMessage(containerStatuses, initContainerStatuses []v1.ContainerStatus) (string, error) {
	initContainerStatusMessages, err := getContainerStatusMessages(initContainerStatuses)
	if err != nil {
		return "", err
	}
	if len(initContainerStatusMessages) > 0 {
		return strings.Join(initContainerStatusMessages, ","), nil
	}
	containerStatusMessages, err := getContainerStatusMessages(containerStatuses)
	if err != nil {
		return "", err
	}
	if len(containerStatusMessages) == 0 {
		return "", nil
	}
	return strings.Join(containerStatusMessages, ","), nil
}

func getContainerStatusMessages(containerStatuses []v1.ContainerStatus) ([]string, error) {
	var containerStatusMessages []string
	for _, containerStatus := range containerStatuses {
		if containerStatus.Ready {
			continue
		}
		if containerStatus.State.Waiting != nil {
			message := containerStatus.State.Waiting.Message
			if containerStatus.State.Waiting.Reason == imagePullBackOffReason {
				return nil, fmt.Errorf("%s: %v", imagePullBackOffReason, message)
			}
			if message != "" {
				containerStatusMessages = append(containerStatusMessages, message)
			}
		} else if containerStatus.State.Terminated != nil {
			message := containerStatus.State.Terminated.Message
			if containerStatus.State.Terminated.Reason == containerCannotRunReason {
				return nil, fmt.Errorf("%s: %v", containerCannotRunReason, message)
			}
			if message != "" {
				containerStatusMessages = append(containerStatusMessages, message)
			}
		}
	}
	return containerStatusMessages, nil
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

func deletePod(clientset *kubernetes.Clientset, watchClient *watch.Client, namespace, podName string, full bool) error {
	log.Infof("Deleting pod %s", podName)
	if err := clientset.Pods(namespace).Delete(podName, &metav1.DeleteOptions{}); errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	for i := 0; i < 15; i++ {
		pod, ok := watchClient.GetPod(namespace, podName)
		if !ok {
			return nil
		}
		if !full {
			allTerminated := true
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Terminated == nil {
					allTerminated = false
				}
			}
			if allTerminated {
				log.Infof("Waiting for pod %s to be deleted: all containers stopped", podName)
				return nil
			}
		}

		log.Infof("Waiting for pod %s to be deleted", podName)
		time.Sleep(waitTime)
	}

	return errTimeoutDelete
}
