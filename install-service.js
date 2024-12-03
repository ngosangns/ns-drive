const fs = require("fs");
const { exec } = require("child_process");
const os = require("os");
const path = require("path");

const platform = os.platform();

// Function to write a file
function writeFile(filePath, content) {
  return new Promise((resolve, reject) => {
    fs.writeFile(filePath, content, { mode: 0o644 }, (err) => {
      if (err) reject(err);
      else resolve();
    });
  });
}

// Function to execute a shell command
function executeCommand(command) {
  return new Promise((resolve, reject) => {
    exec(command, (error, stdout, stderr) => {
      if (error) reject(error);
      else resolve(stdout || stderr);
    });
  });
}

if (platform === "win32") {
} else if (platform === "darwin") {
  // Paths for service and timer files in macOS launchd directory
  const launchAgentDir = path.join(process.env.HOME, "Library", "LaunchAgents");
  const serviceFilePath = path.join(
    launchAgentDir,
    "com.ngosangns.drive.backup.plist"
  );

  // LaunchAgent plist content
  const plistContent = fs.readFileSync("backup.macos.plist", "utf8");

  (async () => {
    try {
      console.log("Creating LaunchAgents directory if not exists...");
      if (!fs.existsSync(launchAgentDir)) {
        fs.mkdirSync(launchAgentDir, { recursive: true });
      }

      console.log("Writing plist file...");
      await writeFile(serviceFilePath, plistContent);

      console.log("Loading LaunchAgent...");
      await executeCommand(`launchctl load -w ${serviceFilePath}`);

      console.log("Service installed and scheduled successfully!");
    } catch (error) {
      console.error("Error installing service:", error.message);
    }
  })();
} else {
  const serviceFilePath = "/etc/systemd/system/ngosangns-drive.backup.service";
  const serviceContent = fs.readFileSync("backup.linux.service", "utf8");

  const timerFilePath = "/etc/systemd/system/ngosangns-drive.backup.timer";
  const timerContent = fs.readFileSync("backup.linux.timer", "utf8");

  (async () => {
    console.log("Writing service file...");
    await writeFile(serviceFilePath, serviceContent);

    console.log("Writing timer file...");
    await writeFile(timerFilePath, timerContent);

    console.log("Reloading systemd daemon...");
    await executeCommand("systemctl daemon-reload");

    console.log("Enabling and starting timer...");
    await executeCommand("systemctl enable ngosangns-drive.backup.timer");
    await executeCommand("systemctl start ngosangns-drive.backup.timer");

    console.log("Service and timer installed successfully!");
  })();
}
