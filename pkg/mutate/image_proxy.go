package mutate

import (
	"errors"
	"fmt"
	dockerparser "github.com/novln/docker-parser"
	dockerparserdocker "github.com/novln/docker-parser/docker"
	core "k8s.io/api/core/v1"
)

// ErrNotIngress will be used when the validating object is not an ingress.
var ErrNotSupported = errors.New("object is not supported")

func patchWithProxy(image string, proxy string) (newimage string, err error) {
	newimage = image
	parsed, err := dockerparser.Parse(image)
	if err != nil {
		return
	}
	if dockerparserdocker.DefaultHostname == parsed.Registry() {
		newimage = fmt.Sprintf("%s/%s", proxy, parsed.Name())
	}
	return
}

func proxyContainer(container core.Container, proxy string, pathPrefix string, patches []map[string]string) []map[string]string {
	s, _ := patchWithProxy(container.Image, proxy)
	patch := map[string]string{
		"op":    "replace",
		"path":  fmt.Sprintf("%s/image", pathPrefix),
		"value": s,
	}
	patches = append(patches, patch)

	return patches
}

func proxyPodSpec(podSpec core.PodSpec, proxy string, pathPrefix string, patches []map[string]string) []map[string]string {

	for i, c := range podSpec.Containers {
		patches = proxyContainer(c, proxy, fmt.Sprintf("%s/containers/%d", pathPrefix, i), patches)
	}

	for i, c := range podSpec.InitContainers {
		patches = proxyContainer(c, proxy, fmt.Sprintf("%s/initContainers/%d", pathPrefix, i), patches)
	}

	return patches
}
