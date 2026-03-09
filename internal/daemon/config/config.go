package config

type Config struct {
	Run  bool `env:"DAEMON" env-default:"false"`
	Tick int  `env:"DAEMON_TICK" env-default:"3"`

	CPULimit float64 `env:"DAEMON_CPU_LIMIT" env-default:"85"`
	RAMLimit float64 `env:"DAEMON_RAM_LIMIT" env-default:"60"`

	CPUQuota float64 `env:"DAEMON_CPU_QUOTA" env-default:"20"`
	RAMQuota string  `env:"DAEMON_RAM_QUOTA" env-default:"8G"`
}
