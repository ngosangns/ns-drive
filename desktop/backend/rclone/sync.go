package rclone

import (
	"context"

	beConfig "desktop/backend/config"
	"desktop/backend/utils"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
	fssync "github.com/rclone/rclone/fs/sync"

	// import fs drivers
	_ "github.com/rclone/rclone/backend/drive"
	_ "github.com/rclone/rclone/backend/local"
)

func Sync(ctx context.Context, config *beConfig.Config, task string, outLog chan string) error {
	// Initialize the config
	fsConfig := fs.GetConfig(ctx)
	fsConfig.Transfers = config.Parallel

	var err error

	switch task {
	// case "pull":
	case "push":
		config.FromFs, config.ToFs = config.ToFs, config.FromFs
	}

	srcFs, err := fs.NewFs(ctx, config.FromFs)
	if utils.HandleError(err, "Failed to initialize source filesystem", nil, nil) != nil {
		return err
	}

	dstFs, err := fs.NewFs(ctx, config.ToFs)
	if utils.HandleError(err, "Failed to initialize destination filesystem", nil, nil) != nil {
		return err
	}

	// Set up filter rules
	filterOpt := filter.GetConfig(ctx).Opt
	filterOpt.FilterFrom = append([]string{config.FilterFile}, filterOpt.FilterFrom...)
	newFilter, err := filter.NewFilter(&filterOpt)
	utils.HandleError(err, "Invalid filters file", nil, func() {
		ctx = filter.ReplaceConfig(ctx, newFilter)
	})

	// Set bandwidth limit
	if config.Bandwidth != "" {
		utils.HandleError(fsConfig.BwLimit.Set(config.Bandwidth), "Failed to set bandwidth limit", nil, nil)
	}

	fsConfig.Reload(ctx)

	return utils.RunRcloneWithRetryAndStats(ctx, true, false, outLog, func() error {
		return utils.HandleError(fssync.Sync(ctx, dstFs, srcFs, false), "Sync failed", nil, nil)
	})
}
