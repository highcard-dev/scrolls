import { describe, expect, it } from "vitest";
import type { FieldSchema } from "./model.js";
import { coerceFieldValue, validateField } from "./validation.js";

const field = (overrides: Partial<FieldSchema> = {}): FieldSchema => ({
  key: "max-players",
  label: "Max players",
  description: "Maximum player count.",
  documentation: "https://minecraft.wiki/w/Server.properties",
  type: "integer",
  min: 1,
  max: 1000,
  ...overrides,
});

describe("coerceFieldValue", () => {
  it("strictly coerces supported scalar inputs", () => {
    expect(coerceFieldValue(field(), "20")).toBe(20);
    expect(coerceFieldValue(field({ type: "number" }), "1.25")).toBe(1.25);
    expect(coerceFieldValue(field({ type: "boolean", min: undefined, max: undefined }), "true")).toBe(true);
    expect(coerceFieldValue(field({ type: "boolean", min: undefined, max: undefined }), "false")).toBe(false);
    expect(coerceFieldValue(field({ type: "string", min: undefined, max: undefined }), "")).toBe("");
  });

  it.each([
    ["True", true],
    ["FALSE", false],
    ["  true  ", true],
  ] as const)("coerces common configuration boolean %j", (input, expected) => {
    expect(
      coerceFieldValue(
        field({ type: "boolean", min: undefined, max: undefined }),
        input,
      ),
    ).toBe(expected);
  });

  it.each(["yes", "1", ""])("rejects non-boolean value %j", (input) => {
    expect(() =>
      coerceFieldValue(
        field({ type: "boolean", min: undefined, max: undefined }),
        input,
      ),
    ).toThrow(/true or false/);
  });

  it.each([
    ["5000.000000", 5000],
  ] as const)("coerces numerically integral configuration value %j", (input, expected) => {
    expect(coerceFieldValue(field(), input)).toBe(expected);
  });

  it.each(["1.5", "5e3", "NaN", "Infinity", "", "  "])(
    "rejects invalid integer %j",
    (input) => expect(() => coerceFieldValue(field(), input)).toThrow(/integer/),
  );
});

describe("validateField", () => {
  it("rejects an integer outside its inclusive range", () => {
    expect(validateField(field(), 0)).toEqual([
      expect.objectContaining({
        code: "range",
        message: "Value must be between 1 and 1000 (inclusive).",
      }),
    ]);
  });

  it("enforces enum membership and a full-string pattern", () => {
    expect(
      validateField(
        field({ type: "enum", values: ["peaceful", "hard"], min: undefined, max: undefined }),
        "unknown",
      ),
    ).toEqual([expect.objectContaining({ code: "enum" })]);
    expect(
      validateField(
        field({ type: "string", pattern: "[a-z]+", min: undefined, max: undefined }),
        "abc123",
      ),
    ).toEqual([expect.objectContaining({ code: "pattern" })]);
  });
});
