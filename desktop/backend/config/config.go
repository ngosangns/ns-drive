package rclone

type Config struct {
	FromFs     string `env:"FROM_FS"`
	ToFs       string `env:"TO_FS"`
	FilterFile string `env:"FILTER_FILE"`
	Bandwidth  string `env:"BANDWIDTH" envDefault:"99999M"`
	Parallel   int    `env:"PARALLEL" envDefault:"1"`
}
