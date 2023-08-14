package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/loft-sh/devpod/pkg/devcontainer/buildkit"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/feature"
	"github.com/loft-sh/devpod/pkg/driver/docker"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/random"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultRootlessBuildkitImage = "moby/buildkit:master-rootless"

const defaultBuildkitImage = "moby/buildkit:master"

func (k *kubernetesDriver) PushDevContainer(ctx context.Context, image string) error {
	return fmt.Errorf("currently prebuilding images through Kubernetes is not supported")
}

func (k *kubernetesDriver) BuildDevContainer(
	ctx context.Context,
	labels []string,
	parsedConfig *config.SubstitutedConfig,
	extendedBuildInfo *feature.ExtendedBuildInfo,
	dockerfilePath,
	dockerfileContent string,
	localWorkspaceFolder string,
	options config.BuildOptions,
) (*config.BuildInfo, error) {
	// namespace
	if k.namespace != "" && k.config.CreateNamespace == "true" {
		k.Log.Debugf("Create namespace '%s'", k.namespace)
		buf := &bytes.Buffer{}
		err := k.runCommand(ctx, []string{"create", "ns", k.namespace}, nil, buf, buf)
		if err != nil {
			k.Log.Debugf("Error creating namespace: %v", err)
		}
	}

	// get cluster architecture
	arch, err := k.getClusterArchitecture(ctx)
	if err != nil {
		return nil, err
	}

	prebuildHash, err := config.CalculatePrebuildHash(parsedConfig.Config, options.Platform, arch, docker.GetContextPath(parsedConfig.Config), dockerfilePath, dockerfileContent, k.Log)
	if err != nil {
		return nil, err
	}

	// check if there is a prebuild image
	if !options.ForceBuild {
		devPodCustomizations := config.GetDevPodCustomizations(parsedConfig.Config)
		if options.Repository != "" {
			options.PrebuildRepositories = append(options.PrebuildRepositories, options.Repository)
		}
		if k.config.BuildRepository != "" {
			options.PrebuildRepositories = append(options.PrebuildRepositories, k.config.BuildRepository)
		}
		options.PrebuildRepositories = append(options.PrebuildRepositories, devPodCustomizations.PrebuildRepository...)
		k.Log.Debugf("Try to find prebuild image %s in repositories %s", prebuildHash, strings.Join(options.PrebuildRepositories, ","))
		for _, prebuildRepo := range options.PrebuildRepositories {
			prebuildImage := prebuildRepo + ":" + prebuildHash
			img, err := image.GetImage(prebuildImage)
			if err == nil && img != nil {
				// prebuild image found
				k.Log.Infof("Found existing prebuilt image %s", prebuildImage)

				// inspect image
				imageDetails, err := k.InspectImage(ctx, prebuildImage)
				if err != nil {
					return nil, errors.Wrap(err, "get image details")
				}

				return &config.BuildInfo{
					ImageDetails:  imageDetails,
					ImageMetadata: extendedBuildInfo.MetadataConfig,
					ImageName:     prebuildImage,
					PrebuildHash:  prebuildHash,
				}, nil
			} else if err != nil {
				k.Log.Debugf("Error trying to find prebuild image %s: %v", prebuildImage, err)
			}
		}
	}

	// check if prebuild
	if !options.SkipPush && options.Repository != "" {
		return nil, fmt.Errorf("you cannot use Kubernetes driver to prebuild images, please use docker instead")
	}

	// check if we shouldn't build
	if options.NoBuild {
		return nil, fmt.Errorf("you cannot build in this mode. Please run 'devpod up' to rebuild the container")
	}

	// get devcontainer id
	id, err := k.getID(labels)
	if err != nil {
		return nil, errors.Wrap(err, "get id")
	}

	// build pod image
	return k.buildPod(ctx, id, prebuildHash, dockerfilePath, dockerfileContent, parsedConfig, options, extendedBuildInfo)
}

