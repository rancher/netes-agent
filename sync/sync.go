package sync

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func Sync(clientset *kubernetes.Clientset, deploymentUnits []metadata.DeploymentUnit, volumes []client.Volume, volumeTemplates []client.VolumeTemplate, storageDrivers []client.StorageDriver) error {
	var pods []v1.Pod
	for _, deploymentUnit := range deploymentUnits {
		pods = append(pods, PodFromDeploymentUnit(deploymentUnit))
	}
	return reconsilePods(clientset, pods)
}

func reconsilePods(clientset *kubernetes.Clientset, pods []v1.Pod) error {
	for _, pod := range pods {
		go func(pod v1.Pod) {
			revision := pod.Labels[revisionLabel]

			existingPod, err := clientset.Pods("default").Get(pod.Name, metav1.GetOptions{})
			if err != nil {
				if err = createPod(clientset, pod); err != nil {
					logrus.Error(err)
					return
				}
				return
			}

			if existingRevision, ok := existingPod.Labels[revisionLabel]; ok {
				if revision != existingRevision {
					logrus.Info("DELETE1")
					if err = deletePod(clientset, pod); err != nil {
						logrus.Error(err)
						return
					}
					if err = createPod(clientset, pod); err != nil {
						logrus.Error(err)
						return
					}
				}
			}
		}(pod)
	}

	podNames := map[string]bool{}
	for _, pod := range pods {
		podNames[pod.Name] = true
	}

	existingPods, err := clientset.Pods("default").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range existingPods.Items {
		go func(pod v1.Pod) {
			if _, ok := pod.Labels[revisionLabel]; !ok {
				return
			}
			if _, ok := podNames[pod.Name]; !ok {
				logrus.Info("DELETE2")
				if err = deletePod(clientset, pod); err != nil {
					logrus.Error(err)
					return
				}
			}
		}(pod)
	}

	return nil
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	logrus.Infof("Creating %s", pod.Name)
	_, err := clientset.Pods("default").Create(&pod)
	return err
}

func deletePod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	logrus.Infof("Deleting %s", pod.Name)
	if err := clientset.Pods("default").Delete(pod.Name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		if _, err := clientset.Pods("default").Get(pod.Name, metav1.GetOptions{}); err != nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}
