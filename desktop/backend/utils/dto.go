package utils

import "desktop/backend/dto"

func NewCommandStoppedDTO(pid int) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandStoped.String(), Pid: &pid}
}

func NewCommandOutputDTO(pid int, output string) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandOutput.String(), Pid: &pid, Error: &output}
}

func NewCommandErrorDTO(pid int, err error) dto.CommandDTO {
	errorString := err.Error()
	return dto.CommandDTO{Command: dto.Error.String(), Pid: &pid, Error: &errorString}
}

func NewCommandStartedDTO(pid int, task string) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandStarted.String(), Pid: &pid, Task: &task}
}
