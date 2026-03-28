package config

type UIConfig struct {
	TableWidth               int    `yaml:"table_width"`
	InfoWidth                int    `yaml:"info_width"`
	BorderColor              string `yaml:"border_color"`
	ActiveBorderColor        string `yaml:"active_border_color"`
	SelectionTextColor       string `yaml:"selection_text_color"`
	SelectionBackgroundColor string `yaml:"selection_background_color"`
}

func DefaultUIConfig() UIConfig {
	return UIConfig{
		TableWidth:               0,
		InfoWidth:                72,
		BorderColor:              "102",
		ActiveBorderColor:        "62",
		SelectionTextColor:       "229",
		SelectionBackgroundColor: "57",
	}
}
