package rclone

import (
	"desktop/backend/models"
	"desktop/backend/utils"
	"os/exec"
)

// runRcloneSync runs the rclone sync command with the provided arguments
func RunRcloneSync(Oc chan []byte, args ...string) error {
	cmdArr := append([]string{"task"}, args...)
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)

	stdout := models.RcloneStdout{C: Oc, Pid: 0}
	cmd.Stdout = stdout
	cmd.Stderr = stdout

	e := cmd.Start()
	if e != nil {
		utils.LogError(e)
		j, _ := utils.NewCommandErrorDTO(0, e).ToJSON()
		Oc <- j
		return e
	}

	// pid := cmd.Process.Pid
	pid := utils.GetRandomPid()
	utils.AddCmd(pid, cmd)

	j, _ := utils.NewCommandStartedDTO(pid, cmdArr[1]).ToJSON()
	Oc <- j

	// Wait for the command to finish
	go func() {
		err := cmd.Wait()
		if err != nil {
			utils.LogError(err)
			j, _ := utils.NewCommandErrorDTO(pid, err).ToJSON()
			Oc <- j
		} else {
			j, _ := utils.NewCommandStoppedDTO(pid).ToJSON()
			Oc <- j
		}
		utils.RemoveCmd(pid)
	}()

	return e
}
