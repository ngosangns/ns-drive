const fs = require("fs");
const path = require("path");

// Read and parse the JSON configuration file
const jsonConfig = JSON.parse(fs.readFileSync("config.json", "utf8"));

// Prepare the content for rclone.conf
let rcloneConfigContent = "";
for (const [remoteName, config] of Object.entries(jsonConfig)) {
  rcloneConfigContent += `[${remoteName}]\n`;
  for (const [key, value] of Object.entries(config)) {
    rcloneConfigContent += `  ${key} = ${value}\n`;
  }
  rcloneConfigContent += "\n";
}

// Path to the rclone configuration file
const rcloneConfigPath = path.join(
  process.env.HOME || process.env.USERPROFILE,
  ".config",
  "rclone",
  "rclone.conf",
);

// Write the formatted configuration to rclone.conf
fs.writeFileSync(rcloneConfigPath, rcloneConfigContent, "utf8");

console.log("rclone configuration imported successfully.");
