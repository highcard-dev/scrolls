import { describe, expect, it } from "vitest";
import { fingerprint } from "./fingerprint.js";

describe("fingerprint", () => {
  it("returns the standard SHA-256 digest without platform APIs", async () => {
    await expect(fingerprint("abc")).resolves.toBe(
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    );
  });

  it("fingerprints exact UTF-8 source bytes including line endings", async () => {
    await expect(fingerprint("Grüße\n")).resolves.not.toBe(
      await fingerprint("Grüße\r\n"),
    );
  });
});
