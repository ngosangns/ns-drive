const { exec } = require("child_process");
const os = require("os");

const filterPart = "--filter-from server.rclonefilter";
const timestamp = new Date()
  .toISOString()
  .replace(/[-T:.Z]/g, "")
  .slice(0, 15); // Format: yyyyMMddHHmmss
const platform = os.platform();

const backupCommand = (remoteName, timestamp) =>
  `rclone sync onedrive-dev:/ ${remoteName}:/ -P ${filterPart} --backup-dir ${remoteName}:/.rclone-backup/${timestamp}`;

const backupAllInWindows = (commands) => {
  for (const command of commands)
    exec(`start cmd.exe /k "@echo ${command} & ${command}"`)
};

if (platform === "win32") {
  backupAllInWindows([
    backupCommand("onedrive-dev-cript", timestamp),
    backupCommand("yandex-cript", timestamp),
  ]);
} else if (platform === "darwin") {
  console.log("The operating system is macOS.");
} else {
  console.log("The operating system is neither Windows nor macOS.");
}
