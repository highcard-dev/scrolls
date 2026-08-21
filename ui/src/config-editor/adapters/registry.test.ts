import { describe, expect, it } from "vitest";
import { createAdapterRegistry } from "./registry.js";

describe("createAdapterRegistry", () => {
  it("provides an adapter for every supported manifest format", () => {
    const registry = createAdapterRegistry();
    expect([...registry.keys()]).toEqual([
      "java-properties",
      "ini",
      "unreal-ini",
      "key-value",
      "json",
      "xml-properties",
      "raw",
    ]);
  });
});
