package models

type Profile struct {
	Name          string   `json:"name"`
	From          string   `json:"from"`
	To            string   `json:"to"`
	IncludedPaths []string `json:"included_paths"`
	ExcludedPaths []string `json:"excluded_paths"`
	Bandwidth     int      `json:"bandwidth" default:"5M"`
	Parallel      int      `json:"parallel" default:"16"`
	BackupPath    string   `json:"backup_path"`
	CachePath     string   `json:"cache_path"`
}

type Profiles []Profile
