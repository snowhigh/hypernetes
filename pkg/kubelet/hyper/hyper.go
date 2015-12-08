/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hyper

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/credentialprovider"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/network"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/probe"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util"
)

const (
	hyperBinName             = "hyper"
	hyperContainerNamePrefix = "kube"
	hyperPodNamePrefix       = "kube"
	hyperBaseMemory          = 64
	hyperDefaultContainerCPU = 1
	hyperDefaultContainerMem = 128
	hyperPodSpecDir          = "/var/lib/kubelet/hyper"
)

// runtime implements the container runtime for hyper
type runtime struct {
	hyperBinAbsPath     string
	dockerKeyring       credentialprovider.DockerKeyring
	containerRefManager *kubecontainer.RefManager
	generator           kubecontainer.RunContainerOptionsGenerator
	recorder            record.EventRecorder
	prober              prober.Prober
	networkPlugin       network.NetworkPlugin
	readinessManager    *kubecontainer.ReadinessManager
	volumeGetter        volumeGetter
	hyperClient         *HyperClient
	kubeClient          client.Interface
	imagePuller         kubecontainer.ImagePuller
}

var _ kubecontainer.Runtime = &runtime{}

type volumeGetter interface {
	GetVolumes(podUID types.UID) (kubecontainer.VolumeMap, bool)
}

// New creates the hyper container runtime which implements the container runtime interface.
func New(generator kubecontainer.RunContainerOptionsGenerator,
	recorder record.EventRecorder,
	networkPlugin network.NetworkPlugin,
	containerRefManager *kubecontainer.RefManager,
	readinessManager *kubecontainer.ReadinessManager,
	volumeGetter volumeGetter,
	kubeClient client.Interface) (kubecontainer.Runtime, error) {

	// check hyper has already installed
	hyperBinAbsPath, err := exec.LookPath(hyperBinName)
	if err != nil {
		glog.Errorf("Hyper: can't find hyper binary")
		return nil, fmt.Errorf("cannot find hyper binary: %v", err)
	}

	hyper := &runtime{
		hyperBinAbsPath:     hyperBinAbsPath,
		dockerKeyring:       credentialprovider.NewDockerKeyring(),
		containerRefManager: containerRefManager,
		generator:           generator,
		recorder:            recorder,
		networkPlugin:       networkPlugin,
		readinessManager:    readinessManager,
		volumeGetter:        volumeGetter,
		hyperClient:         NewHyperClient(),
		kubeClient:          kubeClient,
	}
	hyper.prober = prober.New(hyper, readinessManager, containerRefManager, recorder)
	hyper.imagePuller = kubecontainer.NewImagePuller(recorder, hyper)

	return hyper, nil
}

func (r *runtime) buildCommand(args ...string) *exec.Cmd {
	hyperBinAbsPath, err := exec.LookPath(hyperBinName)
	if err != nil {
		return nil
	}

	cmd := exec.Command(hyperBinAbsPath)
	cmd.Args = append(cmd.Args, args...)
	return cmd
}

