const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const rootDir = path.resolve(__dirname, "..");
const binDir = path.join(rootDir, "bin");
const outName = process.platform === "win32" ? "robodiff.exe" : "robodiff";
const outPath = path.join(binDir, outName);

fs.mkdirSync(binDir, { recursive: true });

execFileSync("go", ["build", "-o", outPath], {
  cwd: rootDir,
  stdio: "inherit",
});
