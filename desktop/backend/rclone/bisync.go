package rclone

import (
	"context"
	"crypto/sha256"
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
	opt.CompareFlag = "size,modtime,checksum"
	opt.Recover = true
	opt.CreateEmptySrcDirs = true
	// opt.Resilient = true
	// opt.DryRun = true

	if err = opt.ConflictResolve.Set(bisync.PreferNewer.String()); err != nil {
		return err
	}

	if err = opt.ConflictLoser.Set(bisync.ConflictLoserDelete.String()); err != nil {
		return err
	}

	if err = opt.CheckSync.Set(bisync.CheckSyncTrue.String()); err != nil {
		return err
	}

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
	filterFileChecksum := "0"
	if config.FilterFile != "" {
		filterFileChecksum, err = utils.CalculateFileHash(filepath.Join(dir, config.FilterFile), sha256.New)
		if utils.HandleError(err, "Failed to calculate hash of filter file", nil, nil) != nil {
			return err
		}
	}
	filterFileChecksumLinePrefix := config.FromFs + "|" + config.ToFs + "|"
	filterFileChecksumLine := filterFileChecksumLinePrefix + filterFileChecksum
	filterFileChecksumLineContained := false
	resyncContent, err := os.ReadFile(path)
	if utils.HandleError(err, "Failed to read .resync file", nil, nil) != nil {
		return err
	}
	resyncContentStrArr := strings.Split(string(resyncContent), "\n")
	for _, line := range resyncContentStrArr {
		if line == filterFileChecksumLine {
			filterFileChecksumLineContained = true
			break
		}
	}
	if !filterFileChecksumLineContained {
		// Update .resync file content
		newResyncContentStrArr := []string{}
		isAdded := false
		for _, line := range resyncContentStrArr {
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, filterFileChecksumLinePrefix) {
				newResyncContentStrArr = append(newResyncContentStrArr, filterFileChecksumLine)
				isAdded = true
			} else {
				newResyncContentStrArr = append(newResyncContentStrArr, line)
			}
		}
		if !isAdded {
			newResyncContentStrArr = append(newResyncContentStrArr, filterFileChecksumLine)
		}
		// Write to .resync file
		err = os.WriteFile(path, []byte(strings.Join(newResyncContentStrArr, "\n")), 0644)
		if utils.HandleError(err, "Failed to write to .resync file", nil, nil) != nil {
			return err
		}
		// Set resync flag
		opt.Resync = true
		// if err = opt.ResyncMode.Set(bisync.PreferNewer.String()); err != nil {
		// 	return err
		// }
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
