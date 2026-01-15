import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  splitTextByJsonAssignments,
  tryExtractJsonishComparison,
  tryPrettifyJsonishValue,
} from "./jsonPrettify.js";

describe("jsonPrettify", () => {
  it("prettifies python bytes-literal JSON arrays (b'[...]') and decodes unicode escapes", () => {
    const raw =
      'b\'[{"rowMetadata": {"description": "0100+0001 pisteen description kentt\\\\u00e4"}}]\'';

    const pretty = tryPrettifyJsonishValue(raw);
    assert.equal(typeof pretty, "string");

    const parsed = JSON.parse(pretty);
    assert.equal(Array.isArray(parsed), true);
    assert.ok(String(parsed[0].rowMetadata.description).includes("kenttä"));
  });

  it("splits assignment text and extracts body=b'[...]' as JSON segment", () => {
    const msg =
      "POST Request : url=http://localhost:8083/api/x " +
      "path_url=/api/x " +
      "headers={'Accept': 'application/json', 'Connection': 'keep-alive'} " +
      'body=b\'[{"a": 1, "text": "kentt\\\\u00e4"}]\'';

    const segments = splitTextByJsonAssignments(msg);

    const bodySeg = segments.find((s) => s.type === "json" && s.key === "body");
    assert.ok(bodySeg);

    const parsedBody = JSON.parse(bodySeg.pretty);
    assert.equal(parsedBody[0].a, 1);
    assert.equal(parsedBody[0].text, "kenttä");

    const headersSeg = segments.find(
      (s) => s.type === "json" && s.key === "headers"
    );
    assert.ok(headersSeg);
    const parsedHeaders = JSON.parse(headersSeg.pretty);
    assert.equal(parsedHeaders.Accept, "application/json");
  });

  it("extracts and prettifies bytes-literal JSON comparisons", () => {
    const text = "b'[{\"a\": 1}]' != b'[{\"a\": 2}]'";
    const comparison = tryExtractJsonishComparison(text);
    assert.ok(comparison);
    assert.equal(comparison.operator, "!=");

    const left = JSON.parse(comparison.left.pretty);
    const right = JSON.parse(comparison.right.pretty);
    assert.equal(left[0].a, 1);
    assert.equal(right[0].a, 2);
  });
});
