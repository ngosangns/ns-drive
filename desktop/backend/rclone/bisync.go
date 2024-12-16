package rclone

import (
	"context"
	beConfig "desktop/backend/config"
	"desktop/backend/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/rclone/rclone/cmd/bisync"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
)

func BiSync(ctx context.Context, config *beConfig.Config, outLog chan string) error {
	var err error

	// Initialize the config
	fsConfig := fs.GetConfig(ctx)
	opt := &bisync.Options{}
	opt.Force = true
	opt.Compare.DownloadHash = true
	opt.CompareFlag = "size,modtime,checksum"
	// opt.DryRun = true

	// if err = opt.ConflictResolve.Set(bisync.PreferNewer.String()); err != nil {
	// 	return err
	// }

	// Handle resync
	dir, err := os.Getwd()
	if utils.HandleError(err, "Failed to get current working directory", nil, nil) != nil {
		return err
	}
	path := filepath.Join(dir, ".resync")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, err = os.Create(path)
		if utils.HandleError(err, "Failed to create .resync file", nil, nil) != nil {
			return err
		}
	}
	resyncContent, err := os.ReadFile(path)
	if utils.HandleError(err, "Failed to read .resync file", nil, nil) != nil {
		return err
	}
	if !strings.Contains(string(resyncContent), "\n"+config.FromFs+"|"+config.ToFs+"\n") {
		err = os.WriteFile(path, []byte("\n"+config.FromFs+"|"+config.ToFs+"\n"), 0644)
		if utils.HandleError(err, "Failed to write to .resync file", nil, nil) != nil {
			return err
		}
		opt.Resync = true
		// if err = opt.ResyncMode.Set(bisync.PreferNewer.String()); err != nil {
		// 	return err
		// }
	}

	if err = opt.ConflictLoser.Set(bisync.ConflictLoserDelete.String()); err != nil {
		return err
	}

	if err = opt.CheckSync.Set(bisync.CheckSyncTrue.String()); err != nil {
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
