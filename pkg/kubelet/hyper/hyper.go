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
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/credentialprovider"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
	"k8s.io/kubernetes/pkg/kubelet/network"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util"
	utilexec "k8s.io/kubernetes/pkg/util/exec"
)

const (
	hyperBinName                = "hyper"
	typeHyper                   = "hyper"
	hyperContainerNamePrefix    = "kube"
	hyperPodNamePrefix          = "kube"
	hyperDefaultContainerCPU    = 1
	hyperDefaultContainerMem    = 128
	hyperPodSpecDir             = "/var/lib/kubelet/hyper"
	hyperLogsDir                = "/var/run/hyper/Pods"
	minimumGracePeriodInSeconds = 2
)

// runtime implements the container runtime for hyper
type runtime struct {
	hyperBinAbsPath     string
	dockerKeyring       credentialprovider.DockerKeyring
	containerLogsDir    string
	containerRefManager *kubecontainer.RefManager
	generator           kubecontainer.RunContainerOptionsGenerator
	recorder            record.EventRecorder
	livenessManager     proberesults.Manager
	networkPlugin       network.NetworkPlugin
	volumeGetter        volumeGetter
	hyperClient         *HyperClient
	kubeClient          client.Interface
	imagePuller         kubecontainer.ImagePuller
	os                  kubecontainer.OSInterface
	version             kubecontainer.Version

	// Disable the internal haproxy service in Hyper pods
	disableHyperInternalService bool

	// Runner of lifecycle events.
	runner kubecontainer.HandlerRunner
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
	livenessManager proberesults.Manager,
	volumeGetter volumeGetter,
	kubeClient client.Interface,
	imageBackOff *util.Backoff,
	serializeImagePulls bool,
	httpClient kubetypes.HttpGetter,
	disableHyperInternalService bool,
	containerLogsDir string,
	os kubecontainer.OSInterface,
) (kubecontainer.Runtime, error) {
	// check hyper has already installed
	hyperBinAbsPath, err := exec.LookPath(hyperBinName)
	if err != nil {
		glog.Errorf("Hyper: can't find hyper binary")
		return nil, fmt.Errorf("cannot find hyper binary: %v", err)
	}

	hyper := &runtime{
		hyperBinAbsPath:             hyperBinAbsPath,
		dockerKeyring:               credentialprovider.NewDockerKeyring(),
		containerLogsDir:            containerLogsDir,
		containerRefManager:         containerRefManager,
		generator:                   generator,
		livenessManager:             livenessManager,
		os:                          os,
		recorder:                    recorder,
		networkPlugin:               networkPlugin,
		volumeGetter:                volumeGetter,
		hyperClient:                 NewHyperClient(),
		kubeClient:                  kubeClient,
		disableHyperInternalService: disableHyperInternalService,
	}

	if serializeImagePulls {
		hyper.imagePuller = kubecontainer.NewSerializedImagePuller(recorder, hyper, imageBackOff)
	} else {
		hyper.imagePuller = kubecontainer.NewImagePuller(recorder, hyper, imageBackOff)
	}

	version, err := hyper.hyperClient.Version()
	if err != nil {
		return nil, fmt.Errorf("cannot get hyper version: %v", err)
	}

	hyperVersion, err := parseVersion(version)
	if err != nil {
		return nil, fmt.Errorf("cannot get hyper version: %v", err)
	}

	hyper.version = hyperVersion

	hyper.runner = lifecycle.NewHandlerRunner(httpClient, hyper, hyper)

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
	output, err := r.buildCommand(args...).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// Version invokes 'hyper version' to get the version information of the hyper
// runtime on the machine.
// The return values are an int array containers the version number.
func (r *runtime) Version() (kubecontainer.Version, error) {
	return r.version, nil
}

// Type returns the name of the container runtime
func (r *runtime) Type() string {
	return "hyper"
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

func (r *runtime) getContainerStatus(container ContainerStatus, image, imageID, startTime string) *kubecontainer.ContainerStatus {
	status := &kubecontainer.ContainerStatus{}

	_, _, _, containerName, restartCount, _, err := r.parseHyperContainerFullName(container.Name)
	if err != nil {
		return status
	}

	status.Name = containerName
	status.ID = kubecontainer.ContainerID{
		Type: typeHyper,
		ID:   container.ContainerID,
	}
	status.Image = image
	status.ImageID = imageID
	status.RestartCount = restartCount

	switch container.Phase {
	case StatusRunning:
		runningStartedAt, err := parseTimeString(container.Running.StartedAt)
		if err != nil {
			glog.Errorf("Hyper: can't parse runningStartedAt %s", container.Running.StartedAt)
			return status
		}

		status.State = kubecontainer.ContainerStateRunning
		status.StartedAt = runningStartedAt
	case StatusFailed, StatusSuccess:
		// TODO: ensure container.Terminated.StartedAt
		if container.Terminated.StartedAt == "" {
			status.StartedAt = time.Now().Add(-2 * time.Second)
		} else {
			terminatedStartedAt, err := parseTimeString(container.Terminated.StartedAt)
			if err != nil {
				glog.Errorf("Hyper: can't parse terminatedStartedAt %s", container.Terminated.StartedAt)
				return status
			}
			status.StartedAt = terminatedStartedAt
		}

		// TODO: ensure container.Terminated.FinishedAt
		if container.Terminated.FinishedAt == "" {
			status.FinishedAt = time.Now()
		} else {
			terminatedFinishedAt, err := parseTimeString(container.Terminated.FinishedAt)
			if err != nil {
				glog.Errorf("Hyper: can't parse terminatedFinishedAt %s", container.Terminated.FinishedAt)
				return status
			}

			status.FinishedAt = terminatedFinishedAt
		}

		status.State = kubecontainer.ContainerStateExited
		status.Reason = container.Terminated.Reason

		status.Message = container.Terminated.Message
		status.ExitCode = container.Terminated.ExitCode
	default:
		if startTime == "" {
			status.StartedAt = time.Now().Add(-2 * time.Second)
		} else {
			startedAt, err := parseTimeString(startTime)
			if err != nil {
				glog.Errorf("Hyper: can't parse startTime %s", container.Terminated.StartedAt)
				return status
			}

			status.StartedAt = startedAt
		}

		status.FinishedAt = time.Now()
		status.State = kubecontainer.ContainerStateExited
		status.Reason = container.Waiting.Reason
		status.ExitCode = 0
	}

	return status
}

func (r *runtime) buildHyperContainerFullName(uid, podName, namespace, containerName string, restartCount int, container api.Container) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%d_%s",
		hyperContainerNamePrefix,
		uid,
		podName,
		namespace,
		containerName,
		restartCount,
		strconv.FormatUint(kubecontainer.HashContainer(&container), 16))
}

