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
      // If the value is inside quotes (including python bytes literals like b"..."),
      // extract the full quoted payload, unescape it, then parse.
      if (
        jsonCandidate.wrapperQuote &&
        jsonCandidate.wrapperStartIndex != null
      ) {
        const wrapped = tryParseJsonishWrappedValue(text, jsonCandidate);
        if (wrapped) {
          if (keyStartIndex > cursor) {
            segments.push({
              type: "text",
              value: text.slice(cursor, keyStartIndex),
            });
          }

          segments.push({
            type: "json",
            key,
            pretty: JSON.stringify(wrapped.parsed, null, 2),
          });

          cursor = wrapped.endIndex;
          continue;
        }
      }

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

      // If the JSON is truncated/unbalanced (common for error bodies with huge stack traces),
      // try parsing a safe prefix by dropping long trailing fields like "trace".
      const truncated = tryParseTruncatedJsonAt(
        text,
        jsonCandidate.jsonStartIndex
      );
      if (truncated) {
        if (keyStartIndex > cursor) {
          segments.push({
            type: "text",
            value: text.slice(cursor, keyStartIndex),
          });
        }

        segments.push({
          type: "json",
          key,
          pretty: JSON.stringify(truncated.parsed, null, 2),
        });

        cursor = truncated.endIndex;
        continue;
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

  if (segments.length === 0) return splitTextByStandaloneJson(text);

  // Post-process leftover text segments for standalone JSON blocks.
  const out = [];
  for (const seg of segments) {
    if (seg.type !== "text") {
      out.push(seg);
      continue;
    }
    const parts = splitTextByStandaloneJson(seg.value);
    // If nothing was split, preserve original segment to avoid churn.
    if (parts.length === 1 && parts[0].type === "text") out.push(seg);
    else out.push(...parts);
  }
  return out;
}

function splitTextByStandaloneJson(text) {
  if (typeof text !== "string" || text.length === 0) {
    return [{ type: "text", value: text ?? "" }];
  }

  const segments = [];
  let cursor = 0;

  while (cursor < text.length) {
    const start = findJsonStartAtLineBoundary(text, cursor);
    if (!start) break;

    // If the JSON is inside quotes (including python bytes literals), parse the unescaped payload.
    if (start.wrapperQuote && start.wrapperStartIndex != null) {
      const wrapped = tryParseJsonishWrappedValue(text, start);
      if (wrapped) {
        if (start.prefixIndex > cursor) {
          segments.push({
            type: "text",
            value: text.slice(cursor, start.prefixIndex),
          });
        }

        segments.push({
          type: "json",
          key: "",
          pretty: JSON.stringify(wrapped.parsed, null, 2),
        });

        cursor = wrapped.endIndex;
        continue;
      }
    }

    const extracted = extractBalancedJson(text, start.jsonStartIndex);
    if (!extracted) {
      const truncated = tryParseTruncatedJsonAt(text, start.jsonStartIndex);
      if (!truncated) {
        cursor = start.jsonStartIndex + 1;
        continue;
      }

      if (start.prefixIndex > cursor) {
        segments.push({
          type: "text",
          value: text.slice(cursor, start.prefixIndex),
        });
      }

      segments.push({
        type: "json",
        key: "",
        pretty: JSON.stringify(truncated.parsed, null, 2),
      });

      cursor = truncated.endIndex;
      continue;
    }

    const parsed = parseJsonish(extracted.jsonText);
    if (!parsed) {
      const truncated = tryParseTruncatedJsonAt(text, start.jsonStartIndex);
      if (!truncated) {
        cursor = start.jsonStartIndex + 1;
        continue;
      }

      if (start.prefixIndex > cursor) {
        segments.push({
          type: "text",
          value: text.slice(cursor, start.prefixIndex),
        });
      }

      segments.push({
        type: "json",
        key: "",
        pretty: JSON.stringify(truncated.parsed, null, 2),
      });

      cursor = truncated.endIndex;
      continue;
    }

    if (start.prefixIndex > cursor) {
      segments.push({
        type: "text",
        value: text.slice(cursor, start.prefixIndex),
      });
    }

    segments.push({
      type: "json",
      key: "",
      pretty: JSON.stringify(parsed, null, 2),
    });

    let nextCursor = extracted.endIndex;
    if (start.wrapperQuote && text[nextCursor] === start.wrapperQuote) {
      nextCursor += 1;
    }
    cursor = nextCursor;
  }

  if (cursor < text.length) {
    segments.push({ type: "text", value: text.slice(cursor) });
  }
  return segments.length === 0 ? [{ type: "text", value: text }] : segments;
}

