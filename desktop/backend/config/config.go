package rclone

type Config struct {
	DebugMode bool `env:"DEBUG_MODE" envDefault:"false"`
}
