package sync

import (
	"time"

	"k8s.io/client-go/pkg/api/v1"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/kattle/types"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
	"k8s.io/client-go/kubernetes"
)

func Sync(m metadata.Client, rancherClient *client.RancherClient, clientset *kubernetes.Clientset) error {
	/*containers, err := m.GetContainers()
	if err != nil {
		return err
	}*/

	deploymentUnits := map[string]*types.DeploymentUnit{}

	/*for _, container := range containers {
	if _, ok := container.Labels["kattle"]; !ok {
		continue
	}

	// TODO: only list containers with label
	apiContainers, err := rancherClient.Container.List(&client.ListOpts{})
	if err != nil {
		return err
	}

	for _, apiContainer := range apiContainers.Data {
		if apiContainer.Uuid == container.UUID {
			dus2, err := rancherClient.DeploymentUnit.List(&client.ListOpts{})
			if err != nil {
				return err
			}

			var dus []client.DeploymentUnit
			for _, du2 := range dus2.Data {
				if du2.Uuid == apiContainer.DeploymentUnitUuid {
					dus = append(dus, du2)
				}
			}
			if len(dus) == 0 {
				continue
			}

			du := dus[0]

			fmt.Println("x", du)
			if deploymentUnit, ok := deploymentUnits[du.Uuid]; ok {
				deploymentUnit.Containers = append(deploymentUnit.Containers, &apiContainer)
			} else {
				deploymentUnits[du.Uuid] = &types.DeploymentUnit{
					Uuid:       du.Uuid,
					RevisionId: du.RevisionId,
					Containers: []*client.Container{
						&apiContainer,
					},
				}
			}
		}
	}*/

	apiContainers, err := rancherClient.Container.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"state": "running",
		},
	})
	if err != nil {
		return err
	}

	for _, apiContainer := range apiContainers.Data {
		if _, ok := apiContainer.Labels[kattleLabel]; !ok {
			continue
		}

		du, err := rancherClient.DeploymentUnit.ById(apiContainer.DeploymentUnitId)
		if err != nil {
			return err
		}

		if deploymentUnit, ok := deploymentUnits[du.Id]; ok {
			deploymentUnit.Containers = append(deploymentUnit.Containers, apiContainer)
		} else {
			deploymentUnits[du.Id] = &types.DeploymentUnit{
				Id:                  du.Id,
				Uuid:                du.Uuid,
				HostId:              du.HostId,
				RevisionId:          du.RevisionId,
				RequestedRevisionId: du.RequestedRevisionId,
				Containers: []client.Container{
					apiContainer,
				},
			}
		}
	}

	var pods []v1.Pod
	for _, deploymentUnit := range deploymentUnits {
		revision, err := rancherClient.Revision.ById(deploymentUnit.RevisionId)
		if err != nil {
			return err
		}
		deploymentUnit.RevisionConfig = *revision.Config
		pods = append(pods, PodFromDeploymentUnit(*deploymentUnit))
	}

	return reconsilePods(clientset, pods)
}

func reconsilePods(clientset *kubernetes.Clientset, pods []v1.Pod) error {
	for _, pod := range pods {
		go func(pod v1.Pod) {
			revision := pod.Labels[revisionLabel]

			existingPod, err := clientset.Pods("default").Get(pod.Name)
			if err != nil {
				if err = createPod(clientset, pod); err != nil {
					//return err
					logrus.Error(err)
				}
				//continue
				return
			}

			if existingRevision, ok := existingPod.Labels[revisionLabel]; ok {
				if revision != existingRevision {
					logrus.Info("DELETE1")
					if err = deletePod(clientset, pod); err != nil {
						//return err
						logrus.Error(err)
					}
					if err = createPod(clientset, pod); err != nil {
						//return err
						logrus.Error(err)
					}
				}
			}

		}(pod)
		/*revision := pod.Labels[revisionLabel]

		existingPod, err := clientset.Pods("default").Get(pod.Name)
		if err != nil {
			if err = createPod(clientset, pod); err != nil {
				return err
			}
			continue
		}

		if existingRevision, ok := existingPod.Labels[revisionLabel]; ok {
			if revision != existingRevision {
				logrus.Info("DELETE1")
				if err = deletePod(clientset, pod); err != nil {
					return err
				}
				if err = createPod(clientset, pod); err != nil {
					return err
				}
			}
		}*/
	}

	podNames := map[string]bool{}
	for _, pod := range pods {
		podNames[pod.Name] = true
	}

	existingPods, err := clientset.Pods("default").List(v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range existingPods.Items {
		go func(pod v1.Pod) {
			if _, ok := pod.Labels[kattleLabel]; !ok {
				//continue
				return
			}
			if _, ok := podNames[pod.Name]; !ok {
				logrus.Info("DELETE2")
				if err = deletePod(clientset, pod); err != nil {
					//return err
					logrus.Error(err)
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
	if err := clientset.Pods("default").Delete(pod.Name, &v1.DeleteOptions{}); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		if _, err := clientset.Pods("default").Get(pod.Name); err != nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}
