package helm

// Probe is a struct that can be used to create a Liveness or Readiness probe.
type Probe struct {
	HttpGet      *HttpGet `yaml:"httpGet,omitempty"`
	Exec         *Exec    `yaml:"exec,omitempty"`
	TCP          *TCP     `yaml:"tcp,omitempty"`
	Period       int      `yaml:"periodSeconds"`
	Success      int      `yaml:"successThreshold"`
	Failure      int      `yaml:"failureThreshold"`
	InitialDelay int      `yaml:"initialDelaySeconds"`
}

// Create a new Probe object that can be apply to HttpProbe or TCPProbe.
func NewProbe(period, initialDelaySeconds, success, failure int) *Probe {
	probe := &Probe{
		Period:       period,
		Success:      success,
		Failure:      failure,
		InitialDelay: initialDelaySeconds,
	}

	// fix default values from
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
	if period == 0 {
		probe.Period = 10
	}
	if success == 0 {
		probe.Success = 1
	}
	if failure == 0 {
		probe.Failure = 3
	}
	return probe
}

// HttpGet is a Probe configuration to check http health.
type HttpGet struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

// Execis a Probe configuration to check exec health.
type Exec struct {
	Command []string `yaml:"command"`
}

// TCP is a Probe configuration to check tcp health.
type TCP struct {
	Port int `yaml:"port"`
}
