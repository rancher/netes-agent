package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/pkg/api/v1"
)

func TestBasic(t *testing.T) {
	simulateEvent(t, basicEvent, nextDeploymentUuid())
}

func TestPodSpec(t *testing.T) {
	_, pod := simulateEvent(t, podSpecEvent, nextDeploymentUuid())
	assert.True(t, len(pod.Status.ContainerStatuses) > 0)

	assert.Equal(t, pod.Spec.RestartPolicy, v1.RestartPolicyNever)
	assert.True(t, pod.Spec.HostIPC)
	assert.True(t, pod.Spec.HostPID)
	assert.Equal(t, pod.Spec.DNSPolicy, v1.DNSDefault)
}

func TestContainerSpec(t *testing.T) {
	_, pod := simulateEvent(t, containerSpecEvent, nextDeploymentUuid())
	container := getNonPauseContainer(t, pod)

	assert.Equal(t, container.TTY, true)
	assert.Equal(t, container.Stdin, true)
	assert.Equal(t, container.WorkingDir, "/usr")
	assert.Equal(t, container.Env[0].Name, "TEST")
	assert.Equal(t, container.Env[0].Value, "true")
}

func TestSecurityContext(t *testing.T) {
	_, pod := simulateEvent(t, securityContextEvent, nextDeploymentUuid())
	container := getNonPauseContainer(t, pod)

	assert.Equal(t, fmt.Sprint(container.SecurityContext.Capabilities.Add[0]), "SYS_NICE")
}

func TestUpgrade(t *testing.T) {
	_, pod := simulateEvent(t, securityContextEvent, nextDeploymentUuid())
	container := getNonPauseContainer(t, pod)

	assert.Equal(t, container.Image, "nginx")

	_, pod = simulateEvent(t, upgradeEvent, pod.Name)
	container = getNonPauseContainer(t, pod)

	assert.Equal(t, container.Image, "nginx:1.13")
}

func TestAlternateNamespace(t *testing.T) {
	_, pod := simulateEvent(t, alternateNamespaceEvent, nextDeploymentUuid())
	assert.Equal(t, pod.Namespace, "testnamespace")
}
