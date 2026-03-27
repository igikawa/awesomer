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
