package helm

import (
	"time"

	"github.com/compose-spec/compose-go/types"
)

// Probe is a struct that can be used to create a Liveness or Readiness probe.
type Probe struct {
	HttpGet      *HttpGet `yaml:"httpGet,omitempty"`
	Exec         *Exec    `yaml:"exec,omitempty"`
	TCP          *TCP     `yaml:"tcp,omitempty"`
	Period       float64  `yaml:"periodSeconds"`
	InitialDelay float64  `yaml:"initialDelaySeconds"`
	Success      uint64   `yaml:"successThreshold"`
	Failure      uint64   `yaml:"failureThreshold"`
}

// Create a new Probe object that can be apply to HttpProbe or TCPProbe.
func NewProbe(period, initialDelaySeconds float64, success, failure uint64) *Probe {
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

// NewProbeWithDuration creates a new Probe object with the given duration from types.
func NewProbeWithDuration(period, initialDelaySeconds *types.Duration, success, failure *uint64) *Probe {

	if period == nil {
		d := types.Duration(0 * time.Second)
		period = &d
	}

	if initialDelaySeconds == nil {
		d := types.Duration(0 * time.Second)
		initialDelaySeconds = &d
	}

	if success == nil {
		s := uint64(0)
		success = &s
	}

	if failure == nil {
		f := uint64(0)
		failure = &f
	}

	p, err := time.ParseDuration(period.String())
	if err != nil {
		p = time.Second * 10
	}

	i, err := time.ParseDuration(initialDelaySeconds.String())
	if err != nil {
		i = time.Second * 0
	}

	return NewProbe(p.Seconds(), i.Seconds(), *success, *failure)

}

// NewProbeFromService creates a new Probe object from a ServiceConfig.
func NewProbeFromService(s *types.ServiceConfig) *Probe {
	if s == nil || s.HealthCheck == nil {
		return NewProbe(0, 0, 0, 0)
	}

	return NewProbeWithDuration(s.HealthCheck.Interval, s.HealthCheck.StartPeriod, nil, s.HealthCheck.Retries)

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
