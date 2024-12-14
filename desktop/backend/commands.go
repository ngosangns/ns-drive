package backend

import (
	"desktop/backend/rclone"
	"desktop/backend/utils"
	"errors"
)

// Pull syncs from the client source path to the client destination path
func (a *App) Pull() error {
	return rclone.RunRcloneSync(a.oc, "pull")
}

// Push syncs from the client destination path to the client source path
func (a *App) Push() error {
	return rclone.RunRcloneSync(a.oc, "push")
}

func (a *App) StopCommand(pid int) {
	cmd, exists := utils.GetCmd(pid)
	if !exists {
		j, e := utils.NewCommandErrorDTO(pid, errors.New("command not found")).ToJSON()
		if e != nil {
			utils.LogError(e)
			return
		}
		a.oc <- j
		utils.LogError(errors.New("command not found"))
		return
	}

	if err := cmd.Process.Kill(); err != nil {
		j, e := utils.NewCommandErrorDTO(pid, err).ToJSON()
		if e != nil {
			utils.LogError(e)
			return
		}
		a.oc <- j
		utils.LogError(err)
	}

	res, err := utils.NewCommandStoppedDTO(pid).ToJSON()
	if err != nil {
		j, e := utils.NewCommandErrorDTO(pid, err).ToJSON()
		if e != nil {
			utils.LogError(e)
			return
		}
		a.oc <- j
		utils.LogError(err)
	}

	a.oc <- res
}
