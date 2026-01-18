package orchestrator

import "testing"

func TestRegistry_RegisterDuplicateName(t *testing.T) {
	reg := NewRegistry()

	primary := &Container{ID: "c1", Name: "alpha"}
	if err := reg.Register(primary); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	conflict := &Container{ID: "c2", Name: "alpha"}
	if err := reg.Register(conflict); err == nil {
		t.Fatalf("expected duplicate name error, got nil")
	}

	existing, ok := reg.GetByName("alpha")
	if !ok || existing.ID != "c1" {
		t.Fatalf("registry name index was overwritten, expected c1 got %v", existing)
	}
}

func TestRegistry_RegisterSameNameSameContainer(t *testing.T) {
	reg := NewRegistry()

	container := &Container{ID: "c3", Name: "beta"}
	if err := reg.Register(container); err != nil {
		t.Fatalf("initial register failed: %v", err)
	}

	updated := &Container{ID: "c3", Name: "beta"}
	if err := reg.Register(updated); err != nil {
		t.Fatalf("re-registering same container should succeed: %v", err)
	}
}
