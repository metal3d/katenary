package generator

import (
	"fmt"
	"katenary/helm"
	"log"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

const (
	cronMulti    = `pods=$(kubectl get pods --selector=%s/component=%s,%s/resource=deployment -o jsonpath='{.items[*].metadata.name}')`
	cronMultiCmd = `
for pod in $pods; do
    kubectl exec -i $pod -c %s -- sh -c '%s'
done`
	cronSingle = `pod=$(kubectl get pods --selector=%s/component=%s,%s/resource=deployment -o jsonpath='{.items[0].metadata.name}')`
	cronCmd    = `
kubectl exec -i $pod -c %s -- sh -c '%s'`
)

type CronDef struct {
	Command  string `yaml:"command"`
	Schedule string `yaml:"schedule"`
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
	log.Println(crons)

	// create a serviceAccount
	sa := helm.NewServiceAccount(deployName)
	// create a role
	role := helm.NewCronRole(deployName)

	// create a roleBinding
	roleBinding := helm.NewRoleBinding(deployName, sa, role)

	// make generation
	fileGeneratorChan <- sa
	fileGeneratorChan <- role
	fileGeneratorChan <- roleBinding

	// create crontabs
	for _, cron := range crons {
		var cmd, podget string
		if cron.Multi {
			podget = cronMulti
			cmd = cronMultiCmd
		} else {
			podget = cronSingle
			cmd = cronCmd
		}
		podget = fmt.Sprintf(podget, helm.K, deployName, helm.K)
		cmd = fmt.Sprintf(cmd, s.Name, cron.Command)
		cmd = podget + cmd

		cronTab := helm.NewCrontab(
			deployName,
			"bitnami/kubectl",
			cmd,
			cron.Schedule,
			sa,
		)
		// add crontab
		fileGeneratorChan <- cronTab
	}

	return
}