function findJsonStartAtLineBoundary(text, startIndex) {
  // Only consider JSON that starts at the beginning of the string or at a new line.
  // This avoids accidentally grabbing braces from arbitrary prose.
  for (let i = startIndex; i < text.length; i++) {
    if (i !== 0 && text[i - 1] !== "\n" && text[i - 1] !== "\r") continue;

    // Skip indentation.
    let j = i;
    while (j < text.length && (text[j] === " " || text[j] === "\t")) j++;

    // python bytes prefix: b'{' / b"{"
    if (text[j] === "b" && (text[j + 1] === "'" || text[j + 1] === '"')) {
      const quote = text[j + 1];
      const afterQuote = j + 2;
      const ch = text[afterQuote];
      if (ch === "{" || ch === "[") {
        return {
          prefixIndex: i,
          wrapperStartIndex: j + 1,
          jsonStartIndex: afterQuote,
          wrapperQuote: quote,
          isBytes: true,
        };
      }
    }

    // quoted JSON: '{' or "{"
    if (text[j] === "'" || text[j] === '"') {
      const quote = text[j];
      const afterQuote = j + 1;
      const ch = text[afterQuote];
      if (ch === "{" || ch === "[") {
        return {
          prefixIndex: i,
          wrapperStartIndex: j,
          jsonStartIndex: afterQuote,
          wrapperQuote: quote,
          isBytes: false,
        };
      }
    }

    if (text[j] === "{" || text[j] === "[") {
      return {
        prefixIndex: i,
        wrapperStartIndex: null,
        jsonStartIndex: j,
        wrapperQuote: null,
        isBytes: false,
      };
    }
  }
  return null;
}

function tryParseTruncatedJsonAt(text, startIndex) {
  // We only attempt this if the value looks like an object.
  if (text[startIndex] !== "{") return null;

  const lineEnd = findLineEnd(text, startIndex);
  const candidate = text.slice(startIndex, lineEnd);

  // Common server error payloads:
  // {"timestamp":...,"status":...,"error":"...","trace":"..."... (TRUNCATED)
  const fields = ["trace", "stackTrace", "stacktrace"]; // try a few variants
  let cutIdx = -1;
  let fieldName = "";
  for (const f of fields) {
    const idx = candidate.indexOf(`"${f}"`);
    if (idx !== -1 && (cutIdx === -1 || idx < cutIdx)) {
      cutIdx = idx;
      fieldName = f;
    }
  }
  if (cutIdx === -1) return null;

  // Cut everything from the last comma before the huge field.
  const commaIdx = candidate.lastIndexOf(",", cutIdx);
  if (commaIdx === -1) return null;

  let prefix = candidate.slice(0, commaIdx).trimEnd();
  if (!prefix.endsWith("}")) {
    prefix = prefix + "}";
  }

  const parsed = parseJsonish(prefix);
  if (!parsed) return null;

  // We consumed up to the start of the dropped field (excluding the comma).
  // Keep the raw tail as plain text so nothing is lost.
  return {
    parsed,
    endIndex: startIndex + commaIdx + 1,
    droppedField: fieldName,
  };
}

function findLineEnd(text, startIndex) {
  for (let i = startIndex; i < text.length; i++) {
    const ch = text[i];
    if (ch === "\n" || ch === "\r") return i;
  }
  return text.length;
}

// Best-effort prettifier for values that might be JSON or Python-literal-ish.
// Returns a pretty JSON string, or null if the input can't be parsed.
export function tryPrettifyJsonishValue(text) {
  if (typeof text !== "string") return null;
  const trimmed = text.trim();
  if (trimmed.length === 0) return null;

  // Unwrap python bytes/quoted strings like:
  //   b"[{\"a\":1}]"  -> [{"a":1}]
  //   b'[{"a":1}]'      -> [{"a":1}]
  //   '{"a": 1}'         -> {"a": 1}
  const unwrapped = unwrapPythonBytesOrQuotedString(trimmed);
  if (unwrapped != null) {
    return tryPrettifyJsonishValue(unwrapped);
  }

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
        value: text.slice(i, endIndex),
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
        value: text.slice(i, endIndex),
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
      return {
        wrapperStartIndex: i + 1,
        jsonStartIndex: afterQuote,
        wrapperQuote: quote,
        isBytes: true,
      };
    }
  }

  // Support quoted JSON: key='{"a":1}' or key="{...}"
  if (text[i] === "'" || text[i] === '"') {
    const quote = text[i];
    const afterQuote = i + 1;
    const ch = text[afterQuote];
    if (ch === "{" || ch === "[") {
      return {
        wrapperStartIndex: i,
        jsonStartIndex: afterQuote,
        wrapperQuote: quote,
        isBytes: false,
      };
    }
  }

  // Raw JSON: key={...} / key=[...]
  if (text[i] === "{" || text[i] === "[") {
    return {
      wrapperStartIndex: null,
      jsonStartIndex: i,
      wrapperQuote: null,
      isBytes: false,
    };
  }

  return null;
}

