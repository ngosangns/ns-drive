const { exec } = require("child_process");
const fs = require("fs");

exec("rclone config dump", (err, stdout, stderr) => {
  try {
    if (err) throw err;
    if (stderr) throw stderr;

    const config = JSON.parse(stdout);
    for (const key in config) {
      if (config[key].token) config[key].token = "";
      if (config[key].password) config[key].password = "";
    }
    fs.writeFileSync("./config.json", JSON.stringify(config, null, 2), "utf8");
    console.log("rclone configuration updated successfully.");
  } catch (parseError) {
    console.error("Error parsing rclone config dump:", parseError);
  }
});
