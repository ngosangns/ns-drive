package backend

import (
	"context"
	"desktop/backend/rclone"
	"desktop/backend/utils"
	"errors"
	"log"
	"time"
)

// Pull syncs from the client source path to the client destination path
func (a *App) Pull() int {
	return a.Sync("pull")
}

// Push syncs from the client destination path to the client source path
func (a *App) Push() int {
	return a.Sync("push")
}

func (a *App) Sync(task string) int {
	id := time.Now().Nanosecond()

	ctx, cancel := context.WithCancel(context.Background())

	config, err := rclone.LoadConfigFromEnv()
	if utils.HandleError(err, "", nil, nil) != nil {
		j, _ := utils.NewCommandErrorDTO(id, err).ToJSON()
		a.oc <- j
		cancel()
		return 0
	}

	ctx, err = rclone.InitConfig(ctx)
	if utils.HandleError(err, "", nil, nil) != nil {
		j, _ := utils.NewCommandErrorDTO(id, err).ToJSON()
		a.oc <- j
		cancel()
		return 0
	}

	outLog := make(chan string)
	utils.AddCmd(id, func() {
		close(outLog)
		cancel()
	})

	go func() {
		for {
			logEntry, ok := <-outLog
			if !ok { // channel is closed
				break
			}
			j, _ := utils.NewCommandOutputDTO(id, logEntry).ToJSON()
			a.oc <- j
		}
	}()

	go func() {
		err := rclone.Sync(ctx, config, task, outLog)
		if utils.HandleError(err, "", nil, nil) != nil {
			j, _ := utils.NewCommandErrorDTO(0, err).ToJSON()
			a.oc <- j
		}

		j, _ := utils.NewCommandStoppedDTO(id).ToJSON()
		a.oc <- j

		log.Println("Sync stopped!")
	}()

	return id
}

func (a *App) StopCommand(id int) {
	cancel, exists := utils.GetCmd(id)
	if !exists {
		j, _ := utils.NewCommandErrorDTO(id, errors.New("command not found")).ToJSON()
		a.oc <- j
		return
	}

	cancel()

	res, _ := utils.NewCommandStoppedDTO(id).ToJSON()
	a.oc <- res
}