function tryParseJsonishWrappedValue(text, info) {
  const quoteIndex = info.wrapperStartIndex;
  const quote = info.wrapperQuote;
  if (quoteIndex == null || !quote) return null;

  const endQuoteIndex = findMatchingQuoteIndex(text, quoteIndex, quote);
  if (endQuoteIndex === -1) return null;

  const raw = text.slice(quoteIndex + 1, endQuoteIndex);
  const unescaped = unescapePythonString(raw);

  const parsed = parseJsonish(unescaped.trim());
  if (parsed) {
    return { parsed, endIndex: endQuoteIndex + 1 };
  }

  // If the unescaped content is a truncated JSON object, try parsing a safe prefix.
  const truncated = tryParseTruncatedJsonString(unescaped);
  if (truncated) {
    return { parsed: truncated.parsed, endIndex: endQuoteIndex + 1 };
  }

  return null;
}

function findMatchingQuoteIndex(text, quoteStartIndex, quoteChar) {
  let escape = false;
  for (let i = quoteStartIndex + 1; i < text.length; i++) {
    const ch = text[i];
    if (escape) {
      escape = false;
      continue;
    }
    if (ch === "\\") {
      escape = true;
      continue;
    }
    if (ch === quoteChar) return i;
  }
  return -1;
}

function unescapePythonString(value) {
  if (typeof value !== "string" || value.length === 0) return value;

  // Handle common python escape sequences used in repr() of strings/bytes.
  return value.replace(
    /\\(x[0-9A-Fa-f]{2}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8}|n|r|t|\\|'|")/g,
    (m, grp) => {
      if (grp === "n") return "\n";
      if (grp === "r") return "\r";
      if (grp === "t") return "\t";
      if (grp === "\\") return "\\";
      if (grp === "'") return "'";
      if (grp === '"') return '"';
      if (grp[0] === "x")
        return String.fromCharCode(parseInt(grp.slice(1), 16));
      if (grp[0] === "u")
        return String.fromCharCode(parseInt(grp.slice(1), 16));
      if (grp[0] === "U") {
        const codePoint = parseInt(grp.slice(1), 16);
        try {
          return String.fromCodePoint(codePoint);
        } catch {
          return m;
        }
      }
      return m;
    }
  );
}

function unwrapPythonBytesOrQuotedString(trimmed) {
  if (typeof trimmed !== "string" || trimmed.length < 2) return null;

  // bytes literal: b'...' or b"..."
  if (trimmed[0] === "b" && (trimmed[1] === "'" || trimmed[1] === '"')) {
    const q = trimmed[1];
    if (trimmed.endsWith(q)) {
      const inner = trimmed.slice(2, -1);
      return unescapePythonString(inner);
    }
  }

  // quoted string: '...' or "..."
  if (trimmed[0] === "'" || trimmed[0] === '"') {
    const q = trimmed[0];
    if (trimmed.endsWith(q)) {
      const inner = trimmed.slice(1, -1);
      return unescapePythonString(inner);
    }
  }

  return null;
}

function tryParseTruncatedJsonString(unescaped) {
  const trimmed = String(unescaped || "").trim();
  if (!trimmed.startsWith("{")) return null;

  const fields = ["trace", "stackTrace", "stacktrace"];
  let cutIdx = -1;
  let fieldName = "";
  for (const f of fields) {
    const idx = trimmed.indexOf(`"${f}"`);
    if (idx !== -1 && (cutIdx === -1 || idx < cutIdx)) {
      cutIdx = idx;
      fieldName = f;
    }
  }
  if (cutIdx === -1) return null;

  const commaIdx = trimmed.lastIndexOf(",", cutIdx);
  if (commaIdx === -1) return null;

  let prefix = trimmed.slice(0, commaIdx).trimEnd();
  if (!prefix.endsWith("}")) prefix = prefix + "}";

  const parsed = parseJsonish(prefix);
  if (!parsed) return null;
  return { parsed, droppedField: fieldName };
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
