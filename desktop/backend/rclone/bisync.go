package rclone

import (
	"context"
	beConfig "desktop/backend/config"
	"desktop/backend/utils"

	"github.com/rclone/rclone/cmd/bisync"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
)

func BiSync(ctx context.Context, config *beConfig.Config, outLog chan string) error {
	var err error

	// Initialize the config
	fsConfig := fs.GetConfig(ctx)

	opt := &bisync.Options{
		Resync: true,
		DryRun: true,
	}

	opt.CompareFlag = "modtime,size,checksum"

	if err = opt.ConflictResolve.Set("prefer-newer"); err != nil {
		return err
	}

	if err = opt.ResyncMode.Set("prefer-newer"); err != nil {
		return err
	}

	var checkSync = "true"
	if err = opt.CheckSync.Set(checkSync); err != nil {
		return err
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

	// Set parallel transfers
	fsConfig.Transfers = config.Parallel

	fsConfig.Progress = true
	fsConfig.Reload(ctx)

	return utils.RunRcloneWithRetryAndStats(ctx, true, false, outLog, func() error {
		return utils.HandleError(bisync.Bisync(ctx, dstFs, srcFs, opt), "Sync failed", nil, nil)
	})
}
