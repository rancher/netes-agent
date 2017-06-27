package sync

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher-metadata/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func Sync(clientset *kubernetes.Clientset, deploymentUnits []metadata.DeploymentUnit, volumes []metadata.Volume) error {
	var pods []v1.Pod
	for _, deploymentUnit := range deploymentUnits {
		pods = append(pods, PodFromDeploymentUnit(deploymentUnit))
	}

	volumeIds := map[string]bool{}
	for _, deploymentUnit := range deploymentUnits {
		for _, container := range deploymentUnit.Containers {
			for _, mount := range container.Mounts {
				volumeIds[mount.VolumeId] = true
			}
		}
	}

	if err := reconcileVolumes(clientset, volumes, volumeIds); err != nil {
		return err
	}

	return reconcilePods(clientset, pods)
}

func reconcileVolumes(clientset *kubernetes.Clientset, volumes []metadata.Volume, volumesIds map[string]bool) error {
	for _, volume := range volumes {
		if _, ok := volumesIds[volume.Id]; !ok {
			continue
		}

		go func(volume metadata.Volume) {
			// TODO: remove these hard-coded values
			volume.Metadata = map[string]interface{}{
				"accessModes": []string{
					"ReadWriteOnce",
				},
				"size": "8Gi",
				"nfs": map[string]interface{}{
					"server": "0.0.0.0",
					"path":   "/",
				},
			}

			pv := PvFromVolume(volume)
			_, err := clientset.PersistentVolumes().Get(pv.Name, metav1.GetOptions{})
			if err != nil {
				if err := createPv(clientset, pv); err != nil {
					log.Error(err)
				}
			}

			pvc := PvcFromVolume(volume)
			_, err = clientset.PersistentVolumeClaims("default").Get(pvc.Name, metav1.GetOptions{})
			if err != nil {
				if err := createPvc(clientset, pvc); err != nil {
					log.Error(err)
				}
			}
		}(volume)
	}
	return nil
}

func createPv(clientset *kubernetes.Clientset, pv v1.PersistentVolume) error {
	log.Infof("Creating PV %s", pv.Name)
	_, err := clientset.PersistentVolumes().Create(&pv)
	return err
}

func createPvc(clientset *kubernetes.Clientset, pvc v1.PersistentVolumeClaim) error {
	log.Infof("Creating PVC %s", pvc.Name)
	_, err := clientset.PersistentVolumeClaims("default").Create(&pvc)
	return err
}

func reconcilePods(clientset *kubernetes.Clientset, pods []v1.Pod) error {
	for _, pod := range pods {
		go func(pod v1.Pod) {
			revision := pod.Labels[revisionLabel]

			existingPod, err := clientset.Pods("default").Get(pod.Name, metav1.GetOptions{})
			if err != nil {
				if err = createPod(clientset, pod); err != nil {
					log.Error(err)
					return
				}
				return
			}

			if existingRevision, ok := existingPod.Labels[revisionLabel]; ok {
				if revision != existingRevision {
					log.Debugf("DELETE1 %s", pod.Name)
					if err = deletePod(clientset, pod); err != nil {
						log.Error(err)
						return
					}
					if err = createPod(clientset, pod); err != nil {
						log.Error(err)
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

	// TODO: use a watch for this
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
				log.Debugf("DELETE2 %s", pod.Name)
				if err = deletePod(clientset, pod); err != nil {
					log.Error(err)
					return
				}
			}
		}(pod)
	}

	return nil
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Creating pod %s", pod.Name)
	_, err := clientset.Pods("default").Create(&pod)
	return err
}

func deletePod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Deleting pod %s", pod.Name)
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