func (r *runtime) parseHyperContainerFullName(containerName string) (string, string, string, string, int, string, error) {
	parts := strings.Split(containerName, "_")
	if len(parts) != 7 {
		return "", "", "", "", 0, "", fmt.Errorf("failed to parse the container full name %q", containerName)
	}

	restartCount, err := strconv.Atoi(parts[5])
	if err != nil {
		return "", "", "", "", 0, "", fmt.Errorf("failed to parse the container full name %q", containerName)
	}
	return parts[1], parts[2], parts[3], parts[4], restartCount, parts[6], nil
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
		var pod kubecontainer.Pod
		var containers []*kubecontainer.Container

		if !all && podInfo.Status != StatusRunning {
			continue
		}

		podID := podInfo.PodInfo.Spec.Labels[KEY_API_POD_UID]
		podName, podNamespace, err := kubecontainer.ParsePodFullName(podInfo.PodName)
		if err != nil {
			glog.V(5).Infof("Hyper: pod %s is not managed by kubelet", podInfo.PodName)
			continue
		}

		pod.ID = types.UID(podID)
		pod.Name = podName
		pod.Namespace = podNamespace

		for _, cinfo := range podInfo.PodInfo.Spec.Containers {
			var container kubecontainer.Container
			container.ID = kubecontainer.ContainerID{Type: typeHyper, ID: cinfo.ContainerID}
			container.Image = cinfo.Image

			for _, cstatus := range podInfo.PodInfo.Status.Status {
				if cstatus.ContainerID == cinfo.ContainerID {
					switch cstatus.Phase {
					case StatusRunning:
						container.State = kubecontainer.ContainerStateRunning
					default:
						container.State = kubecontainer.ContainerStateExited
					}

					createAt, err := parseTimeString(cstatus.Running.StartedAt)
					if err == nil {
						container.Created = createAt.Unix()
					}
				}
			}

			_, _, _, containerName, _, containerHash, err := r.parseHyperContainerFullName(cinfo.Name)
			if err != nil {
				glog.V(5).Infof("Hyper: container %s is not managed by kubelet", cinfo.Name)
				continue
			}
			container.Name = containerName

			hash, err := strconv.ParseUint(containerHash, 16, 64)
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
	items, err := r.kubeClient.Services(pod.Namespace).List(api.ListOptions{})
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

func (r *runtime) buildHyperPod(pod *api.Pod, restartCount int, pullSecrets []api.Secret) ([]byte, error) {
	// check and pull image
	for _, c := range pod.Spec.Containers {
		if err, _ := r.imagePuller.PullImage(pod, &c, pullSecrets); err != nil {
			return nil, err
		}
	}

	// build hyper volume spec
	specMap := make(map[string]interface{})
	volumeMap, ok := r.volumeGetter.GetVolumes(pod.UID)
	if !ok {
		return nil, fmt.Errorf("cannot get the volumes for pod %q", kubecontainer.GetPodFullName(pod))
	}

	volumes := make([]map[string]interface{}, 0, 1)
	for name, volume := range volumeMap {
		glog.V(4).Infof("Hyper: volume %s, path %s, meta %s", name, volume.Builder.GetPath(), volume.Builder.GetMetaData())
		v := make(map[string]interface{})
		v[KEY_NAME] = name

		// Process rbd volume
		metadata := volume.Builder.GetMetaData()
		if metadata != nil && metadata["volume_type"].(string) == "rbd" {
			v[KEY_VOLUME_DRIVE] = metadata["volume_type"]
			v["source"] = "rbd:" + metadata["name"].(string)
			monitors := make([]string, 0, 1)
			for _, host := range metadata["hosts"].([]interface{}) {
				for _, port := range metadata["ports"].([]interface{}) {
					monitors = append(monitors, fmt.Sprintf("%s:%s", host.(string), port.(string)))
				}
			}
			v["option"] = map[string]interface{}{
				"user":     metadata["auth_username"],
				"keyring":  metadata["keyring"],
				"monitors": monitors,
			}
		} else {
			glog.V(4).Infof("Hyper: volume %s %s", name, volume.Builder.GetPath())

			v[KEY_VOLUME_DRIVE] = VOLUME_TYPE_VFS
			v[KEY_VOLUME_SOURCE] = volume.Builder.GetPath()
		}

		volumes = append(volumes, v)
	}

	glog.V(4).Infof("Hyper volumes: %v", volumes)

	if !r.disableHyperInternalService {
		services := r.buildHyperPodServices(pod)
		if services == nil {
			// services can't be null for kubernetes, so fake one if it is null
			services = []HyperService{
				{
					ServiceIP:   "127.0.0.2",
					ServicePort: 65534,
				},
			}
		}
		specMap["services"] = services
	}

	// build hyper containers spec
	var containers []map[string]interface{}
	var k8sHostNeeded = true
	dnsServers := make(map[string]string)
	for _, container := range pod.Spec.Containers {
		c := make(map[string]interface{})
		c[KEY_NAME] = r.buildHyperContainerFullName(
			string(pod.UID),
			string(pod.Name),
			string(pod.Namespace),
			container.Name,
			restartCount,
			container)
		c[KEY_IMAGE] = container.Image
		c[KEY_TTY] = container.TTY

		if container.WorkingDir != "" {
			c[KEY_WORKDIR] = container.WorkingDir
		}

		opts, err := r.generator.GenerateRunContainerOptions(pod, &container)
		if err != nil {
			return nil, err
		}

		command, args := kubecontainer.ExpandContainerCommandAndArgs(&container, opts.Envs)
		if len(command) > 0 {
			c[KEY_ENTRYPOINT] = command
		}
		if len(args) > 0 {
			c[KEY_COMMAND] = args
		}

		// dns
		for _, dns := range opts.DNS {
			dnsServers[dns] = dns
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

				// Setup global hosts volume
				if volume.Name == "k8s-managed-etc-hosts" && k8sHostNeeded {
					k8sHostNeeded = false
					volumes = append(volumes, map[string]interface{}{
						KEY_NAME:          volume.Name,
						KEY_VOLUME_DRIVE:  VOLUME_TYPE_VFS,
						KEY_VOLUME_SOURCE: volume.HostPath,
					})
				}
			}
			c[KEY_VOLUMES] = containerVolumes
		}

		containers = append(containers, c)
	}
	specMap[KEY_CONTAINERS] = containers
	specMap[KEY_VOLUMES] = volumes

	// dns
	if len(dnsServers) > 0 {
		dns := []string{}
		for d := range dnsServers {
			dns = append(dns, d)
		}
		specMap[KEY_DNS] = dns
	}

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
	podResource[KEY_MEMORY] = ((podMemLimit) / 1000 / 1024) / 1024
	specMap[KEY_RESOURCE] = podResource
	glog.V(5).Infof("Hyper: pod limit vcpu=%v mem=%vMiB", podResource[KEY_VCPU], podResource[KEY_MEMORY])

	// Setup labels
	podLabels := map[string]string{KEY_API_POD_UID: string(pod.UID)}
	for k, v := range pod.Labels {
		podLabels[k] = v
	}
	specMap[KEY_LABELS] = podLabels

	// other params required
	specMap[KEY_ID] = kubecontainer.BuildPodFullName(pod.Name, pod.Namespace)
	specMap[KEY_TTY] = false

	// Cap hostname at 63 chars (specification is 64bytes which is 63 chars and the null terminating char).
	const hostnameMaxLen = 63
	podHostname := pod.Name
	if len(podHostname) > hostnameMaxLen {
		podHostname = podHostname[:hostnameMaxLen]
	}
	specMap[KEY_HOSTNAME] = podHostname

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

func (r *runtime) GetPodRestartCount(podID string) (int, error) {
	containers, err := r.hyperClient.ListContainers()
	if err != nil {
		return 0, err
	}

	for _, c := range containers {
		if c.podID != podID {
			continue
		}

		_, _, _, _, restartCount, _, err := r.parseHyperContainerFullName(c.name)
		if err != nil {
			continue
		}

		return restartCount, nil
	}

	return 0, nil
}

func (r *runtime) RunPod(pod *api.Pod, restartCount int, pullSecrets []api.Secret) error {
	podFullName := kubecontainer.BuildPodFullName(pod.Name, pod.Namespace)

	podData, err := r.buildHyperPod(pod, restartCount, pullSecrets)
	if err != nil {
		glog.Errorf("Hyper: buildHyperPod failed, error: %v", err)
		return err
	}

	err = r.savePodSpec(string(podData), podFullName)
	if err != nil {
		glog.Errorf("Hyper: savePodSpec failed, error: %v", err)
		return err
	}

	// Setup pod's network by network plugin
	err = r.networkPlugin.SetUpPod(pod.Namespace, pod.Name, "", "hyper")
	if err != nil {
		glog.Errorf("Hyper: networkPlugin.SetUpPod %s failed, error: %v", pod.Name, err)
		return err
	}

	// Create and start hyper pod
	podSpec, err := r.getPodSpec(podFullName)
	if err != nil {
		glog.Errorf("Hyper: create pod %s failed, error: %v", podFullName, err)
		return err
	}
	result, err := r.hyperClient.CreatePod(podSpec)
	if err != nil {
		glog.Errorf("Hyper: create pod %s failed, error: %v", podData, err)
		return err
	}

	podID := string(result["ID"].(string))

	err = r.hyperClient.StartPod(podID)
	if err != nil {
		glog.Errorf("Hyper: start pod %s (ID:%s) failed, error: %v", pod.Name, podID, err)
		destroyErr := r.hyperClient.RemovePod(podID)
		if destroyErr != nil {
			glog.Errorf("Hyper: destory pod %s (ID:%s) failed: %v", pod.Name, podID, destroyErr)
		}
		return err
	}

	podStatus, err := r.GetPodStatus(pod.UID, pod.Name, pod.Namespace)
	if err != nil {
		return err
	}
	runningPod := kubecontainer.ConvertPodStatusToRunningPod(podStatus)

	for _, container := range pod.Spec.Containers {
		var containerID kubecontainer.ContainerID

		for _, runningContainer := range runningPod.Containers {
			if container.Name == runningContainer.Name {
				containerID = runningContainer.ID
			}
		}

		// Update container references
		ref, err := kubecontainer.GenerateContainerRef(pod, &container)
		if err != nil {
			glog.Errorf("Couldn't make a ref to pod %q, container %v: '%v'", pod.Name, container.Name, err)
		} else {
			r.containerRefManager.SetRef(containerID, ref)
		}

		// Create a symbolic link to the Hyper container log file using a name
		// which captures the full pod name, the container name and the
		// container ID. Cluster level logging will capture these symbolic
		// filenames which can be used for search terms in Elasticsearch or for
		// labels for Cloud Logging.
		containerLogFile := path.Join(hyperLogsDir, podID, fmt.Sprintf("%s-json.log", containerID.ID))
		symlinkFile := LogSymlink(r.containerLogsDir, podFullName, container.Name, containerID.ID)
		if err = r.os.Symlink(containerLogFile, symlinkFile); err != nil {
			glog.Errorf("Failed to create symbolic link to the log file of pod %q container %q: %v", podFullName, container.Name, err)
		}

		if container.Lifecycle != nil && container.Lifecycle.PostStart != nil {
			handlerErr := r.runner.Run(containerID, pod, &container, container.Lifecycle.PostStart)
			if handlerErr != nil {
				err := fmt.Errorf("PostStart handler: %v", handlerErr)
				if e := r.KillPod(pod, runningPod); e != nil {
					glog.Errorf("KillPod %v failed: %v", podFullName, e)
				}
				return err
			}
		}
	}

	return nil
}

// Syncs the running pod into the desired pod.
func (r *runtime) SyncPod(pod *api.Pod, podStatus api.PodStatus, internalPodStatus *kubecontainer.PodStatus, pullSecrets []api.Secret, backOff *util.Backoff) error {
	// TODO: (random-liu) Stop using running pod in SyncPod()
	// TODO: (random-liu) Rename podStatus to apiPodStatus, rename internalPodStatus to podStatus, and use new pod status as much as possible,
	// we may stop using apiPodStatus someday.
	runningPod := kubecontainer.ConvertPodStatusToRunningPod(internalPodStatus)
	podFullName := kubecontainer.BuildPodFullName(pod.Name, pod.Namespace)

	// Add references to all containers.
	unidentifiedContainers := make(map[kubecontainer.ContainerID]*kubecontainer.Container)
	for _, c := range runningPod.Containers {
		unidentifiedContainers[c.ID] = c
	}

	restartPod := false
	for _, container := range pod.Spec.Containers {
		expectedHash := kubecontainer.HashContainer(&container)

		c := runningPod.FindContainerByName(container.Name)
		if c == nil {
			if kubecontainer.ShouldContainerBeRestartedOldVersion(&container, pod, &podStatus) {
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

		liveness, found := r.livenessManager.Get(c.ID)
		if found && liveness != proberesults.Success && pod.Spec.RestartPolicy != api.RestartPolicyNever {
			glog.Infof("Pod %q container %q is unhealthy, it will be killed and re-created.", podFullName, container.Name)
			restartPod = true
			break
		}

		delete(unidentifiedContainers, c.ID)
	}

	// If there is any unidentified containers, restart the pod.
	if len(unidentifiedContainers) > 0 {
		restartPod = true
	}

	if restartPod {
		restartCount := 0
		// Only kill existing pod
		podID, err := r.hyperClient.GetPodIDByName(podFullName)
		if err == nil && len(podID) > 0 {
			// Update pod restart count
			restartCount, err = r.GetPodRestartCount(podID)
			if err != nil {
				glog.Errorf("Hyper: get pod startcount failed: %v", err)
				return err
			}
			restartCount++

			if err := r.KillPod(pod, runningPod); err != nil {
				glog.Errorf("Hyper: kill pod %s failed, error: %s", runningPod.Name, err)
				return err
			}
		}

		if err := r.RunPod(pod, restartCount, pullSecrets); err != nil {
			glog.Errorf("Hyper: run pod %s failed, error: %s", pod.Name, err)
			return err
		}
	}
	return nil
}

// KillPod kills all the containers of a pod.
func (r *runtime) KillPod(pod *api.Pod, runningPod kubecontainer.Pod) error {
	if len(runningPod.Name) == 0 {
		return nil
	}

	// preStop hook
	for _, c := range runningPod.Containers {
		r.containerRefManager.ClearRef(c.ID)

		var container *api.Container
		if pod != nil {
			for i, containerSpec := range pod.Spec.Containers {
				if c.Name == containerSpec.Name {
					container = &pod.Spec.Containers[i]
					break
				}
			}
		}

		gracePeriod := int64(minimumGracePeriodInSeconds)
		if pod != nil {
			switch {
			case pod.DeletionGracePeriodSeconds != nil:
				gracePeriod = *pod.DeletionGracePeriodSeconds
			case pod.Spec.TerminationGracePeriodSeconds != nil:
				gracePeriod = *pod.Spec.TerminationGracePeriodSeconds
			}
		}

		start := unversioned.Now()
		if pod != nil && container != nil && container.Lifecycle != nil && container.Lifecycle.PreStop != nil {
			glog.V(4).Infof("Running preStop hook for container %q", container.Name)
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer util.HandleCrash()
				if err := r.runner.Run(c.ID, pod, container, container.Lifecycle.PreStop); err != nil {
					glog.Errorf("preStop hook for container %q failed: %v", container.Name, err)
				}
			}()
			select {
			case <-time.After(time.Duration(gracePeriod) * time.Second):
				glog.V(2).Infof("preStop hook for container %q did not complete in %d seconds", container.Name, gracePeriod)
			case <-done:
				glog.V(4).Infof("preStop hook for container %q completed", container.Name)
			}
			gracePeriod -= int64(unversioned.Now().Sub(start.Time).Seconds())
		}

		// always give containers a minimal shutdown window to avoid unnecessary SIGKILLs
		if gracePeriod < minimumGracePeriodInSeconds {
			gracePeriod = minimumGracePeriodInSeconds
		}
	}

	var podID string
	podName := kubecontainer.BuildPodFullName(runningPod.Name, runningPod.Namespace)
	glog.V(4).Infof("Hyper: killing pod %q.", podName)

	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		glog.Errorf("Hyper: ListPods failed, error: %s", err)
		return err
	}

	for _, podInfo := range podInfos {
		if podInfo.PodName == podName {
			podID = podInfo.PodID

			// Remove log links
			for _, c := range podInfo.PodInfo.Status.Status {
				_, _, _, containerName, _, _, err := r.parseHyperContainerFullName(c.Name)
				if err != nil {
					continue
				}
				symlinkFile := LogSymlink(r.containerLogsDir, podName, containerName, c.ContainerID)
				err = os.Remove(symlinkFile)
				if err != nil && !os.IsNotExist(err) {
					glog.Warningf("Failed to remove container log symlink %q: %v", symlinkFile, err)
				}
			}

			break
		}
	}

	cmds := append([]string{}, "rm", podID)
	_, err = r.runCommand(cmds...)
	if err != nil {
		glog.Errorf("Hyper: remove pod %s failed, error: %s", podID, err)
		return err
	}

	// Teardown pod's network
	err = r.networkPlugin.TearDownPod(runningPod.Namespace, runningPod.Name, "", "hyper")
	if err != nil {
		glog.Errorf("Hyper: networkPlugin.TearDownPod failed, error: %v", err)
		return err
	}

	// Delete pod spec file
	specFileName := path.Join(hyperPodSpecDir, podName)
	_, err = os.Stat(specFileName)
	if err == nil {
		e := os.Remove(specFileName)
		if e != nil {
			glog.Errorf("Hyper: delete spec file for %s failed, error: %v", runningPod.Name, e)
		}
	}

	return nil
}

// GetAPIPodStatus returns the status of the given pod.
func (r *runtime) GetAPIPodStatus(pod *api.Pod) (*api.PodStatus, error) {
	// Get the pod status.
	podStatus, err := r.GetPodStatus(pod.UID, pod.Name, pod.Namespace)
	if err != nil {
		return nil, err
	}
	return r.ConvertPodStatusToAPIPodStatus(pod, podStatus)
}

// GetPodStatus retrieves the status of the pod, including the information of
// all containers in the pod. Clients of this interface assume the containers
// statuses in a pod always have a deterministic ordering (eg: sorted by name).
func (r *runtime) GetPodStatus(uid types.UID, name, namespace string) (*kubecontainer.PodStatus, error) {
	status := &kubecontainer.PodStatus{
		ID:        uid,
		Name:      name,
		Namespace: namespace,
	}

	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		glog.Errorf("Hyper: ListPods failed, error: %s", err)
		return nil, err
	}

	podFullName := kubecontainer.BuildPodFullName(name, namespace)
	for _, podInfo := range podInfos {
		if podInfo.PodName != podFullName {
			continue
		}

		if len(podInfo.PodInfo.Status.PodIP) > 0 {
			status.IP = podInfo.PodInfo.Status.PodIP[0]
		}

		for _, containerInfo := range podInfo.PodInfo.Status.Status {
			for _, container := range podInfo.PodInfo.Spec.Containers {
				if container.ContainerID == containerInfo.ContainerID {
					status.ContainerStatuses = append(
						status.ContainerStatuses,
						r.getContainerStatus(containerInfo, container.Image, container.ImageID,
							podInfo.PodInfo.Status.StartTime))
				}
			}
		}
	}

	glog.V(5).Infof("Hyper: get pod %s status %s", podFullName, status)

	return status, nil
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
		return fmt.Errorf("Hyper: Failed to pull image: %v", err)
	}

	if exist, _ := r.hyperClient.IsImagePresent("haproxy", "1.4"); !exist {
		err = r.hyperClient.PullImage("haproxy:1.4", credential)
		if err != nil {
			return fmt.Errorf("Hyper: Failed to pull haproxy:1.4 image: %v", err)
		}
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
				ID:       imgInfo.imageID,
				RepoTags: []string{fmt.Sprintf("%v:%v", imgInfo.repository, imgInfo.tag)},
				Size:     imgInfo.virtualSize,
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
func (r *runtime) GetContainerLogs(pod *api.Pod, containerID kubecontainer.ContainerID, logOptions *api.PodLogOptions, stdout, stderr io.Writer) error {
	glog.V(4).Infof("Hyper: running logs on container %s", containerID.ID)

	var tailLines, since int64
	if logOptions.SinceSeconds != nil && *logOptions.SinceSeconds != 0 {
		since = *logOptions.SinceSeconds
	}
	if logOptions.TailLines != nil && *logOptions.TailLines != 0 {
		tailLines = *logOptions.TailLines
	}
	opts := ContainerLogsOptions{
		Container:    containerID.ID,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Follow:       logOptions.Follow,
		Timestamps:   logOptions.Timestamps,
		Since:        since,
		TailLines:    tailLines,
	}

	return r.hyperClient.ContainerLogs(opts)
}

// hyperExitError implemets /pkg/util/exec.ExitError interface.
type hyperExitError struct{ *exec.ExitError }

var _ utilexec.ExitError = &hyperExitError{}

func (r *hyperExitError) ExitStatus() int {
	if status, ok := r.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}
	return 0
}

// Runs the command in the container of the specified pod
func (r *runtime) RunInContainer(containerID kubecontainer.ContainerID, cmd []string) ([]byte, error) {
	glog.V(4).Infof("Hyper: running %s in container %s.", cmd, containerID.ID)

	buffer := bytes.NewBuffer(nil)
	err := r.ExecInContainer(containerID, cmd, nil, nopCloser{buffer}, nil, true)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = &hyperExitError{exitErr}
		}

		return nil, err
	}

	return buffer.ReadBytes('\n')
}