// runCommand invokes hyper binary with arguments and returns the result
// from stdout in a list of strings. Each string in the list is a line.
func (r *runtime) runCommand(args ...string) ([]string, error) {
	output, err := r.buildCommand(args...).Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// Version invokes 'hyper version' to get the version information of the hyper
// runtime on the machine.
// The return values are an int array containers the version number.
func (r *runtime) Version() (kubecontainer.Version, error) {
	version, err := r.hyperClient.Version()
	if err != nil {
		return nil, err
	}

	return parseVersion(version)
}

func parseTimeString(str string) (time.Time, error) {
	t := time.Date(0, 0, 0, 0, 0, 0, 0, time.Local)
	if str == "" {
		return t, nil
	}

	layout := "2006-01-02T15:04:05Z"
	t, err := time.Parse(layout, str)
	if err != nil {
		return t, err
	}

	return t, nil
}

func (r *runtime) getContainerStatus(container ContainerStatus, image, imageID string) api.ContainerStatus {
	var status api.ContainerStatus

	_, _, _, containerName, err := r.parseHyperContainerFullName(container.Name)
	if err != nil {
		return status
	}

	status.Name = strings.Split(containerName, ".")[0]
	status.ContainerID = container.ContainerID
	status.Image = image
	status.ImageID = imageID

	switch container.Phase {
	case StatusRunning:
		runningStartedAt, err := parseTimeString(container.Running.StartedAt)
		if err != nil {
			glog.Errorf("Hyper: can't parse runningStartedAt %s", container.Running.StartedAt)
			return status
		}

		status.State = api.ContainerState{
			Running: &api.ContainerStateRunning{
				StartedAt: unversioned.NewTime(runningStartedAt),
			},
		}
	case StatusPending:
		status.State = api.ContainerState{
			Waiting: &api.ContainerStateWaiting{
				Reason: container.Waiting.Reason,
			},
		}
	case StatusFailed, StatusSuccess:
		terminatedStartedAt, err := parseTimeString(container.Terminated.StartedAt)
		if err != nil {
			glog.Errorf("Hyper: can't parse terminatedStartedAt %s", container.Terminated.StartedAt)
			return status
		}

		terminatedFinishedAt, err := parseTimeString(container.Terminated.FinishedAt)
		if err != nil {
			glog.Errorf("Hyper: can't parse terminatedFinishedAt %s", container.Terminated.FinishedAt)
			return status
		}

		status.State = api.ContainerState{
			Terminated: &api.ContainerStateTerminated{
				ExitCode:   container.Terminated.ExitCode,
				Reason:     container.Terminated.Reason,
				Message:    container.Terminated.Message,
				StartedAt:  unversioned.NewTime(terminatedStartedAt),
				FinishedAt: unversioned.NewTime(terminatedFinishedAt),
			},
		}
	default:
		glog.Warningf("Hyper: Unknown pod state: %q", container.Phase)
	}

	return status
}

func (r *runtime) buildHyperPodFullName(uid, name, namespace string) string {
	return fmt.Sprintf("%s_%s_%s_%s", hyperPodNamePrefix, uid, name, namespace)
}

func (r *runtime) buildHyperContainerFullName(uid, podName, namespace, containerName string, container api.Container) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%08x",
		hyperContainerNamePrefix,
		uid,
		podName,
		namespace,
		containerName+"."+strconv.FormatUint(kubecontainer.HashContainer(&container), 16),
		rand.Uint32())
}

func (r *runtime) parseHyperPodFullName(podFullName string) (string, string, string, error) {
	parts := strings.Split(podFullName, "_")
	if len(parts) != 4 {
		return "", "", "", fmt.Errorf("failed to parse the pod full name %q", podFullName)
	}
	return parts[1], parts[2], parts[3], nil
}

func (r *runtime) parseHyperContainerFullName(containerName string) (string, string, string, string, error) {
	parts := strings.Split(containerName, "_")
	if len(parts) != 6 {
		return "", "", "", "", fmt.Errorf("failed to parse the container full name %q", containerName)
	}
	return parts[1], parts[2], parts[3], parts[4], nil
}

// GetPods returns a list containers group by pods. The boolean parameter
// specifies whether the runtime returns all containers including those already
// exited and dead containers (used for garbage collection).
func (r *runtime) GetPods(all bool) ([]*kubecontainer.Pod, error) {
	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		return nil, err
	}

	var kubepods []*kubecontainer.Pod
	for _, podInfo := range podInfos {
		if !all && podInfo.Status != "running" {
			continue
		}

		var pod kubecontainer.Pod
		var containers []*kubecontainer.Container

		podID, podName, podNamespace, err := r.parseHyperPodFullName(podInfo.PodName)
		if err != nil {
			glog.Errorf("Hyper: pod %s is not managed by kubelet", podInfo.PodName)
			continue
		}

		pod.ID = types.UID(podID)
		pod.Name = podName
		pod.Namespace = podNamespace

		for _, cinfo := range podInfo.PodInfo.Spec.Containers {
			var container kubecontainer.Container
			container.ID = types.UID(cinfo.ContainerID)
			container.Image = cinfo.Image

			for _, cstatus := range podInfo.PodInfo.Status.Status {
				if cstatus.ContainerID == cinfo.ContainerID {
					createAt, err := parseTimeString(cstatus.Running.StartedAt)
					if err == nil {
						container.Created = createAt.Unix()
					}
				}
			}

			_, _, _, containerName, err := r.parseHyperContainerFullName(cinfo.Name)
			if err != nil {
				return nil, err
			}
			container.Name = strings.Split(containerName, ".")[0]

			hash, err := strconv.ParseUint(strings.Split(containerName, ".")[1], 16, 8)
			if err == nil {
				container.Hash = hash
			}

			containers = append(containers, &container)
		}
		pod.Containers = containers

		kubepods = append(kubepods, &pod)
	}

	return kubepods, nil
}

