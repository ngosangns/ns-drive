package main

import "encoding/json"

type Command string

const (
	CommandStoped  Command = "command_stoped"
	CommandOutput  Command = "command_output"
	CommandStarted Command = "command_started"
	Error          Command = "error"
)

func (c Command) String() string {
	return string(c)
}

type CommandDTO struct {
	Command string  `json:"command"`
	Pid     *int    `json:"pid,omitempty"`
	Error   *string `json:"error,omitempty"`
	Task    *string `json:"task"`
}

func (c CommandDTO) ToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(c)
	if err != nil {
		return []byte{}, err
	}
	return jsonData, nil
}

func NewCommandStoppedDTO(pid int) CommandDTO {
	return CommandDTO{Command: CommandStoped.String(), Pid: &pid}
}

func NewCommandOutputDTO(pid int, output string) CommandDTO {
	return CommandDTO{Command: CommandOutput.String(), Pid: &pid, Error: &output}
}

func NewCommandErrorDTO(pid int, err error) CommandDTO {
	errorString := err.Error()
	return CommandDTO{Command: Error.String(), Pid: &pid, Error: &errorString}
}

func NewCommandStartedDTO(pid int, task string) CommandDTO {
	return CommandDTO{Command: CommandStarted.String(), Pid: &pid, Task: &task}
}
