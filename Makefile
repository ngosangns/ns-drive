# Load environment variables from .env file
ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

# Targets
.PHONY: export import web pull pull-full push backup install-service uninstall-service build-app

# Export
export:
	node export.js

# Import
import:
	node import.js

# Client
web:
	rclone rcd --rc-web-gui

pull:
	@trap 'kill 0' SIGINT SIGTERM EXIT; \
	rclone sync $(CLIENT_FROM_PATH) $(CLIENT_PATH) -P --filter-from $(CLIENT_FILTER_PATH) --track-renames --bwlimit $(CLIENT_LIMIT_BANDWIDTH) --transfers $(CLIENT_PARALLEL)

pull-full:
	@rclone sync $(CLIENT_FROM_PATH) $(CLIENT_PATH) -P --filter-from $(CLIENT_FILTER_PATH) --track-renames --bwlimit $(CLIENT_LIMIT_BANDWIDTH) --transfers $(CLIENT_PARALLEL)

push:
	@rclone sync $(CLIENT_PATH) $(CLIENT_FROM_PATH) -P --filter-from $(CLIENT_FILTER_PATH) --transfers $(CLIENT_PARALLEL)

# Server
backup:
	node server-backup.js

install-service:
	node install-service.js

uninstall-service:
	node uninstall-service.js

# Desktop
build-app:
	cd desktop && \
	wails build -v 2 && \
	# codesign --sign - --force --deep build/bin/desktop.app && \
	# upx --best build/bin/desktop --force-macos && \
	# chmod +x build/bin/desktop.app/Contents/MacOS/desktop && \
	rm -rf ../desktop.app && \
	mv build/bin/desktop.app ../desktop.app
	# open ../desktop.app