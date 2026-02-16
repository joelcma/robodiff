const { app, BrowserWindow, dialog, ipcMain } = require("electron");
const { spawn } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
let getPort;

let backendProcess = null;
let mainWindow = null;
let backendAddr = null;
let backendDir = process.env.robodiff_DIR || "";
let logStream = null;

function getSettingsPath() {
  return path.join(app.getPath("userData"), "robodiff-settings.json");
}

function getLogPath() {
  return path.join(app.getPath("userData"), "robodiff.log");
}

function logLine(message) {
  const line = `[${new Date().toISOString()}] ${message}\n`;
  try {
    const logPath = getLogPath();
    fs.mkdirSync(path.dirname(logPath), { recursive: true });
    if (!logStream) {
      logStream = fs.createWriteStream(logPath, { flags: "a" });
    }
    logStream.write(line);
  } catch {
    try {
      fs.appendFileSync(path.join(os.tmpdir(), "robodiff.log"), line);
    } catch {
      // Ignore logging errors.
    }
  }
}

function loadStoredDir() {
  try {
    const raw = fs.readFileSync(getSettingsPath(), "utf8");
    const data = JSON.parse(raw);
    if (typeof data.lastDir === "string" && data.lastDir.trim() !== "") {
      return expandTilde(data.lastDir.trim());
    }
  } catch {
    // Ignore missing/invalid settings.
  }
  return "";
}

function saveStoredDir(dir) {
  try {
    const payload = { lastDir: dir };
    fs.writeFileSync(getSettingsPath(), JSON.stringify(payload, null, 2));
  } catch {
    // Ignore settings write errors.
  }
}

function expandTilde(inputPath) {
  if (!inputPath) return inputPath;
  if (inputPath === "~") return os.homedir();
  if (inputPath.startsWith("~/") || inputPath.startsWith("~\\")) {
    return path.join(os.homedir(), inputPath.slice(2));
  }
  return inputPath;
}

const devServerUrl =
  process.env.VITE_DEV_SERVER_URL ||
  (process.env.ELECTRON_DEV ? "http://localhost:5173" : "");
const isDev = !app.isPackaged && Boolean(devServerUrl);

function resolveBackendBinary() {
  const binaryName = process.platform === "win32" ? "robodiff.exe" : "robodiff";
  if (app.isPackaged) {
    return path.join(process.resourcesPath, "bin", binaryName);
  }
  return path.join(app.getAppPath(), "bin", binaryName);
}

async function startBackend({ dir } = {}) {
  const binaryPath = resolveBackendBinary();
  logLine(`Starting backend. binary=${binaryPath}`);
  if (!fs.existsSync(binaryPath)) {
    const message = [
      "Go backend binary not found.",
      "Run: npm run build:go",
      "Expected: " + binaryPath,
    ].join("\n");
    await dialog.showMessageBox({
      type: "error",
      title: "Backend missing",
      message,
    });
    logLine("Backend binary missing; quitting.");
    app.quit();
    return null;
  }

  if (!backendAddr) {
    if (!getPort) {
      ({ default: getPort } = await import("get-port"));
    }
    const port = await getPort();
    backendAddr = `127.0.0.1:${port}`;
  }
  if (typeof dir === "string") {
    backendDir = expandTilde(dir.trim());
  }
  const addr = backendAddr;
  const apiBase = `http://${addr}`;
  process.env.robodiff_API_BASE = apiBase;

  const args = ["--addr", addr];
  if (backendDir) {
    process.env.robodiff_DIR = backendDir;
    args.push("--dir", backendDir);
  }
  logLine(`Backend args: ${args.join(" ")}`);
  const cwd = app.getPath("home");

  backendProcess = spawn(binaryPath, args, {
    cwd,
    env: { ...process.env },
    stdio: isDev ? "inherit" : "ignore",
    windowsHide: true,
  });

  backendProcess.on("exit", (code) => {
    if (app.isQuitting) return;
    logLine(`Backend exited (${code}).`);
    console.error(`Backend exited (${code}).`);
  });

  return apiBase;
}

function stopBackend() {
  return new Promise((resolve) => {
    if (!backendProcess) {
      resolve();
      return;
    }

    const proc = backendProcess;
    backendProcess = null;
    proc.once("exit", () => resolve());
    proc.kill();

    setTimeout(() => resolve(), 1000);
  });
}

async function restartBackend(dir) {
  await stopBackend();
  let normalized = "";
  if (typeof dir === "string" && dir.trim() !== "") {
    normalized = expandTilde(dir.trim());
    saveStoredDir(normalized);
  }
  return startBackend({ dir: normalized });
}

async function createWindow() {
  const preloadPath = path.join(app.getAppPath(), "electron", "preload.js");
  logLine(`Preload path: ${preloadPath}`);
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 900,
    minWidth: 960,
    minHeight: 640,
    backgroundColor: "#0c0c0f",
    webPreferences: {
      preload: preloadPath,
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  if (isDev) {
    logLine(`Loading dev URL: ${devServerUrl}`);
    await mainWindow.loadURL(devServerUrl);
    mainWindow.webContents.openDevTools({ mode: "detach" });
  } else {
    const indexPath = path.join(app.getAppPath(), "web", "dist", "index.html");
    logLine(`Loading file: ${indexPath}`);
    try {
      await mainWindow.loadFile(indexPath);
    } catch (err) {
      logLine(`Load failed: ${err && err.message ? err.message : String(err)}`);
      await dialog.showMessageBox({
        type: "error",
        title: "Failed to load UI",
        message: "Robodiff failed to load its UI.",
        detail: String(err),
      });
    }
  }

  mainWindow.webContents.on("did-fail-load", (_event, code, desc, url) => {
    logLine(`did-fail-load code=${code} desc=${desc} url=${url}`);
  });

  mainWindow.webContents.on("render-process-gone", (_event, details) => {
    logLine(`render-process-gone: ${details.reason}`);
  });

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

app.whenReady().then(async () => {
  if (process.platform === "win32") {
    app.setAppUserModelId("com.robodiff.app");
  }

  if (!backendDir) {
    const storedDir = loadStoredDir();
    if (storedDir) {
      backendDir = storedDir;
    }
  }

  logLine("App ready.");
  await startBackend();
  await createWindow();

  ipcMain.handle("robodiff:selectDir", async () => {
    const browserWindow = BrowserWindow.getFocusedWindow() || mainWindow;
    const result = await dialog.showOpenDialog(browserWindow, {
      title: "Select Robot Results Folder",
      properties: ["openDirectory"],
    });
    if (result.canceled || result.filePaths.length === 0) {
      return { canceled: true };
    }
    const dir = result.filePaths[0];
    await restartBackend(dir);
    return { canceled: false, dir };
  });

  ipcMain.handle("robodiff:setDir", async (_event, dir) => {
    if (typeof dir !== "string" || dir.trim() === "") {
      return { canceled: true };
    }
    const nextDir = expandTilde(dir.trim());
    await restartBackend(nextDir);
    return { canceled: false, dir: nextDir };
  });

  app.on("activate", async () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      await createWindow();
    }
  });
});

app.on("before-quit", () => {
  app.isQuitting = true;
  if (backendProcess) {
    backendProcess.kill();
  }
});

app.on("window-all-closed", () => {
  app.quit();
});