func (r *runtime) buildHyperPodServices(pod *api.Pod) []HyperService {
	items, err := r.kubeClient.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		glog.Warningf("Get services failed: %v", err)
		return nil
	}

	var services []HyperService
	for _, svc := range items.Items {
		hyperService := HyperService{
			ServiceIP: svc.Spec.ClusterIP,
		}
		endpoints, _ := r.kubeClient.Endpoints(pod.Namespace).Get(svc.Name)
		for _, svcPort := range svc.Spec.Ports {
			hyperService.ServicePort = svcPort.Port
			for _, ep := range endpoints.Subsets {
				for _, epPort := range ep.Ports {
					if svcPort.Name == "" || svcPort.Name == epPort.Name {
						for _, eh := range ep.Addresses {
							hyperService.Hosts = append(hyperService.Hosts, HyperServiceBackend{
								HostIP:   eh.IP,
								HostPort: epPort.Port,
							})
						}
					}
				}
			}
			services = append(services, hyperService)
		}
	}

	return services
}

func (r *runtime) buildHyperPod(pod *api.Pod, pullSecrets []api.Secret) ([]byte, error) {
	// check and pull image
	for _, c := range pod.Spec.Containers {
		if err := r.imagePuller.PullImage(pod, &c, pullSecrets); err != nil {
			return nil, err
		}
	}

	// build hyper volume spec
	specMap := make(map[string]interface{})
	volumeMap, ok := r.volumeGetter.GetVolumes(pod.UID)
	if !ok {
		return nil, fmt.Errorf("cannot get the volumes for pod %q", kubecontainer.GetPodFullName(pod))
	}

	volumes := []map[string]string{}
	for name, volume := range volumeMap {
		glog.V(4).Infof("Hyper: volume %s %s", name, volume.GetPath())
		v := make(map[string]string)
		v[KEY_NAME] = name
		v[KEY_VOLUME_DRIVE] = VOLUME_TYPE_VFS
		v[KEY_VOLUME_SOURCE] = volume.GetPath()
		volumes = append(volumes, v)
	}
	specMap[KEY_VOLUMES] = volumes

	services := r.buildHyperPodServices(pod)
	if services == nil {
		// Just for fake
		services = []HyperService{
			{
				ServiceIP:   "127.0.0.2",
				ServicePort: 65534,
			},
		}
	}
	specMap["services"] = services

	// build hyper containers spec
	var containers []map[string]interface{}
	for _, container := range pod.Spec.Containers {
		c := make(map[string]interface{})
		c[KEY_NAME] = r.buildHyperContainerFullName(
			string(pod.UID),
			string(pod.Name),
			string(pod.Namespace),
			container.Name,
			container)
		c[KEY_IMAGE] = container.Image
		c[KEY_TTY] = container.TTY
		if len(container.Command) > 0 {
			c[KEY_COMMAND] = container.Command
		}
		if container.WorkingDir != "" {
			c[KEY_WORKDIR] = container.WorkingDir
		}
		if len(container.Args) > 0 {
			c[KEY_CONTAINER_ARGS] = container.Args
		}

		opts, err := r.generator.GenerateRunContainerOptions(pod, &container)
		if err != nil {
			return nil, err
		}

		// dns
		if len(opts.DNS) > 0 {
			c[KEY_DNS] = opts.DNS
		}

		// envs
		envs := make([]map[string]string, 0, 1)
		for _, e := range opts.Envs {
			envs = append(envs, map[string]string{
				"env":   e.Name,
				"value": e.Value,
			})
		}
		c[KEY_ENVS] = envs

		// port-mappings
		var ports []map[string]interface{}
		for _, mapping := range opts.PortMappings {
			p := make(map[string]interface{})
			p[KEY_CONTAINER_PORT] = mapping.ContainerPort
			if mapping.HostPort != 0 {
				p[KEY_HOST_PORT] = mapping.HostPort
			}
			p[KEY_PROTOCOL] = mapping.Protocol
			ports = append(ports, p)
		}
		c[KEY_PORTS] = ports

		// volumes
		if len(opts.Mounts) > 0 {
			var containerVolumes []map[string]interface{}
			for _, volume := range opts.Mounts {
				v := make(map[string]interface{})
				v[KEY_MOUNTPATH] = volume.ContainerPath
				v[KEY_VOLUME] = volume.Name
				v[KEY_READONLY] = volume.ReadOnly
				containerVolumes = append(containerVolumes, v)
			}
			c[KEY_VOLUMES] = containerVolumes
		}

		containers = append(containers, c)
	}
	specMap[KEY_CONTAINERS] = containers

	// build hyper pod resources spec
	var podCPULimit, podMemLimit int64
	podResource := make(map[string]int64)
	for _, container := range pod.Spec.Containers {
		resource := container.Resources.Limits
		var containerCPULimit, containerMemLimit int64
		for name, limit := range resource {
			switch name {
			case api.ResourceCPU:
				containerCPULimit = limit.MilliValue()
			case api.ResourceMemory:
				containerMemLimit = limit.MilliValue()
			}
		}
		if containerCPULimit == 0 {
			containerCPULimit = hyperDefaultContainerCPU
		}
		if containerMemLimit == 0 {
			containerMemLimit = hyperDefaultContainerMem * 1024 * 1024 * 1000
		}
		podCPULimit += containerCPULimit
		podMemLimit += containerMemLimit
	}

	podResource[KEY_VCPU] = (podCPULimit + 999) / 1000
	podResource[KEY_MEMORY] = int64(hyperBaseMemory) + ((podMemLimit)/1000/1024)/1024
	specMap[KEY_RESOURCE] = podResource
	glog.V(5).Infof("Hyper: pod limit vcpu=%v mem=%vMiB", podResource[KEY_VCPU], podResource[KEY_MEMORY])

	// other params required
	specMap[KEY_ID] = r.buildHyperPodFullName(string(pod.UID), string(pod.Name), string(pod.Namespace))
	specMap[KEY_TTY] = true

	podData, err := json.Marshal(specMap)
	if err != nil {
		return nil, err
	}

	return podData, nil
}