func (k *kubernetesDriver) buildPod(
	ctx context.Context,
	id,
	prebuildHash,
	dockerfilePath,
	dockerfileContent string,
	parsedConfig *config.SubstitutedConfig,
	options config.BuildOptions,
	extendedBuildInfo *feature.ExtendedBuildInfo,
) (*config.BuildInfo, error) {
	if k.config.BuildRepository == "" {
		return nil, fmt.Errorf("please specify a build repository DevPod can push to as otherwise building in Kubernetes is not possible")
	}

	// get build options
	imageName := k.config.BuildRepository + ":" + prebuildHash
	buildOptions, deleteFolders, err := docker.CreateBuildOptions(
		dockerfilePath,
		dockerfileContent,
		parsedConfig,
		extendedBuildInfo,
		imageName,
		"",
		nil,
		prebuildHash,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, folder := range deleteFolders {
			_ = os.RemoveAll(folder)
		}
	}()
	if options.SkipPush {
		buildOptions.Load = false
		buildOptions.Push = false
	} else {
		buildOptions.Load = false
		buildOptions.Push = true
	}

	// get pod
	var pod *corev1.Pod
	if k.config.BuildkitPrivileged == "true" {
		pod = getPrivilegedBuildKitPod(id, k.config.BuildkitImage, parseResources(k.config.BuildkitResources, k.Log))
	} else {
		pod = getRootlessBuildKitPod(id, k.config.BuildkitImage, parseResources(k.config.BuildkitResources, k.Log))
	}

	// parse node selector
	if k.config.BuildkitNodeSelector != "" {
		pod.Spec.NodeSelector, err = parseLabels(k.config.BuildkitNodeSelector)
		if err != nil {
			return nil, fmt.Errorf("parse node selector: %w", err)
		}
	}

	// delete existing pod
	k.Log.Debugf("Delete existing build pod")
	err = k.deletePod(ctx, pod.Name)
	if err != nil {
		return nil, errors.Wrap(err, "delete existing pod")
	}

	// encode pod
	podRaw, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}

	// create the pod
	k.Log.Infof("Create build Pod '%s'", pod.Name)
	buf := &bytes.Buffer{}
	err = k.runCommand(ctx, []string{"create", "-f", "-"}, strings.NewReader(string(podRaw)), buf, buf)
	if err != nil {
		return nil, errors.Wrapf(err, "create pod: %s", buf.String())
	}
	defer func() {
		k.Log.Infof("Delete build Pod '%s'", pod.Name)
		err = k.deletePod(ctx, pod.Name)
		if err != nil {
			k.Log.Errorf("Error deleting build Pod '%s': %v", pod.Name, err)
		}
	}()

	// wait for pod running
	k.Log.Infof("Waiting for build Pod '%s' to come up...", pod.Name)
	_, err = k.waitPodRunning(ctx, pod.Name)
	if err != nil {
		return nil, err
	}

	// exec to container
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "create pipe")
	}
	defer stdinWriter.Close()
	defer stdoutWriter.Close()

	// create cancel context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	writer := k.Log.Writer(logrus.InfoLevel, false)
	defer writer.Close()
	go func() {
		defer stdoutWriter.Close()
		defer cancel()

		err = k.runCommand(cancelCtx, []string{"exec", "-i", pod.Name, "--", "buildctl", "dial-stdio"}, stdinReader, stdoutWriter, writer)
		if err != nil && !strings.Contains(err.Error(), "signal: killed") {
			k.Log.Errorf("Error dialing build kit container: %v", err)
		}
	}()

	// create build kit client
	buildKitClient, err := newBuildKitClient(cancelCtx, stdoutReader, stdinWriter)
	if err != nil {
		return nil, errors.Wrap(err, "create buildkit client")
	}
	defer buildKitClient.Close()

	// build
	k.Log.Infof("Start building image '%s'...", imageName)
	err = buildkit.Build(cancelCtx, buildKitClient, writer, options.Platform, buildOptions, k.Log)
	if err != nil {
		return nil, errors.Wrap(err, "build")
	}

	// check registry
	k.Log.Infof("Done building image '%s'", imageName)
	var imageDetails *config.ImageDetails
	if !options.SkipPush {
		imageDetails, err = k.InspectImage(ctx, imageName)
		if err != nil {
			return nil, errors.Wrap(err, "inspect image")
		}
	}

	return &config.BuildInfo{
		ImageName:     imageName,
		PrebuildHash:  prebuildHash,
		ImageDetails:  imageDetails,
		ImageMetadata: extendedBuildInfo.MetadataConfig,
	}, nil
}

func newBuildKitClient(ctx context.Context, reader io.Reader, writer io.WriteCloser) (*client.Client, error) {
	conn := stdio.NewStdioStream(reader, writer, false)
	return client.New(ctx, "", client.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return conn, nil
	}))
}

func getPrivilegedBuildKitPod(id, buildKitImage string, resources corev1.ResourceRequirements) *corev1.Pod {
	if buildKitImage == "" {
		buildKitImage = defaultBuildkitImage
	}

	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   id + "-" + "buildkit",
			Labels: DevPodLabels,
		},
		Spec: corev1.PodSpec{
			EnableServiceLinks: new(bool),
			Containers: []corev1.Container{
				{
					Name:      "buildkitd",
					Image:     buildKitImage,
					Resources: resources,
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"buildctl",
									"debug",
									"workers",
								},
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       30,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"buildctl",
									"debug",
									"workers",
								},
							},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       30,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{true}[0],
					},
				},
			},
		},
	}
}

func getRootlessBuildKitPod(id, buildKitImage string, resources corev1.ResourceRequirements) *corev1.Pod {
	if buildKitImage == "" {
		buildKitImage = defaultRootlessBuildkitImage
	}

	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   id + "-" + "buildkit",
			Labels: DevPodLabels,
			Annotations: map[string]string{
				"container.apparmor.security.beta.kubernetes.io/buildkitd": "unconfined",
			},
		},
		Spec: corev1.PodSpec{
			EnableServiceLinks: new(bool),
			Containers: []corev1.Container{
				{
					Name:  "buildkitd",
					Image: buildKitImage,
					Args: []string{
						"--oci-worker-no-process-sandbox",
					},
					Resources: resources,
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"buildctl",
									"debug",
									"workers",
								},
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       30,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"buildctl",
									"debug",
									"workers",
								},
							},
						},
						InitialDelaySeconds: 2,
						PeriodSeconds:       30,
					},
					SecurityContext: &corev1.SecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeUnconfined,
						},
						RunAsUser:  &[]int64{1000}[0],
						RunAsGroup: &[]int64{1000}[0],
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "buildkitd",
							MountPath: "/home/user/.local/share/buildkit",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "buildkitd",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func (k *kubernetesDriver) getClusterArchitecture(ctx context.Context) (string, error) {
	k.Log.Infof("Find out cluster architecture...")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := k.runCommand(ctx, []string{"run", "-i", "devpod-" + random.String(6), "-q", "--rm", "--restart=Never", "--image", k.helperImage(), "--", "sh"}, strings.NewReader("uname -a; exit 0"), stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("find out cluster architecture: %s %s %w", stdout.String(), stderr.String(), err)
	}

	unameOutput := stdout.String()
	if strings.Contains(unameOutput, "arm") || strings.Contains(unameOutput, "aarch") {
		return "arm64", nil
	}

	return "amd64", nil
}

func (k *kubernetesDriver) helperImage() string {
	if k.config.HelperImage != "" {
		return k.config.HelperImage
	}

	return "busybox"
}
