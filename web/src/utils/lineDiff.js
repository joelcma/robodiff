// Line-level diff with alignment (side-by-side).
// Designed for prettified JSON strings.

function diffChars(a, b, maxCells) {
  const aStr = String(a ?? "");
  const bStr = String(b ?? "");
  const n = aStr.length;
  const m = bStr.length;
  if (n === 0 && m === 0) {
    return {
      left: [],
      right: [],
    };
  }
  if (n * m > maxCells) {
    return {
      left: [{ text: aStr, type: "equal" }],
      right: [{ text: bStr, type: "equal" }],
    };
  }

  const cols = m + 1;
  const dp = new Uint16Array((n + 1) * cols);
  for (let i = 1; i <= n; i++) {
    for (let j = 1; j <= m; j++) {
      const idx = i * cols + j;
      if (aStr[i - 1] === bStr[j - 1]) {
        dp[idx] = dp[(i - 1) * cols + (j - 1)] + 1;
      } else {
        const up = dp[(i - 1) * cols + j];
        const left = dp[i * cols + (j - 1)];
        dp[idx] = up >= left ? up : left;
      }
    }
  }

  // Backtrack to produce char ops.
  const ops = [];
  let i = n;
  let j = m;
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && aStr[i - 1] === bStr[j - 1]) {
      ops.push({ op: "equal", ch: aStr[i - 1] });
      i -= 1;
      j -= 1;
      continue;
    }
    const up = i > 0 ? dp[(i - 1) * cols + j] : -1;
    const left = j > 0 ? dp[i * cols + (j - 1)] : -1;
    if (i > 0 && (j === 0 || up >= left)) {
      ops.push({ op: "delete", ch: aStr[i - 1] });
      i -= 1;
    } else if (j > 0) {
      ops.push({ op: "insert", ch: bStr[j - 1] });
      j -= 1;
    } else {
      break;
    }
  }
  ops.reverse();

  // Convert ops to segments per side.
  const leftSegs = [];
  const rightSegs = [];

  let lBuf = "";
  let lType = "equal";
  function pushLeft() {
    if (lBuf.length === 0) return;
    leftSegs.push({ text: lBuf, type: lType });
    lBuf = "";
  }

  let rBuf = "";
  let rType = "equal";
  function pushRight() {
    if (rBuf.length === 0) return;
    rightSegs.push({ text: rBuf, type: rType });
    rBuf = "";
  }

  for (const op of ops) {
    if (op.op === "equal") {
      if (lType !== "equal") {
        pushLeft();
        lType = "equal";
      }
      if (rType !== "equal") {
        pushRight();
        rType = "equal";
      }
      lBuf += op.ch;
      rBuf += op.ch;
    } else if (op.op === "delete") {
      if (lType !== "delete") {
        pushLeft();
        lType = "delete";
      }
      lBuf += op.ch;
    } else if (op.op === "insert") {
      if (rType !== "insert") {
        pushRight();
        rType = "insert";
      }
      rBuf += op.ch;
    }
  }

  pushLeft();
  pushRight();

  return { left: leftSegs, right: rightSegs };
}

function alignSimple(aLines, bLines) {
  const max = Math.max(aLines.length, bLines.length);
  const rows = [];

  for (let i = 0; i < max; i++) {
    const a = aLines[i] ?? "";
    const b = bLines[i] ?? "";
    const equal = a === b;
    rows.push({
      kind: equal ? "equal" : "replace",
      left: { text: a, type: equal ? "equal" : "change" },
      right: { text: b, type: equal ? "equal" : "change" },
    });
  }

  return { rows, mode: "simple" };
}

export function diffAlignLines(aText, bText, options = {}) {
  const aLines = String(aText ?? "").split("\n");
  const bLines = String(bText ?? "").split("\n");

  const maxCells = options.maxCells ?? 2_000_000;
  const n = aLines.length;
  const m = bLines.length;

  if (n === 0 && m === 0) return { rows: [], mode: "lcs" };
  if (n * m > maxCells) return alignSimple(aLines, bLines);

  const cols = m + 1;
  const dir = new Uint8Array((n + 1) * cols);
  let prev = new Uint32Array(cols);
  let curr = new Uint32Array(cols);

  for (let i = 1; i <= n; i++) {
    curr[0] = 0;
    for (let j = 1; j <= m; j++) {
      const idx = i * cols + j;
      if (aLines[i - 1] === bLines[j - 1]) {
        curr[j] = prev[j - 1] + 1;
        dir[idx] = 1; // diag
      } else if (prev[j] >= curr[j - 1]) {
        curr[j] = prev[j];
        dir[idx] = 2; // up
      } else {
        curr[j] = curr[j - 1];
        dir[idx] = 3; // left
      }
    }
    const tmp = prev;
    prev = curr;
    curr = tmp;
  }

  const ops = [];
  let i = n;
  let j = m;
  while (i > 0 || j > 0) {
    const idx = i * cols + j;
    const d = dir[idx];

    if (i > 0 && j > 0 && d === 1 && aLines[i - 1] === bLines[j - 1]) {
      ops.push({ op: "equal", a: aLines[i - 1], b: bLines[j - 1] });
      i -= 1;
      j -= 1;
    } else if (i > 0 && (j === 0 || d === 2)) {
      ops.push({ op: "delete", a: aLines[i - 1] });
      i -= 1;
    } else if (j > 0) {
      ops.push({ op: "insert", b: bLines[j - 1] });
      j -= 1;
    } else {
      // Should not happen, but avoid infinite loops.
      break;
    }
  }

  ops.reverse();

  // Coalesce runs so delete/insert pairs become "replace" rows.
  const rows = [];
  const maxCharCells = options.maxCharCells ?? 40_000;

  let idx = 0;
  while (idx < ops.length) {
    if (ops[idx].op === "equal") {
      rows.push({
        kind: "equal",
        left: { text: ops[idx].a, type: "equal" },
        right: { text: ops[idx].b, type: "equal" },
      });
      idx += 1;
      continue;
    }

    const deletes = [];
    const inserts = [];
    while (idx < ops.length && ops[idx].op === "delete") {
      deletes.push(ops[idx].a);
      idx += 1;
    }
    while (idx < ops.length && ops[idx].op === "insert") {
      inserts.push(ops[idx].b);
      idx += 1;
    }

    const pairCount = Math.min(deletes.length, inserts.length);
    for (let p = 0; p < pairCount; p++) {
      const aLine = deletes[p];
      const bLine = inserts[p];
      const parts = diffChars(aLine, bLine, maxCharCells);
      rows.push({
        kind: "replace",
        left: { text: aLine, type: "delete", parts: parts.left },
        right: { text: bLine, type: "insert", parts: parts.right },
      });
    }
    for (let d = pairCount; d < deletes.length; d++) {
      rows.push({
        kind: "delete",
        left: { text: deletes[d], type: "delete" },
        right: { text: "", type: "empty" },
      });
    }
    for (let ins = pairCount; ins < inserts.length; ins++) {
      rows.push({
        kind: "insert",
        left: { text: "", type: "empty" },
        right: { text: inserts[ins], type: "insert" },
      });
    }
  }

  return { rows, mode: "lcs" };
}