func (r *runtime) savePodSpec(spec, podFullName string) error {
	// ensure hyperPodSpecDir is created
	_, err := os.Stat(hyperPodSpecDir)
	if err != nil && os.IsNotExist(err) {
		e := os.MkdirAll(hyperPodSpecDir, 0755)
		if e != nil {
			return e
		}
	}

	// save spec to file
	specFileName := path.Join(hyperPodSpecDir, podFullName)
	err = ioutil.WriteFile(specFileName, []byte(spec), 0664)
	if err != nil {
		return err
	}

	return nil
}

func (r *runtime) getPodSpec(podFullName string) (string, error) {
	specFileName := path.Join(hyperPodSpecDir, podFullName)
	_, err := os.Stat(specFileName)
	if err != nil {
		return "", err
	}

	spec, err := ioutil.ReadFile(specFileName)
	if err != nil {
		return "", err
	}

	return string(spec), nil
}

func (r *runtime) RunPod(pod *api.Pod, pullSecrets []api.Secret) error {
	podData, err := r.buildHyperPod(pod, pullSecrets)
	if err != nil {
		glog.Errorf("Hyper: buildHyperPod failed, error: %s", err)
		return err
	}

	podFullName := r.buildHyperPodFullName(string(pod.UID), string(pod.Name), string(pod.Namespace))
	err = r.savePodSpec(string(podData), podFullName)
	if err != nil {
		glog.Errorf("Hyper: savePodSpec failed, error: %s", err)
		return err
	}

	// Setup pod's network by network plugin
	err = r.networkPlugin.SetUpPod(pod.Namespace, podFullName, "", "hyper")
	if err != nil {
		glog.Errorf("Hyper: networkPlugin.SetUpPod %s failed, error: %s", pod.Name, err)
		return err
	}

	// Create and start hyper pod
	podSpec, err := r.getPodSpec(podFullName)
	if err != nil {
		glog.Errorf("Hyper: create pod %s failed, error: %s", podFullName, err)
		return err
	}
	result, err := r.hyperClient.CreatePod(podSpec)
	if err != nil {
		glog.Errorf("Hyper: create pod %s failed, error: %s", podData, err)
		return err
	}

	podID := string(result["ID"].(string))
	err = r.hyperClient.StartPod(podID)
	if err != nil {
		glog.Errorf("Hyper: start pod %s (ID:%s) failed, error: %s", pod.Name, podID, err)
		destroyErr := r.hyperClient.RemovePod(podID)
		if destroyErr != nil {
			glog.Errorf("Hyper: destory pod %s (ID:%s) failed: %s", pod.Name, podID, destroyErr)
		}
		return err
	}

	return nil
}

