package config

type Config struct {
	DebugMode       bool   `env:"DEBUG_MODE" envDefault:"false"`
	ProfileFilePath string `env:"PROFILE_FILE_PATH"`
	ResyncFilePath  string `env:"RESYNC_FILE_PATH"`
	RcloneFilePath  string `env:"RCLONE_FILE_PATH"`
}
