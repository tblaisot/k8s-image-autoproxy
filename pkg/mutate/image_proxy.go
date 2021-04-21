package mutate

import (
	"fmt"

	dockerparser "github.com/novln/docker-parser"
	dockerparserdocker "github.com/novln/docker-parser/docker"
	core "k8s.io/api/core/v1"
)

func patchImageWithProxy(options Options, image string) (newimage string, err error) {
	newimage = image
	parsed, err := dockerparser.Parse(image)
	if err != nil {
		return
	}
	if dockerparserdocker.DefaultHostname == parsed.Registry() {
		newimage = fmt.Sprintf("%s/%s", options.proxy, parsed.Name())
	}
	return
}

func proxyContainer(options Options, container core.Container, pathPrefix string, patches []map[string]string) []map[string]string {
	if options.enabled {
		s, _ := patchImageWithProxy(options, container.Image)
		if s != container.Image {
			patch := map[string]string{
				"op":    "replace",
				"path":  fmt.Sprintf("%s/image", pathPrefix),
				"value": s,
			}
			patches = append(patches, patch)
		}
	}
	return patches
}

func proxyPodSpec(options Options, podSpec core.PodSpec, pathPrefix string, patches []map[string]string) []map[string]string {
	if options.enabled {
		for i, c := range podSpec.Containers {
			patches = proxyContainer(options, c, fmt.Sprintf("%s/containers/%d", pathPrefix, i), patches)
		}

		for i, c := range podSpec.InitContainers {
			patches = proxyContainer(options, c, fmt.Sprintf("%s/initContainers/%d", pathPrefix, i), patches)
		}
	}
	return patches
}