// Syncs the running pod into the desired pod.
func (r *runtime) SyncPod(pod *api.Pod, runningPod kubecontainer.Pod, podStatus api.PodStatus, pullSecrets []api.Secret, backOff *util.Backoff) error {
	podFullName := r.buildHyperPodFullName(string(pod.UID), string(pod.Name), string(pod.Namespace))
	if len(runningPod.Containers) == 0 {
		glog.V(4).Infof("Pod %q is not running, will start it", podFullName)
		return r.RunPod(pod, pullSecrets)
	}

	// Add references to all containers.
	unidentifiedContainers := make(map[types.UID]*kubecontainer.Container)
	for _, c := range runningPod.Containers {
		unidentifiedContainers[c.ID] = c
	}

	restartPod := false
	for _, container := range pod.Spec.Containers {
		expectedHash := kubecontainer.HashContainer(&container)

		c := runningPod.FindContainerByName(container.Name)
		if c == nil {
			if kubecontainer.ShouldContainerBeRestarted(&container, pod, &podStatus, r.readinessManager) {
				glog.V(3).Infof("Container %+v is dead, but RestartPolicy says that we should restart it.", container)
				restartPod = true
				break
			}
			continue
		}

		containerChanged := c.Hash != 0 && c.Hash != expectedHash
		if containerChanged {
			glog.V(4).Infof("Pod %q container %q hash changed (%d vs %d), it will be killed and re-created.",
				podFullName, container.Name, c.Hash, expectedHash)
			restartPod = true
			break
		}

		result, err := r.prober.Probe(pod, podStatus, container, string(c.ID), c.Created)
		if err == nil && result != probe.Success {
			glog.V(4).Infof("Pod %q container %q is unhealthy (probe result: %v), it will be killed and re-created.",
				podFullName, container.Name, result)
			restartPod = true
			break
		}

		if err != nil {
			glog.V(2).Infof("Hyper: probe container %q failed: %v", container.Name, err)
		}

		delete(unidentifiedContainers, c.ID)
	}

	// If there is any unidentified containers, restart the pod.
	if len(unidentifiedContainers) > 0 {
		restartPod = true
	}

	if restartPod {
		if err := r.KillPod(nil, runningPod); err != nil {
			glog.Errorf("Hyper: kill pod %s failed, error: %s", runningPod.Name, err)
			return err
		}
		if err := r.RunPod(pod, pullSecrets); err != nil {
			glog.Errorf("Hyper: run pod %s failed, error: %s", pod.Name, err)
			return err
		}
	}
	return nil
}

