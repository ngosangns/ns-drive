const fs = require("fs");
const { exec } = require("child_process");
const os = require("os");

const platform = os.platform();

// Paths for service and timer files
const serviceFilePath = "/etc/systemd/system/ngosangns-drive.backup.service";
const timerFilePath = "/etc/systemd/system/ngosangns-drive.backup.timer";

function executeCommand(command) {
  return new Promise((resolve, reject) => {
    exec(command, (error, stdout, stderr) => {
      if (error) reject(error);
      else resolve(stdout || stderr);
    });
  });
}

if (platform === "win32") {
  // Windows uninstallation logic (if applicable)
} else if (platform === "darwin") {
  // macOS uninstallation logic (if applicable)
} else {
  (async () => {
    console.log("Stopping and disabling timer...");
    await executeCommand("systemctl stop ngosangns-drive.backup.timer");
    await executeCommand("systemctl disable ngosangns-drive.backup.timer");

    console.log("Reloading systemd daemon...");
    await executeCommand("systemctl daemon-reload");

    console.log("Removing service and timer files...");
    await fs.unlink(serviceFilePath, (err) => {
      if (err) {
        console.error("Error removing service file:", err);
      }
    });
    await fs.unlink(timerFilePath, (err) => {
      if (err) {
        console.error("Error removing timer file:", err);
      }
    });

    console.log("Service and timer uninstalled successfully!");
  })();
}
