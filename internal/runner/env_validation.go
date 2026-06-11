package runner

import (
	"sort"

	"go-chrome/internal/flow"
	"go-chrome/internal/template"
)

// MissingEnvVars returns environment variable references that cannot be
// resolved for enabled steps at or after startStep.
func MissingEnvVars(f *flow.Flow, startStep int, provider template.EnvProvider) []string {
	if f == nil {
		return nil
	}
	if startStep < 0 {
		startStep = 0
	}
	if startStep >= len(f.Steps) {
		return nil
	}

	required := make(map[string]struct{})
	for _, step := range f.Steps[startStep:] {
		if !step.Enabled {
			continue
		}
		addEnvVars(required, step.Target.Value)
		addEnvVars(required, step.Input.Text)
		addEnvVars(required, step.Note)
	}
	if len(required) == 0 {
		return nil
	}

	var missing []string
	for key := range required {
		if provider == nil {
			missing = append(missing, key)
			continue
		}
		if _, found, _ := provider.GetEnvValue(key); !found {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func addEnvVars(dst map[string]struct{}, text string) {
	for _, key := range template.ScanEnvVars(text) {
		dst[key] = struct{}{}
	}
}