// KillPod kills all the containers of a pod.
func (r *runtime) KillPod(pod *api.Pod, runningPod kubecontainer.Pod) error {
	var podID string
	podName := r.buildHyperPodFullName(string(runningPod.ID), runningPod.Name, runningPod.Namespace)
	glog.V(4).Infof("Hyper: killing pod %q.", podName)

	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		glog.Errorf("Hyper: ListPods failed, error: %s", err)
		return err
	}

	for _, podInfo := range podInfos {
		if podInfo.PodName == podName {
			podID = podInfo.PodID
			break
		}
	}

	//err = r.hyperClient.RemovePod(podID)
	cmds := append([]string{}, "rm", podID)
	_, err = r.runCommand(cmds...)
	if err != nil {
		glog.Errorf("Hyper: remove pod %s failed, error: %s", podID, err)
		return err
	}

	// Teardown pod's network
	podFullName := r.buildHyperPodFullName(string(pod.UID), string(pod.Name), string(pod.Namespace))
	err = r.networkPlugin.TearDownPod(runningPod.Namespace, podFullName, "", "hyper")
	if err != nil {
		glog.Errorf("Hyper: networkPlugin.TearDownPod failed, error: %v", err)
		return err
	}

	// Delete pod spec file
	specFileName := path.Join(hyperPodSpecDir, podFullName)
	_, err = os.Stat(specFileName)
	if err == nil {
		e := os.Remove(specFileName)
		if e != nil {
			glog.Errorf("Hyper: delete spec file for %s failed, error: %v", pod.Name, e)
		}
	}

	return nil
}

// GetPodStatus retrieves the status of the pod, including the information of
// all containers in the pod. Clients of this interface assume the containers
// statuses in a pod always have a deterministic ordering (eg: sorted by name).
func (r *runtime) GetPodStatus(pod *api.Pod) (*api.PodStatus, error) {
	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		glog.Errorf("Hyper: ListPods failed, error: %s", err)
		return nil, err
	}

	var status api.PodStatus
	podFullName := r.buildHyperPodFullName(string(pod.UID), string(pod.Name), string(pod.Namespace))
	for _, podInfo := range podInfos {
		if podInfo.PodName != podFullName {
			continue
		}

		if len(podInfo.PodInfo.Status.PodIP) > 0 {
			status.PodIP = podInfo.PodInfo.Status.PodIP[0]
		}

		status.HostIP = podInfo.PodInfo.Status.HostIP
		status.Phase = api.PodPhase(podInfo.PodInfo.Status.Phase)
		status.Message = podInfo.PodInfo.Status.Message
		status.Reason = podInfo.PodInfo.Status.Reason
		for _, containerInfo := range podInfo.PodInfo.Status.Status {
			for _, container := range podInfo.PodInfo.Spec.Containers {
				if container.ContainerID == containerInfo.ContainerID {
					status.ContainerStatuses = append(
						status.ContainerStatuses,
						r.getContainerStatus(containerInfo, container.Image, container.ImageID))
				}
			}
		}
	}

	glog.V(5).Infof("Hyper: get pod %s status %s", podFullName, status)

	return &status, nil
}

// PullImage pulls an image from the network to local storage using the supplied
// secrets if necessary.
func (r *runtime) PullImage(image kubecontainer.ImageSpec, pullSecrets []api.Secret) error {
	img := image.Image

	repoToPull, tag := parseImageName(img)
	if exist, _ := r.hyperClient.IsImagePresent(repoToPull, tag); exist {
		return nil
	}

	keyring, err := credentialprovider.MakeDockerKeyring(pullSecrets, r.dockerKeyring)
	if err != nil {
		return err
	}

	creds, ok := keyring.Lookup(repoToPull)
	if !ok || len(creds) == 0 {
		glog.V(4).Infof("Hyper: pulling image %s without credentials", img)
	}

	var credential string
	if len(creds) > 0 {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(creds[0]); err != nil {
			return err
		}
		credential = base64.URLEncoding.EncodeToString(buf.Bytes())
	}

	err = r.hyperClient.PullImage(img, credential)
	if err != nil {
		return fmt.Errorf("Hyper: Failed to pull image: %v:", err)
	}
	return nil
}

