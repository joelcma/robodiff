import { splitTextByJsonAssignments } from "./jsonPrettify";

export function isHttpRequestMessage(text) {
  if (typeof text !== "string") return false;
  return /\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+Request\s*:/i.test(text);
}

export function buildCurlFromText(text) {
  const req = extractHttpRequestFromText(text);
  if (!req) return null;

  const parts = ["curl", "-X", req.method, shellQuote(req.url)];

  if (
    req.headers &&
    typeof req.headers === "object" &&
    !Array.isArray(req.headers)
  ) {
    for (const [k, v] of Object.entries(req.headers)) {
      if (String(k).toLowerCase() === "content-length") continue;
      parts.push("-H", shellQuote(`${k}: ${String(v ?? "")}`));
    }
  }

  if (req.body != null && req.body !== "") {
    parts.push("--data-raw", shellQuote(req.body));
  }

  return {
    url: req.url,
    curl: parts.join(" "),
  };
}

export function extractHttpRequestFromText(text) {
  if (!isHttpRequestMessage(text)) return null;

  const segments = splitTextByJsonAssignments(text);

  const methodMatch = text.match(
    /\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+Request\s*:/i
  );
  const method = (methodMatch?.[1] || "GET").toUpperCase();

  const urlSeg = segments.find((s) => s?.type === "copy" && isUrlKey(s.key));
  const url = urlSeg?.value;
  if (!url) return null;

  const headersSeg = segments.find(
    (s) => s?.type === "json" && String(s.key).toLowerCase() === "headers"
  );
  const bodySeg = segments.find(
    (s) => s?.type === "json" && String(s.key).toLowerCase() === "body"
  );

  const headers = normalizeObjectJson(headersSeg?.pretty);
  const bodyValue = normalizeAnyJson(bodySeg?.pretty);
  const bodyText =
    bodyValue == null
      ? ""
      : typeof bodyValue === "string"
      ? bodyValue
      : JSON.stringify(bodyValue);

  return {
    method,
    url,
    headers: normalizeHeaders(headers),
    body: bodyText,
  };
}

function isUrlKey(key) {
  if (!key) return false;
  const lower = String(key).toLowerCase();
  return lower === "url" || lower === "path_url" || lower.endsWith("url");
}

function normalizeObjectJson(prettyJson) {
  if (!prettyJson) return null;
  try {
    const parsed = JSON.parse(prettyJson);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed))
      return null;
    return parsed;
  } catch {
    return null;
  }
}

function normalizeAnyJson(prettyJson) {
  if (!prettyJson) return null;
  try {
    return JSON.parse(prettyJson);
  } catch {
    return null;
  }
}

function normalizeHeaders(headersObj) {
  if (!headersObj) return {};
  const out = {};
  for (const [k, v] of Object.entries(headersObj)) {
    if (String(k).toLowerCase() === "content-length") continue;
    out[k] = v && typeof v === "object" ? JSON.stringify(v) : String(v ?? "");
  }
  return out;
}

function shellQuote(value) {
  // POSIX-ish single-quote escaping: ' -> '\''
  const s = String(value ?? "");
  return `'${s.replace(/'/g, `'\\''`)}'`;
}
