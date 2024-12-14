package utils

import "desktop/backend/dto"

func NewCommandStoppedDTO(pid int) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandStoped.String(), Id: &pid}
}

func NewCommandOutputDTO(pid int, output string) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandOutput.String(), Id: &pid, Error: &output}
}

func NewCommandErrorDTO(pid int, err error) dto.CommandDTO {
	errorString := err.Error()
	return dto.CommandDTO{Command: dto.Error.String(), Id: &pid, Error: &errorString}
}

func NewCommandStartedDTO(pid int, task string) dto.CommandDTO {
	return dto.CommandDTO{Command: dto.CommandStarted.String(), Id: &pid, Task: &task}
}
