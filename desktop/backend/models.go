package backend

type RcloneStdout struct {
	pid int
	c   chan []byte
}

// Support command output
func (r RcloneStdout) Write(p []byte) (n int, err error) {
	j, e := NewCommandOutputDTO(r.pid, string(p)).ToJSON()
	if e != nil {
		return 0, e
	}
	r.c <- j
	return len(p), nil
}
