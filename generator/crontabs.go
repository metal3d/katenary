package generator

import (
	"fmt"
	"katenary/helm"
	"katenary/logger"
	"log"

	"github.com/alessio/shellescape"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

const (
	cronMulti    = `pods=$(kubectl get pods --selector=%s/component=%s,%s/resource=deployment -o jsonpath='{.items[*].metadata.name}')`
	cronMultiCmd = `
for pod in $pods; do
    kubectl exec -i $pod -c %s -- sh -c %s
done`
	cronSingle = `pod=$(kubectl get pods --selector=%s/component=%s,%s/resource=deployment -o jsonpath='{.items[0].metadata.name}')`
	cronCmd    = `
kubectl exec -i $pod -c %s -- sh -c %s`
)

type CronDef struct {
	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
	Image    string `yaml:"image"`
	Multi    bool   `yaml:"allPods,omitempty"`
}

func buildCrontab(deployName string, deployment *helm.Deployment, s *types.ServiceConfig, fileGeneratorChan HelmFileGenerator) {
	// get the cron label from the service
	var crondef string
	var ok bool
	if crondef, ok = s.Labels[helm.LABEL_CRON]; !ok {
		return
	}

	// parse yaml
	crons := []CronDef{}
	err := yaml.Unmarshal([]byte(crondef), &crons)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if len(crons) == 0 {
		return
	}

	// create a serviceAccount
	sa := helm.NewServiceAccount(deployName)
	// create a role
	role := helm.NewCronRole(deployName)

	// create a roleBinding
	roleBinding := helm.NewRoleBinding(deployName, sa, role)

	// make generation
	logger.Magenta(ICON_RBAC, "Generating ServiceAccount, Role and RoleBinding for cron jobs", deployName)
	fileGeneratorChan <- sa
	fileGeneratorChan <- role
	fileGeneratorChan <- roleBinding

	index := len(crons) - 1 // will be 0 when there is only one cron - made to name crons

	// create crontabs
	for _, cron := range crons {
		escaped := shellescape.Quote(cron.Command)
		var cmd, podget string
		if cron.Multi {
			podget = cronMulti
			cmd = cronMultiCmd
		} else {
			podget = cronSingle
			cmd = cronCmd
		}
		podget = fmt.Sprintf(podget, helm.K, deployName, helm.K)
		cmd = fmt.Sprintf(cmd, s.Name, escaped)
		cmd = podget + cmd

		if cron.Image == "" {
			cron.Image = `bitnami/kubectl:{{ printf "%s.%s" .Capabilities.KubeVersion.Major .Capabilities.KubeVersion.Minor }}`
		}

		name := deployName
		if index > 0 {
			name = fmt.Sprintf("%s-%d", deployName, index)
			index++
		}

		// add crontab
		suffix := ""
		if index > 0 {
			suffix = fmt.Sprintf("%d", index)
		}
		cronTab := helm.NewCrontab(
			name,
			cron.Image,
			cmd,
			cron.Schedule,
			sa,
		)
		logger.Magenta(ICON_CRON, "Generating crontab", deployName, suffix)
		fileGeneratorChan <- cronTab
	}

	return
}
