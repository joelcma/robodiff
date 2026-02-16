export function getApiBase() {
  if (typeof window !== "undefined" && window.robodiff?.apiBase) {
    return window.robodiff.apiBase;
  }
  return "";
}

export function buildApiUrl(path) {
  if (!path) return getApiBase();
  if (/^https?:\/\//i.test(path)) return path;
  const base = getApiBase();
  if (!base) return path;
  if (path.startsWith("/")) return `${base}${path}`;
  return `${base}/${path}`;
}