// Forward the specified port from the specified pod to the stream.
func (r *runtime) PortForward(pod *kubecontainer.Pod, port uint16, stream io.ReadWriteCloser) error {
	// TODO: port forward for hyper
	return fmt.Errorf("Hyper: PortForward unimplemented")
}

// Runs the command in the container of the specified pod.
// Attaches the processes stdin, stdout, and stderr. Optionally uses a
// tty.
func (r *runtime) ExecInContainer(containerID kubecontainer.ContainerID, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool) error {
	glog.V(4).Infof("Hyper: execing %s in container %s.", cmd, containerID.ID)

	args := append([]string{}, "exec", "-a", containerID.ID)
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

func (r *runtime) AttachContainer(containerID kubecontainer.ContainerID, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool) error {
	glog.V(4).Infof("Hyper: attaching container %s.", containerID.ID)

	opts := AttachToContainerOptions{
		Container:    containerID.ID,
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
	}

	return r.hyperClient.Attach(opts)
}

// TODO(yifan): Delete this function when the logic is moved to kubelet.
func (r *runtime) ConvertPodStatusToAPIPodStatus(pod *api.Pod, status *kubecontainer.PodStatus) (*api.PodStatus, error) {
	apiPodStatus := &api.PodStatus{
		PodIP:             status.IP,
		ContainerStatuses: make([]api.ContainerStatus, 0, 1),
	}

	containerStatuses := make(map[string]*api.ContainerStatus)
	for _, c := range status.ContainerStatuses {
		var st api.ContainerState
		switch c.State {
		case kubecontainer.ContainerStateRunning:
			st.Running = &api.ContainerStateRunning{
				StartedAt: unversioned.NewTime(c.StartedAt),
			}
		case kubecontainer.ContainerStateExited:
			st.Terminated = &api.ContainerStateTerminated{
				ExitCode:    c.ExitCode,
				StartedAt:   unversioned.NewTime(c.StartedAt),
				Reason:      c.Reason,
				Message:     c.Message,
				FinishedAt:  unversioned.NewTime(c.FinishedAt),
				ContainerID: c.ID.String(),
			}
		default:
			// Unknown state.
			st.Waiting = &api.ContainerStateWaiting{}
		}

		status, ok := containerStatuses[c.Name]
		if !ok {
			containerStatuses[c.Name] = &api.ContainerStatus{
				Name:         c.Name,
				Image:        c.Image,
				ImageID:      c.ImageID,
				ContainerID:  c.ID.String(),
				RestartCount: c.RestartCount,
				State:        st,
			}
			continue
		}

		// Found multiple container statuses, fill that as last termination state.
		if status.LastTerminationState.Waiting == nil &&
			status.LastTerminationState.Running == nil &&
			status.LastTerminationState.Terminated == nil {
			status.LastTerminationState = st
		}
	}

	for _, c := range pod.Spec.Containers {
		cs, ok := containerStatuses[c.Name]
		if !ok {
			cs = &api.ContainerStatus{
				Name:  c.Name,
				Image: c.Image,
				// TODO(yifan): Add reason and message.
				State: api.ContainerState{Waiting: &api.ContainerStateWaiting{}},
			}
		}
		apiPodStatus.ContainerStatuses = append(apiPodStatus.ContainerStatuses, *cs)
	}

	sort.Sort(kubetypes.SortedContainerStatuses(apiPodStatus.ContainerStatuses))

	return apiPodStatus, nil
}

func (r *runtime) GarbageCollect(gcPolicy kubecontainer.ContainerGCPolicy) error {
	podInfos, err := r.hyperClient.ListPods()
	if err != nil {
		return err
	}

	for _, pod := range podInfos {
		// omit not managed pods
		_, _, err := kubecontainer.ParsePodFullName(pod.PodName)
		if err != nil {
			continue
		}

		// omit running pods
		if pod.Status == StatusRunning {
			continue
		}

		// TODO: Replace lastTime with pod exited time
		lastTime, err := parseTimeString(pod.PodInfo.Status.StartTime)
		if err != nil {
			lastTime = time.Now().Add(-1 * time.Hour)
		}

		if lastTime.Before(time.Now().Add(-gcPolicy.MinAge)) {
			// Remove log links
			for _, c := range pod.PodInfo.Status.Status {
				_, _, _, containerName, _, _, err := r.parseHyperContainerFullName(c.Name)
				if err != nil {
					continue
				}
				symlinkFile := LogSymlink(r.containerLogsDir, pod.PodName, containerName, c.ContainerID)
				err = os.Remove(symlinkFile)
				if err != nil && !os.IsNotExist(err) {
					glog.Warningf("Failed to remove container log symlink %q: %v", symlinkFile, err)
				}
			}

			// Remove the pod
			cmds := append([]string{}, "rm", pod.PodID)
			_, err = r.runCommand(cmds...)
			if err != nil {
				glog.Warningf("Hyper GarbageCollect: remove pod %s failed, error: %s", pod.PodID, err)
				return err
			}
		}

	}

	return nil
}

// TODO(yifan): Delete this function when the logic is moved to kubelet.
func (r *runtime) GetPodStatusAndAPIPodStatus(pod *api.Pod) (*kubecontainer.PodStatus, *api.PodStatus, error) {
	// Get the pod status.
	podStatus, err := r.GetPodStatus(pod.UID, pod.Name, pod.Namespace)
	if err != nil {
		return nil, nil, err
	}
	apiPodStatus, err := r.ConvertPodStatusToAPIPodStatus(pod, podStatus)
	return podStatus, apiPodStatus, err
}

// LogSymlink generates symlink file path for specified container
func LogSymlink(containerLogsDir, podFullName, containerName, containerID string) string {
	return path.Join(containerLogsDir, fmt.Sprintf("%s_%s-%s.log", podFullName, containerName, containerID))
}
