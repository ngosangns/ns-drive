package main

import (
	"context"
	rclone "desktop/backend/rclone"
	"os"
)

func main() {
	os.Chdir("../../../../")
	rclone.Initial()
	ctx := context.Background()
	ctx = rclone.InitConfig(ctx)
	outLog := make(chan string)
	rclone.Sync(ctx, outLog)
}
