import { describe, expect, it } from "vitest";
import { jsonAdapter } from "./json.js";

describe("jsonAdapter", () => {
  it("updates a JSON scalar without reformatting siblings", () => {
    const source =
      '{\n  "server": { "name": "Old", "players": 20 },\n  "future": true\n}\n';
    const changed = jsonAdapter.set(
      jsonAdapter.parse(source),
      "/server/players",
      50,
    );
    expect(jsonAdapter.serialize(changed)).toBe(
      '{\n  "server": { "name": "Old", "players": 50 },\n  "future": true\n}\n',
    );
    expect(jsonAdapter.get(changed, "/server/name")).toBe("Old");
  });

  it("supports escaped JSON pointer segments and array indexes", () => {
    const document = jsonAdapter.parse(
      '{"a/b":{"~key":[false,null,"value"]}}',
    );
    expect(jsonAdapter.get(document, "/a~1b/~0key/0")).toBe(false);
    expect(jsonAdapter.get(document, "/a~1b/~0key/1")).toBe(null);
    expect(jsonAdapter.get(document, "/a~1b/~0key/2")).toBe("value");
  });

  it("reports invalid JSON with a line and column", () => {
    const document = jsonAdapter.parse('{\n  "broken":,\n}\n');
    expect(document.issues).toEqual([
      expect.objectContaining({
        code: "invalid-json",
        severity: "error",
        line: 2,
        column: 12,
      }),
    ]);
  });

  it("rejects writes to objects and properties whose parent is missing", () => {
    const document = jsonAdapter.parse('{"server":{"players":20}}');
    expect(() => jsonAdapter.set(document, "/server", "invalid")).toThrow(
      /scalar/,
    );
    expect(() => jsonAdapter.set(document, "/missing/nested", 1)).toThrow(/pointer/);
  });

  it("adds missing scalar properties without reformatting existing JSON", () => {
    const source = '{\n  "server": {\n    "name": "Druid"\n  }\n}\n';
    const changed = jsonAdapter.set(jsonAdapter.parse(source), "/server/port", 5520);

    expect(jsonAdapter.serialize(changed)).toBe(
      '{\n  "server": {\n    "name": "Druid",\n    "port": 5520\n  }\n}\n',
    );
    expect(jsonAdapter.get(changed, "/server/port")).toBe(5520);
  });

  it("preserves CRLF line endings while adding a missing property", () => {
    const source = "{\r\n  \"server\": {\r\n    \"name\": \"Druid\"\r\n  }\r\n}\r\n";
    const changed = jsonAdapter.set(jsonAdapter.parse(source), "/server/port", 5520);

    expect(jsonAdapter.serialize(changed)).toBe(
      "{\r\n  \"server\": {\r\n    \"name\": \"Druid\",\r\n    \"port\": 5520\r\n  }\r\n}\r\n",
    );
  });

  it("keeps unsafe JSON integer identifiers exact and editable as strings", () => {
    const source = '{"steamId":765611980123456789}\n';
    const document = jsonAdapter.parse(source);
    expect(jsonAdapter.get(document, "/steamId")).toBe("765611980123456789");
    expect(jsonAdapter.serialize(document)).toBe(source);

    const changed = jsonAdapter.set(document, "/steamId", "765611980123456780");
    expect(jsonAdapter.serialize(changed)).toBe('{"steamId":765611980123456780}\n');
  });
});
