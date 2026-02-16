const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("robodiff", {
  apiBase: process.env.robodiff_API_BASE || "",
  selectDirectory: () => ipcRenderer.invoke("robodiff:selectDir"),
  setDirectory: (dir) => ipcRenderer.invoke("robodiff:setDir", dir),
});