// IsImagePresent checks whether the container image is already in the local storage.
func (r *runtime) IsImagePresent(image kubecontainer.ImageSpec) (bool, error) {
	repoToPull, tag := parseImageName(image.Image)
	glog.V(4).Infof("Hyper: checking is image %s present", image.Image)
	exist, err := r.hyperClient.IsImagePresent(repoToPull, tag)
	if err != nil {
		glog.Warningf("Hyper: checking image failed, error: %s", err)
		return false, err
	}

	return exist, nil
}

// Gets all images currently on the machine.
func (r *runtime) ListImages() ([]kubecontainer.Image, error) {
	var images []kubecontainer.Image

	if outputs, err := r.hyperClient.ListImages(); err != nil {
		for _, imgInfo := range outputs {
			image := kubecontainer.Image{
				ID:   imgInfo.imageID,
				Tags: []string{imgInfo.tag},
				Size: imgInfo.virtualSize,
			}
			images = append(images, image)
		}
	}

	return images, nil
}

// Removes the specified image.
func (r *runtime) RemoveImage(image kubecontainer.ImageSpec) error {
	err := r.hyperClient.RemoveImage(image.Image)
	if err != nil {
		return err
	}

	return nil
}

// GetContainerLogs returns logs of a specific container. By
// default, it returns a snapshot of the container log. Set 'follow' to true to
// stream the log. Set 'follow' to false and specify the number of lines (e.g.
// "100" or "all") to tail the log.
func (r *runtime) GetContainerLogs(pod *api.Pod, containerID string, tail string, follow bool, stdout, stderr io.Writer) error {
	// TODO: get container logs for hyper
	return fmt.Errorf("Hyper: GetContainerLogs unimplemented")
}

// Runs the command in the container of the specified pod
func (r *runtime) RunInContainer(containerID string, cmd []string) ([]byte, error) {
	glog.V(4).Infof("Hyper: running %s in container %s.", cmd, containerID)

	args := append([]string{}, "exec", containerID)
	args = append(args, cmd...)

	result, err := r.runCommand(args...)
	return []byte(strings.Join(result, "\n")), err
}

// Forward the specified port from the specified pod to the stream.
func (r *runtime) PortForward(pod *kubecontainer.Pod, port uint16, stream io.ReadWriteCloser) error {
	// TODO: port forward for hyper
	return fmt.Errorf("Hyper: PortForward unimplemented")
}

// Runs the command in the container of the specified pod.
// Attaches the processes stdin, stdout, and stderr. Optionally uses a
// tty.
func (r *runtime) ExecInContainer(containerID string, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool) error {
	glog.V(4).Infof("Hyper: execing %s in container %s.", cmd, containerID)

	args := append([]string{}, "exec", "-a", containerID)
	args = append(args, cmd...)
	command := r.buildCommand(args...)

	p, err := kubecontainer.StartPty(command)
	if err != nil {
		return err
	}
	defer p.Close()

	// make sure to close the stdout stream
	defer stdout.Close()

	if stdin != nil {
		go io.Copy(p, stdin)
	}

	if stdout != nil {
		go io.Copy(stdout, p)
	}
	return command.Wait()

}

func (r *runtime) AttachContainer(containerID string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool) error {
	glog.V(4).Infof("Hyper: attaching container %s.", containerID)

	opts := AttachToContainerOptions{
		Container:    containerID,
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Stream:       true,
		Logs:         true,
		Stdin:        stdin != nil,
		Stdout:       stdout != nil,
		Stderr:       stderr != nil,
		RawTerminal:  tty,
	}
	return r.hyperClient.Attach(opts)
}
