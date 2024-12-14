package models

import "desktop/backend/utils"

type RcloneStdout struct {
	Pid int
	C   chan []byte
}

// Support command output
func (r RcloneStdout) Write(p []byte) (n int, err error) {
	j, e := utils.NewCommandOutputDTO(r.Pid, string(p)).ToJSON()
	if e != nil {
		return 0, e
	}
	r.C <- j
	return len(p), nil
}
