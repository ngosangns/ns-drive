package backend

import (
	"context"
	"desktop/backend/rclone"
	"desktop/backend/utils"
	"errors"
	"log"
	"time"
)

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
		switch task {
		case "pull":
			err = rclone.Sync(ctx, config, "pull", outLog)
		case "push":
			err = rclone.Sync(ctx, config, "push", outLog)
		case "bi":
			err = rclone.BiSync(ctx, config, outLog)
		}

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
