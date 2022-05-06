package helm

import "sync"

var (
	made   = make(map[string]bool)
	locker = sync.Mutex{}
)

// ResetMadePVC resets the cache of made PVCs.
// Useful in tests only.
func ResetMadePVC() {
	locker.Lock()
	defer locker.Unlock()
	made = make(map[string]bool)
}

// Storage is a struct for a PersistentVolumeClaim.
type Storage struct {
	*K8sBase `yaml:",inline"`
	Spec     *PVCSpec
}

// NewPVC creates a new PersistentVolumeClaim object.
func NewPVC(name, storageName string) *Storage {
	locker.Lock()
	defer locker.Unlock()
	if _, ok := made[name+storageName]; ok {
		return nil
	}
	made[name+storageName] = true
	pvc := &Storage{}
	pvc.K8sBase = NewBase()
	pvc.K8sBase.Kind = "PersistentVolumeClaim"
	pvc.K8sBase.Metadata.Labels[K+"/pvc-name"] = storageName
	pvc.K8sBase.ApiVersion = "v1"
	pvc.K8sBase.Metadata.Name = ReleaseNameTpl + "-" + storageName
	pvc.K8sBase.Metadata.Labels[K+"/component"] = name
	pvc.Spec = &PVCSpec{
		Resouces: map[string]interface{}{
			"requests": map[string]string{
				"storage": "{{ .Values." + name + ".persistence." + storageName + ".capacity }}",
			},
		},
		AccessModes: []string{"ReadWriteOnce"},
	}
	return pvc
}

// PVCSpec is a struct for a PersistentVolumeClaim spec.
type PVCSpec struct {
	Resouces    map[string]interface{} `yaml:"resources"`
	AccessModes []string               `yaml:"accessModes"`
}
