package info

import "testing"

func TestAPIJailLifecycle(t *testing.T) {
	api := NewAPI()

	if api.InJail(10) {
		t.Fatal("InJail(10) = true, want false for unknown pid")
	}

	api.SetJail(10)
	if !api.InJail(10) {
		t.Fatal("InJail(10) = false, want true after SetJail")
	}

	api.DeleteFromJail(10)
	if api.InJail(10) {
		t.Fatal("InJail(10) = true, want false after DeleteFromJail")
	}
}

func TestAPIPIDsReturnsSnapshot(t *testing.T) {
	api := NewAPI()

	api.SetJail(10)
	api.SetJail(20)

	pids := api.PIDs()
	if len(pids) != 2 {
		t.Fatalf("len(PIDs()) = %d, want 2", len(pids))
	}
}
