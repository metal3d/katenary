package generator

import (
	"fmt"
	"io/ioutil"
	"katenary/compose"
	"katenary/helm"
	"katenary/logger"
	"katenary/tools"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

// applyEnvMapLabel will get all LABEL_MAP_ENV to rebuild the env map with tpl.
func applyEnvMapLabel(s *types.ServiceConfig, c *helm.Container) {

	locker.Lock()
	defer locker.Unlock()
	mapenv, ok := s.Labels[helm.LABEL_MAP_ENV]
	if !ok {
		return
	}

	// the mapenv is a YAML string
	var envmap map[string]EnvVal
	err := yaml.Unmarshal([]byte(mapenv), &envmap)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		return
	}

	// add in envmap
	for k, v := range envmap {
		vstring := fmt.Sprintf("%v", v)
		s.Environment[k] = &vstring
		touched := false
		if c.Env != nil {
			c.Env = make([]*helm.Value, 0)
		}
		for _, env := range c.Env {
			if env.Name == k {
				env.Value = v
				touched = true
			}
		}
		if !touched {
			c.Env = append(c.Env, &helm.Value{Name: k, Value: v})
		}
	}
}

// readEnvFile read environment file and add to the values.yaml map.
func readEnvFile(envfilename string) map[string]EnvVal {
	env := make(map[string]EnvVal)
	content, err := ioutil.ReadFile(envfilename)
	if err != nil {
		logger.ActivateColors = true
		logger.Red(err.Error())
		logger.ActivateColors = false
		os.Exit(2)
	}
	// each value is on a separate line with KEY=value
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			kv := strings.SplitN(line, "=", 2)
			env[kv[0]] = kv[1]
		}
	}
	return env
}

// prepareEnvFromFiles generate configMap or secrets from environment files.
func prepareEnvFromFiles(name string, s *types.ServiceConfig, container *helm.Container, fileGeneratorChan HelmFileGenerator) {

	// prepare secrets
	secretsFiles := make([]string, 0)
	if v, ok := s.Labels[helm.LABEL_ENV_SECRET]; ok {
		secretsFiles = strings.Split(v, ",")
	}

	var secretVars []string
	if v, ok := s.Labels[helm.LABEL_SECRETVARS]; ok {
		secretVars = strings.Split(v, ",")
	}

	for i, s := range secretVars {
		secretVars[i] = strings.TrimSpace(s)
	}

	// manage environment files (env_file in compose)
	for _, envfile := range s.EnvFile {
		f := tools.PathToName(envfile)
		f = strings.ReplaceAll(f, ".env", "")
		isSecret := false
		for _, s := range secretsFiles {
			s = strings.TrimSpace(s)
			if s == envfile {
				isSecret = true
			}
		}
		var store helm.InlineConfig
		if !isSecret {
			logger.Bluef(ICON_CONF+" Generating configMap from %s\n", envfile)
			store = helm.NewConfigMap(name, envfile)
		} else {
			logger.Bluef(ICON_SECRET+" Generating secret from %s\n", envfile)
			store = helm.NewSecret(name, envfile)
		}

		envfile = filepath.Join(compose.GetCurrentDir(), envfile)
		if err := store.AddEnvFile(envfile, secretVars); err != nil {
			logger.ActivateColors = true
			logger.Red(err.Error())
			logger.ActivateColors = false
			os.Exit(2)
		}

		section := "configMapRef"
		if isSecret {
			section = "secretRef"
		}

		container.EnvFrom = append(container.EnvFrom, map[string]map[string]string{
			section: {
				"name": store.Metadata().Name,
			},
		})

		// read the envfile and remove them from the container environment or secret
		envs := readEnvFile(envfile)
		for varname := range envs {
			if !isSecret {
				// remove varname from container
				for i, s := range container.Env {
					if s.Name == varname {
						container.Env = append(container.Env[:i], container.Env[i+1:]...)
						i--
					}
				}
			}
		}

		if store != nil {
			fileGeneratorChan <- store.(HelmFile)
		}
	}
}
