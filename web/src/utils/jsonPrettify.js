// Splits a string into text and "key={...}" / "key=[...]" segments.
// Also supports patterns like key=b'{...}' and Python-dict-ish literals (single quotes).

export function splitTextByJsonAssignments(text) {
  if (typeof text !== "string" || text.length === 0) {
    return [{ type: "text", value: text ?? "" }];
  }

  const segments = [];
  let cursor = 0;

  while (cursor < text.length) {
    const eqIndex = text.indexOf("=", cursor);
    if (eqIndex === -1) break;

    const keyStartIndex = findKeyStart(text, eqIndex - 1);
    const key = text.slice(keyStartIndex, eqIndex);

    // 1) JSON blocks
    const jsonCandidate = findJsonStartAfterEquals(text, eqIndex + 1);
    if (jsonCandidate) {
      const extracted = extractBalancedJson(text, jsonCandidate.jsonStartIndex);
      if (extracted) {
        const parsed = parseJsonish(extracted.jsonText);
        if (parsed) {
          if (keyStartIndex > cursor) {
            segments.push({
              type: "text",
              value: text.slice(cursor, keyStartIndex),
            });
          }

          segments.push({
            type: "json",
            key,
            pretty: JSON.stringify(parsed, null, 2),
          });

          let nextCursor = extracted.endIndex;
          // If the JSON was inside quotes, skip the closing quote too.
          if (
            jsonCandidate.wrapperQuote &&
            text[nextCursor] === jsonCandidate.wrapperQuote
          ) {
            nextCursor += 1;
          }
          cursor = nextCursor;
          continue;
        }
      }
    }

    // 2) Copyable URL-ish values (url=..., path_url=...)
    if (isUrlKey(key)) {
      const extractedUrl = extractSimpleValue(text, eqIndex + 1);
      if (extractedUrl) {
        if (keyStartIndex > cursor) {
          segments.push({
            type: "text",
            value: text.slice(cursor, keyStartIndex),
          });
        }

        segments.push({
          type: "copy",
          key,
          value: extractedUrl.value,
          label: "Copy URL",
        });

        cursor = extractedUrl.endIndex;
        continue;
      }
    }

    cursor = eqIndex + 1;
  }

  if (cursor < text.length) {
    segments.push({ type: "text", value: text.slice(cursor) });
  }

  if (segments.length === 0) return [{ type: "text", value: text }];
  return segments;
}

function isUrlKey(key) {
  if (!key) return false;
  const lower = key.toLowerCase();
  return lower === "url" || lower === "path_url" || lower.endsWith("url");
}

function extractSimpleValue(text, startIndex) {
  let i = startIndex;
  while (i < text.length && /\s/.test(text[i])) i++;

  if (i >= text.length) return null;

  // Support quoted values: url="..." / url='...'
  if (text[i] === '"' || text[i] === "'") {
    const quote = text[i];
    i++;
    const valueStart = i;
    while (i < text.length) {
      const ch = text[i];
      if (ch === quote) {
        return {
          value: text.slice(valueStart, i),
          endIndex: i + 1,
        };
      }
      i++;
    }
    return null;
  }

  // Unquoted: stop at whitespace (or common separators like comma)
  const valueStart = i;
  while (i < text.length && !/\s/.test(text[i]) && text[i] !== ",") i++;
  if (i === valueStart) return null;

  return {
    value: text.slice(valueStart, i),
    endIndex: i,
  };
}

function findKeyStart(text, indexBeforeEquals) {
  let i = indexBeforeEquals;
  while (i >= 0) {
    const ch = text[i];
    if (!/[A-Za-z0-9_]/.test(ch)) return i + 1;
    i--;
  }
  return 0;
}

function findJsonStartAfterEquals(text, startIndex) {
  let i = startIndex;
  while (i < text.length && /\s/.test(text[i])) i++;

  // Support python bytes prefix: b'...'
  if (text[i] === "b" && (text[i + 1] === "'" || text[i + 1] === '"')) {
    const quote = text[i + 1];
    const afterQuote = i + 2;
    const ch = text[afterQuote];
    if (ch === "{" || ch === "[") {
      return { jsonStartIndex: afterQuote, wrapperQuote: quote };
    }
  }

  // Support quoted JSON: key='{"a":1}' or key="{...}"
  if (text[i] === "'" || text[i] === '"') {
    const quote = text[i];
    const afterQuote = i + 1;
    const ch = text[afterQuote];
    if (ch === "{" || ch === "[") {
      return { jsonStartIndex: afterQuote, wrapperQuote: quote };
    }
  }

  // Raw JSON: key={...} / key=[...]
  if (text[i] === "{" || text[i] === "[") {
    return { jsonStartIndex: i, wrapperQuote: null };
  }

  return null;
}

function extractBalancedJson(text, startIndex) {
  const open = text[startIndex];
  const close = open === "{" ? "}" : "]";

  let depth = 0;
  let inString = false;
  let escape = false;

  for (let i = startIndex; i < text.length; i++) {
    const ch = text[i];

    if (inString) {
      if (escape) {
        escape = false;
        continue;
      }
      if (ch === "\\") {
        escape = true;
        continue;
      }
      if (ch === '"') inString = false;
      continue;
    }

    if (ch === '"') {
      inString = true;
      continue;
    }

    if (ch === open) depth++;
    if (ch === close) depth--;

    if (depth === 0) {
      return {
        jsonText: text.slice(startIndex, i + 1),
        endIndex: i + 1,
      };
    }
  }

  return null;
}

function parseJsonish(maybeJson) {
  if (typeof maybeJson !== "string") return null;

  // First: strict JSON
  try {
    return JSON.parse(maybeJson);
  } catch {
    // continue
  }

  // Second: Python-literal-ish object/list (single quotes, None/True/False)
  const pythonish = pythonLiteralToJson(maybeJson);
  if (pythonish) {
    try {
      return JSON.parse(pythonish);
    } catch {
      // continue
    }
  }

  return null;
}

function pythonLiteralToJson(value) {
  // Only attempt if it looks like a dict/list with single quotes.
  if (!(value.startsWith("{") || value.startsWith("["))) return null;
  if (!value.includes("'")) return null;

  // Minimal, heuristic conversion:
  // - Replace Python booleans/nulls
  // - Replace single quotes with double quotes
  // This works well for common headers/body dicts.
  return value
    .replace(/\bNone\b/g, "null")
    .replace(/\bTrue\b/g, "true")
    .replace(/\bFalse\b/g, "false")
    .replace(/'/g, '"');
}
