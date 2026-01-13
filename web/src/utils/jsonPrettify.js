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

// Best-effort prettifier for values that might be JSON or Python-literal-ish.
// Returns a pretty JSON string, or null if the input can't be parsed.
export function tryPrettifyJsonishValue(text) {
  if (typeof text !== "string") return null;
  const trimmed = text.trim();
  if (trimmed.length === 0) return null;

  // Handle python bytes prefix like: b'{"a": 1}'
  if (
    trimmed.startsWith("b'{") ||
    trimmed.startsWith('b"{') ||
    trimmed.startsWith("b'[") ||
    trimmed.startsWith('b"[')
  ) {
    const quote = trimmed[1];
    const inner = trimmed.slice(2);
    if (inner.endsWith(quote)) {
      return tryPrettifyJsonishValue(inner.slice(0, -1));
    }
  }

  const parsed = parseJsonish(trimmed);
  if (!parsed) return null;
  try {
    return JSON.stringify(parsed, null, 2);
  } catch {
    return null;
  }
}

// Best-effort extractor for comparisons like:
//   { ... } != { ... }
//   '{ ... }' != '{ ... }'
//   b'{ ... }' != b'{ ... }'
// Returns null if it doesn't look like a JSON-ish comparison.
export function tryExtractJsonishComparison(text) {
  if (typeof text !== "string") return null;
  const source = text;
  let cursor = 0;

  // Find first JSON-ish value in the string.
  const left = extractJsonishValueAtOrAfter(source, cursor);
  if (!left) return null;

  cursor = left.endIndex;
  while (cursor < source.length && /\s/.test(source[cursor])) cursor++;

  const op = source.startsWith("!=", cursor)
    ? "!="
    : source.startsWith("==", cursor)
    ? "=="
    : null;
  if (!op) return null;
  cursor += op.length;
  while (cursor < source.length && /\s/.test(source[cursor])) cursor++;

  const right = extractJsonishValueAtOrAfter(source, cursor);
  if (!right) return null;

  const leftPretty = tryPrettifyJsonishValue(left.value);
  const rightPretty = tryPrettifyJsonishValue(right.value);
  if (!leftPretty || !rightPretty) return null;

  return {
    prefix: source.slice(0, left.startIndex),
    operator: op,
    left: { raw: left.value, pretty: leftPretty },
    right: { raw: right.value, pretty: rightPretty },
    suffix: source.slice(right.endIndex),
  };
}

function extractJsonishValueAtOrAfter(text, startIndex) {
  // Scan forward to a possible json-ish wrapper or direct {/[.
  for (let i = startIndex; i < text.length; i++) {
    const ch = text[i];
    if (ch === "{" || ch === "[") {
      const extracted = extractBalancedJson(text, i);
      if (!extracted) continue;
      return {
        startIndex: i,
        endIndex: extracted.endIndex,
        value: extracted.jsonText,
      };
    }

    // quoted: '{...}' or "{...}"
    if (
      (ch === "'" || ch === '"') &&
      (text[i + 1] === "{" || text[i + 1] === "[")
    ) {
      const quote = ch;
      const jsonStart = i + 1;
      const extracted = extractBalancedJson(text, jsonStart);
      if (!extracted) continue;
      let endIndex = extracted.endIndex;
      if (text[endIndex] === quote) endIndex += 1;
      return {
        startIndex: i,
        endIndex,
        value: extracted.jsonText,
      };
    }

    // python bytes: b'{...}' or b"{...}"
    if (
      ch === "b" &&
      (text[i + 1] === "'" || text[i + 1] === '"') &&
      (text[i + 2] === "{" || text[i + 2] === "[")
    ) {
      const quote = text[i + 1];
      const jsonStart = i + 2;
      const extracted = extractBalancedJson(text, jsonStart);
      if (!extracted) continue;
      let endIndex = extracted.endIndex;
      if (text[endIndex] === quote) endIndex += 1;
      return {
        startIndex: i,
        endIndex,
        value: extracted.jsonText,
      };
    }
  }
  return null;
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
