package backend

import "errors"

// Pull syncs from the client source path to the client destination path
func (a *App) Pull() error {
	return a.RunRcloneSync("pull")
}

// Push syncs from the client destination path to the client source path
func (a *App) Push() error {
	return a.RunRcloneSync("push")
}

func (a *App) StopCommand(pid int) {
	cmd, exists := GetCmd(pid)
	if !exists {
		j, e := NewCommandErrorDTO(pid, errors.New("Command not found")).ToJSON()
		if e != nil {
			a.LogError(e)
			return
		}
		Oc <- j
		a.LogError(errors.New("Command not found"))
		return
	}

	if err := cmd.Process.Kill(); err != nil {
		j, e := NewCommandErrorDTO(pid, err).ToJSON()
		if e != nil {
			a.LogError(e)
			return
		}
		Oc <- j
		a.LogError(err)
	}

	res, err := NewCommandStoppedDTO(pid).ToJSON()
	if err != nil {
		j, e := NewCommandErrorDTO(pid, err).ToJSON()
		if e != nil {
			a.LogError(e)
			return
		}
		Oc <- j
		a.LogError(err)
	}

	Oc <- res
}
