const { exec, spawn } = require("child_process");
const os = require("os");
const platform = os.platform();

const SERVER_BACKUP_PAIRS = [
  {
    from: "onedrive-dev:/",
    to: ["yandex:/"],
    filterPath: ".rclonefilter.server",
    isBackupChanges: true,
    limitBandwidth: "9999M",
    parallel: 24,
  },
  {
    from: "google-photos:/media/all",
    to: ["yandex:/.google-photos/media/all"],
    filterPath: "",
    isBackupChanges: true,
    limitBandwidth: "9999M",
    parallel: 24,
  },
];

// Build command
const filterPart = (path) => `--filter-from ${path}`;
const bandwidthPart = (limit) => `--bwlimit ${limit}`;
const backupDirPart = (toPath, timestamp) =>
  `--backup-dir ${toPath}.rclone-backup/${timestamp}`;
const parallelPart = (num) => `--transfers ${num}`;
const backupCommand = (
  fromPath,
  toPath,
  filterPath,
  isBackupChanges,
  limitBandwidth,
  parallel,
  now
) =>
  `rclone sync ${fromPath} ${toPath} -P ${
    filterPath.length ? filterPart(filterPath) : ""
  } ${isBackupChanges ? backupDirPart(toPath, now) : ""} ${bandwidthPart(
    limitBandwidth
  )} ${parallelPart(parallel)}`;

const backupAllInWindows = (commands) => {
  for (const command of commands)
    exec(`start cmd.exe /k "@echo ${command} & ${command}"`);
};
const backupAllInLinux = (commands) => {
  const sessionName = "ngosangns-drive-backup";
  let tmuxCommand = [];
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
    console.log(`tmux session "${sessionName}" closed with exit code ${code}`);
  });
  process.on("error", (err) => {
    console.error(`Error running tmux: ${err.message}`);
  });
};

const commands = [];
for (const pair of SERVER_BACKUP_PAIRS) {
  const now = new Date()
    .toISOString()
    .replace(/[-T:.Z]/g, "")
    .slice(0, 15); // Format: yyyyMMddHHmmss
  for (const path of pair.to.split(",").filter((i) => i.length))
    commands.push(
      backupCommand(
        pair.from,
        path,
        pair.filterPath,
        pair.isBackupChanges,
        pair.limitBandwidth,
        pair.parallel,
        now
      )
    );

  if (platform === "win32") {
    backupAllInWindows(commands);
  } else if (platform === "darwin") {
    console.log("The operating system is macOS.");
  } else {
    backupAllInLinux(commands);
  }
}
