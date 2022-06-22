package generator

import (
	"katenary/helm"
	"katenary/logger"
	"katenary/tools"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

var (
	// VolumeValues is the map of volumes for each deployment
	// containing volume configuration
	VolumeValues = make(map[string]map[string]map[string]EnvVal)
)

// AddVolumeValues add a volume to the values.yaml map for the given deployment name.
func AddVolumeValues(deployment string, volname string, values map[string]EnvVal) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := VolumeValues[deployment]; !ok {
		VolumeValues[deployment] = make(map[string]map[string]EnvVal)
	}
	VolumeValues[deployment][volname] = values
}

// addVolumeFrom takes the LABEL_VOLUMEFROM to get volumes from another container. This can only work with
// container that has got LABEL_SAMEPOD as we need to get the volumes from another container in the same deployment.
func addVolumeFrom(deployment *helm.Deployment, container *helm.Container, s *types.ServiceConfig) {
	labelfrom, ok := s.Labels[helm.LABEL_VOLUMEFROM]
	if !ok {
		return
	}

	// decode Yaml from the label
	var volumesFrom map[string]map[string]string
	err := yaml.Unmarshal([]byte(labelfrom), &volumesFrom)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		return
	}

	// for each declared volume "from", we will find it from the deployment volumes and add it to the container.
	// Then, to avoid duplicates, we will remove it from the ServiceConfig object.
	for name, volumes := range volumesFrom {
		for volumeName := range volumes {
			initianame := volumeName
			volumeName = tools.PathToName(volumeName)
			// get the volume from the deployment container "name"
			var ctn *helm.Container
			for _, c := range deployment.Spec.Template.Spec.Containers {
				if c.Name == name {
					ctn = c
					break
				}
			}
			if ctn == nil {
				logger.ActivateColors = true
				logger.Redf("VolumeFrom: container %s not found", name)
				logger.ActivateColors = false
				continue
			}
			// get the volume from the container
			for _, v := range ctn.VolumeMounts {
				switch v := v.(type) {
				case map[string]interface{}:
					if v["name"] == volumeName {
						if container.VolumeMounts == nil {
							container.VolumeMounts = make([]interface{}, 0)
						}
						// make a copy of the volume mount and then add it to the VolumeMounts
						var mountpoint = make(map[string]interface{})
						for k, v := range v {
							mountpoint[k] = v
						}
						container.VolumeMounts = append(container.VolumeMounts, mountpoint)

						// remove the volume from the ServiceConfig
						for i, vol := range s.Volumes {
							if vol.Source == initianame {
								s.Volumes = append(s.Volumes[:i], s.Volumes[i+1:]...)
								i--
								break
							}
						}
					}
				}
			}
		}
	}
}

// prepareVolumes add the volumes of a service.
func prepareVolumes(
	deployment, name string,
	s *types.ServiceConfig,
	container *helm.Container,
	fileGeneratorChan HelmFileGenerator) []map[string]interface{} {

	volumes := make([]map[string]interface{}, 0)
	mountPoints := make([]interface{}, 0)
	configMapsVolumes := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_VOL_CM]; ok {
		configMapsVolumes = strings.Split(v, ",")
		for i, cm := range configMapsVolumes {
			configMapsVolumes[i] = strings.TrimSpace(cm)
		}
	}

	for _, vol := range s.Volumes {

		volname := vol.Source
		volepath := vol.Target

		if volname == "" {
			logger.ActivateColors = true
			logger.Yellowf("Warning, volume source to %s is empty for %s -- skipping\n", volepath, name)
			logger.ActivateColors = false
			continue
		}

		isConfigMap := false
		for _, cmVol := range configMapsVolumes {
			if tools.GetRelPath(volname) == cmVol {
				isConfigMap = true
				break
			}
		}

		// local volume cannt be mounted
		if !isConfigMap && (strings.HasPrefix(volname, ".") || strings.HasPrefix(volname, "/")) {
			logger.ActivateColors = true
			logger.Redf("You cannot, at this time, have local volume in %s deployment\n", name)
			logger.ActivateColors = false
			continue
		}
		if isConfigMap {
			// check if the volname path points on a file, if so, we need to add subvolume to the interface
			stat, err := os.Stat(volname)
			if err != nil {
				logger.ActivateColors = true
				logger.Redf("An error occured reading volume path %s\n", err.Error())
				logger.ActivateColors = false
				continue
			}
			pointToFile := ""
			if !stat.IsDir() {
				pointToFile = filepath.Base(volname)
			}

			// the volume is a path and it's explicitally asked to be a configmap in labels
			cm := buildConfigMapFromPath(name, volname)
			cm.K8sBase.Metadata.Name = helm.ReleaseNameTpl + "-" + name + "-" + tools.PathToName(volname)

			// build a configmapRef for this volume
			volname := tools.PathToName(volname)
			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"configMap": map[string]string{
					"name": cm.K8sBase.Metadata.Name,
				},
			})
			if len(pointToFile) > 0 {
				mountPoints = append(mountPoints, map[string]interface{}{
					"name":      volname,
					"mountPath": volepath,
					"subPath":   pointToFile,
				})
			} else {
				mountPoints = append(mountPoints, map[string]interface{}{
					"name":      volname,
					"mountPath": volepath,
				})
			}
			if cm != nil {
				fileGeneratorChan <- cm
			}
		} else {
			// It's a Volume. Mount this from PVC to declare.

			volname = strings.ReplaceAll(volname, "-", "")

			isEmptyDir := false
			for _, v := range EmptyDirs {
				v = strings.ReplaceAll(v, "-", "")
				if v == volname {
					volumes = append(volumes, map[string]interface{}{
						"name":     volname,
						"emptyDir": map[string]string{},
					})
					mountPoints = append(mountPoints, map[string]interface{}{
						"name":      volname,
						"mountPath": volepath,
					})
					container.VolumeMounts = append(container.VolumeMounts, mountPoints...)
					isEmptyDir = true
					break
				}
			}
			if isEmptyDir {
				continue
			}

			volumes = append(volumes, map[string]interface{}{
				"name": volname,
				"persistentVolumeClaim": map[string]string{
					"claimName": helm.ReleaseNameTpl + "-" + volname,
				},
			})
			mountPoints = append(mountPoints, map[string]interface{}{
				"name":      volname,
				"mountPath": volepath,
			})

			logger.Yellow(ICON_STORE+" Generate volume values", volname, "for container named", name, "in deployment", deployment)
			AddVolumeValues(deployment, volname, map[string]EnvVal{
				"enabled":  false,
				"capacity": "1Gi",
			})

			if pvc := helm.NewPVC(deployment, volname); pvc != nil {
				fileGeneratorChan <- pvc
			}
		}
	}
	// add the volume in the container and return the volume definition to add in Deployment
	container.VolumeMounts = append(container.VolumeMounts, mountPoints...)
	return volumes
}
