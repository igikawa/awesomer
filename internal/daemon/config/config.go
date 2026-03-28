package config

type Config struct {
	Run  bool `yaml:"run"`
	Tick int  `yaml:"tick"`

	CPULimit float64 `yaml:"cpu_limit"`
	RAMLimit float64 `yaml:"ram_limit"`

	CPUQuota  float64  `yaml:"cpu_quota"`
	RAMQuota  string   `yaml:"ram_quota"`
	Whitelist []string `yaml:"whitelist"`
}

func DefaultConfig() Config {
	return Config{
		Run:       false,
		Tick:      3,
		CPULimit:  85,
		RAMLimit:  60,
		CPUQuota:  20,
		RAMQuota:  "8G",
		Whitelist: []string{"systemd", "sshd"},
	}
}
