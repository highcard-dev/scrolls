import { describe, expect, it } from "vitest";
import { rawAdapter } from "./raw.js";

describe("rawAdapter", () => {
  it("round-trips arbitrary content byte-for-byte", () => {
    const source = "arbitrary\r\n\0content\n";
    expect(rawAdapter.serialize(rawAdapter.parse(source))).toBe(source);
  });

  it("blocks typed writes through the raw fallback", () => {
    const document = rawAdapter.parse("arbitrary content\n");
    expect(() => rawAdapter.set(document, "key", "value")).toThrow(/Raw mode/);
  });
});
