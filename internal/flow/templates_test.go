package flow

import (
	"strings"
	"testing"
)

func TestListBuiltinTemplatesNotEmpty(t *testing.T) {
	temps := ListBuiltinTemplates()
	if len(temps) == 0 {
		t.Fatalf("expected at least one built-in template")
	}
	seenIDs := map[string]bool{}
	for _, tpl := range temps {
		if tpl.ID == "" {
			t.Errorf("template has empty ID: %+v", tpl)
		}
		if seenIDs[tpl.ID] {
			t.Errorf("duplicate template ID: %s", tpl.ID)
		}
		seenIDs[tpl.ID] = true
		if tpl.Name == "" {
			t.Errorf("template %q has empty name", tpl.ID)
		}
		if tpl.Factory == nil {
			t.Errorf("template %q has nil factory", tpl.ID)
		}
	}
}

func TestListBuiltinTemplatesReturnsCopy(t *testing.T) {
	a := ListBuiltinTemplates()
	b := ListBuiltinTemplates()
	a[0].Name = "MUTATED"
	if b[0].Name == "MUTATED" {
		t.Errorf("ListBuiltinTemplates did not return a defensive copy")
	}
}

func TestBuiltinTemplateFactoriesProduceFreshFlows(t *testing.T) {
	for _, tpl := range ListBuiltinTemplates() {
		t.Run(tpl.ID, func(t *testing.T) {
			f1 := tpl.Factory()
			f2 := tpl.Factory()
			if f1 == f2 {
				t.Errorf("factory returned the same flow twice: %p", f1)
			}
			if f1.ID == f2.ID {
				t.Errorf("factory returned the same flow ID twice: %s", f1.ID)
			}
			if len(f1.Steps) == 0 {
				t.Errorf("factory produced a flow with no steps: %s", tpl.ID)
			}
		})
	}
}

func TestFindBuiltinTemplateKnownAndUnknown(t *testing.T) {
	temps := ListBuiltinTemplates()
	if len(temps) == 0 {
		t.Skip("no built-in templates to find")
	}
	known := temps[0].ID
	if _, ok := FindBuiltinTemplate(known); !ok {
		t.Errorf("FindBuiltinTemplate(%q) returned not found", known)
	}
	if _, ok := FindBuiltinTemplate("does-not-exist"); ok {
		t.Errorf("FindBuiltinTemplate returned a hit for an unknown ID")
	}
}

func TestBuiltinTemplateFactoryProducesValidateFlow(t *testing.T) {
	for _, tpl := range ListBuiltinTemplates() {
		t.Run(tpl.ID, func(t *testing.T) {
			f := tpl.Factory()
			if err := Validate(f); err != nil {
				t.Errorf("factory produced an invalid flow: %v", err)
			}
			// Factory output must always include a navigate step so the
			// flow can be run end-to-end.
			hasNav := false
			for _, s := range f.Steps {
				if s.Type == StepNavigate {
					hasNav = true
					break
				}
			}
			if !hasNav {
				t.Errorf("template %q flow has no navigate step", tpl.ID)
			}
		})
	}
}

func TestBuiltinTemplateNamesAreUserFacing(t *testing.T) {
	for _, tpl := range ListBuiltinTemplates() {
		if strings.TrimSpace(tpl.Name) == "" {
			t.Errorf("template %q has empty name", tpl.ID)
		}
	}
}
