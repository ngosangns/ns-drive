const fs = require("fs");
const path = require("path");
const os = require("os");

const platform = os.platform();
const rcloneConfigPath = path.join(
  platform === "win32"
    ? process.env.APPDATA
    : (process.env.HOME || process.env.USERPROFILE) + "/.config",
  "rclone",
  "rclone.conf"
);

fs.copyFile(rcloneConfigPath, "./rclone.conf", (err) => {
  if (err) {
    console.error("Error copying the file:", err);
  } else {
    console.log("File copied successfully!");
  }
});
