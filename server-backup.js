const { exec, spawn } = require("child_process");
const os = require("os");
const platform = os.platform();

const SERVER_BACKUP_PAIRS = [
  {
    from: "google-drive:/drive/",
    to: ["yandex:/drive/"],
    backupChangesPath: "yandex:/drive/.rclone-backup/",
    filterPath: ".rclonefilter.server",
    isBackupChanges: true,
    limitBandwidth: "9999M",
    parallel: 24,
  },
  {
    from: "google-photos:/media/all/",
    to: ["yandex:/google-photos/media/all/"],
    filterPath: "",
    isBackupChanges: true,
    backupChangesPath: "yandex:/.rclone-backup/google-photos/media/all/",
    limitBandwidth: "9999M",
    parallel: 24,
  },
];
const IS_USE_TMUX = false;

// Build command
const filterPart = (path) => `--filter-from ${path}`;
const bandwidthPart = (limit) => `--bwlimit ${limit}`;
const backupDirPart = (backupChangesPath, timestamp) =>
  `--backup-dir ${backupChangesPath}/${timestamp}`;
const parallelPart = (num) => `--transfers ${num}`;
const backupCommand = (
  fromPath,
  toPath,
  filterPath,
  isBackupChanges,
  backupChangesPath,
  limitBandwidth,
  parallel,
  now
) =>
  `rclone sync ${fromPath} ${toPath} -P ${
    filterPath.length ? filterPart(filterPath) : ""
  } ${
    isBackupChanges ? backupDirPart(backupChangesPath, now) : ""
  } ${bandwidthPart(limitBandwidth)} ${parallelPart(parallel)}`;

const backupAllInWindows = (commands) => {
  for (const command of commands)
    exec(`start cmd.exe /k "@echo ${command} & ${command}"`);
};
const backupAllInLinux = (commands, timestamp) => {
  let tmuxCommand = [];

  if (IS_USE_TMUX) {
    const sessionName = "ngosangns-drive-backup-" + timestamp;
    for (let i = 0; i < commands.length; i++) {
      if (i === 0)
        tmuxCommand.push(
          `tmux new-session -d -s ${sessionName} '${commands[i]}'`
        );
      else
        tmuxCommand.push(
          `tmux split-window -v -t ${sessionName} '${commands[i]}'`
        );
    }
    tmuxCommand.push(
      `tmux select-layout even-horizontal`,
      `tmux attach-session -t ${sessionName}`
    );
    tmuxCommand = tmuxCommand.join(" ; ").split(/\s+/g);

    const process = spawn(tmuxCommand[0], tmuxCommand.slice(1), {
      stdio: "inherit",
      shell: true,
    });
    process.on("close", (code) => {
      console.log(
        `tmux session "${sessionName}" closed with exit code ${code}`
      );
    });
    process.on("error", (err) => {
      console.error(`Error running tmux: ${err.message}`);
    });
  } else {
    for (const command of commands) {
      commandParts = command.split(/\s+/g);
      const process = spawn(commandParts[0], commandParts.slice(1), {
        stdio: "inherit",
        shell: true,
      });
      process.on("close", (code) => {
        console.log(`Command "${command}" finished with exit code ${code}`);
      });
      process.on("error", (err) => {
        console.error(`Error running command "${command}": ${err.message}`);
      });
    }
  }
};

const commands = [];
for (const pair of SERVER_BACKUP_PAIRS) {
  const now = new Date()
    .toISOString()
    .replace(/[-T:.Z]/g, "")
    .slice(0, 15); // Format: yyyyMMddHHmmss
  for (const path of pair.to)
    commands.push(
      backupCommand(
        pair.from,
        path,
        pair.filterPath,
        pair.isBackupChanges,
        pair.backupChangesPath,
        pair.limitBandwidth,
        pair.parallel,
        now
      )
    );

  if (platform === "win32") {
    backupAllInWindows(commands);
  } else {
    backupAllInLinux(commands, now);
  }
}
