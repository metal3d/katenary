package generator

import (
	"katenary/helm"
	"katenary/logger"

	"github.com/compose-spec/compose-go/types"
)

// This function will try to yied deployment and services based on a service from the compose file structure.
func buildDeployment(name string, s *types.ServiceConfig, linked map[string]types.ServiceConfig, fileGeneratorChan HelmFileGenerator) {

	logger.Magenta(ICON_PACKAGE+" Generating deployment for ", name)
	deployment := helm.NewDeployment(name)

	newContainerForDeployment(name, name, deployment, s, fileGeneratorChan)

	// Add selectors
	selectors := buildSelector(name, s)
	selectors[helm.K+"/resource"] = "deployment"
	deployment.Spec.Selector = map[string]interface{}{
		"matchLabels": selectors,
	}
	deployment.Spec.Template.Metadata.Labels = selectors

	// Now, the linked services (same pod)
	for lname, link := range linked {
		newContainerForDeployment(name, lname, deployment, &link, fileGeneratorChan)
		// append ports and expose ports to the deployment,
		// to be able to generate them in the Service file
		if len(link.Ports) > 0 || len(link.Expose) > 0 {
			s.Ports = append(s.Ports, link.Ports...)
			s.Expose = append(s.Expose, link.Expose...)
		}
	}

	// Remove duplicates in volumes
	volumes := make([]map[string]interface{}, 0)
	done := make(map[string]bool)
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		name := vol["name"].(string)
		if _, ok := done[name]; ok {
			continue
		} else {
			done[name] = true
			volumes = append(volumes, vol)
		}
	}
	deployment.Spec.Template.Spec.Volumes = volumes

	// Then, create Services and possible Ingresses for ingress labels, "ports" and "expose" section
	if len(s.Ports) > 0 || len(s.Expose) > 0 {
		for _, s := range generateServicesAndIngresses(name, s) {
			if s != nil {
				fileGeneratorChan <- s
			}
		}
	}

	// add the volumes in Values
	if len(VolumeValues[name]) > 0 {
		AddValues(name, map[string]EnvVal{"persistence": VolumeValues[name]})
	}

	// the deployment is ready, give it
	fileGeneratorChan <- deployment

	// and then, we can say that it's the end
	fileGeneratorChan <- nil
}
