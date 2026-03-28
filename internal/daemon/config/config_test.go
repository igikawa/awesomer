package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Run {
		t.Fatal("Run = true, want false")
	}
	if cfg.Tick != 3 {
		t.Fatalf("Tick = %d, want 3", cfg.Tick)
	}
	if cfg.CPULimit != 85 {
		t.Fatalf("CPULimit = %v, want 85", cfg.CPULimit)
	}
	if cfg.RAMLimit != 60 {
		t.Fatalf("RAMLimit = %v, want 60", cfg.RAMLimit)
	}
	if cfg.CPUQuota != 20 {
		t.Fatalf("CPUQuota = %v, want 20", cfg.CPUQuota)
	}
	if cfg.RAMQuota != "8G" {
		t.Fatalf("RAMQuota = %q, want 8G", cfg.RAMQuota)
	}
	if len(cfg.Whitelist) != 2 {
		t.Fatalf("len(Whitelist) = %d, want 2", len(cfg.Whitelist))
	}
	if cfg.Whitelist[0] != "systemd" || cfg.Whitelist[1] != "sshd" {
		t.Fatalf("Whitelist = %v, want [systemd sshd]", cfg.Whitelist)
	}
}
