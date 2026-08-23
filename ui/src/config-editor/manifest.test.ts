import { describe, expect, it } from "vitest";
import { fieldsForVersion, validateManifest } from "./manifest.js";
import {
  duplicateFieldManifest,
  duplicateManifest,
  validManifest,
  versionedFile,
} from "./test-fixtures.js";

describe("validateManifest", () => {
  it("accepts a versioned manifest and preserves stable field keys", () => {
    const result = validateManifest({
      version: 1,
      server: {
        family: "minecraft",
        displayName: "Minecraft Server",
        appVersion: "1.21.7",
      },
      files: [
        {
          path: "server.properties",
          format: "java-properties",
          label: "server.properties",
          sections: [
            {
              id: "players",
              label: "Players",
              fields: [
                {
                  key: "max-players",
                  label: "Max players",
                  type: "integer",
                  min: 1,
                  max: 1000,
                  description: "Maximum player count.",
                  documentation: "https://minecraft.wiki/w/Server.properties",
                  restartRequired: true,
                },
              ],
            },
          ],
        },
      ],
    });

    expect(result.files[0]!.sections[0]!.fields[0]!.key).toBe("max-players");
  });

  it.each([
    "../secret",
    "/absolute",
    "C:/server.properties",
    "https://host/file",
    "folder\\file",
  ])("rejects unsafe file path %s", (path) => {
    expect(() => validateManifest(validManifest(path))).toThrow(
      /safe relative path/,
    );
  });

  it("rejects duplicate file paths", () => {
    expect(() => validateManifest(duplicateManifest())).toThrow(/duplicate/i);
  });

  it("rejects duplicate field keys within a file", () => {
    expect(() => validateManifest(duplicateFieldManifest())).toThrow(
      /duplicate/i,
    );
  });

  it("accepts a file with no typed fields for a lossless raw-only editor", () => {
    const manifest = validManifest();
    manifest.files[0]!.sections = [];

    const result = validateManifest(manifest);

    expect(result.files[0]!.sections).toEqual([]);
  });

  it("rejects inconsistent enum and numeric constraints", () => {
    const manifest = validManifest();
    const field = manifest.files[0]!.sections[0]!.fields[0]!;
    field.type = "enum";
    delete field.min;
    delete field.max;
    expect(() => validateManifest(manifest)).toThrow(/enum values/);

    const ranged = validManifest();
    ranged.files[0]!.sections[0]!.fields[0]!.min = 10;
    ranged.files[0]!.sections[0]!.fields[0]!.max = 1;
    expect(() => validateManifest(ranged)).toThrow(/minimum/i);
  });
});

describe("fieldsForVersion", () => {
  it("includes fields only inside their inclusive version range", () => {
    expect(
      fieldsForVersion(versionedFile(), "1.20.4").map((field) => field.key),
    ).toEqual(["always", "legacy"]);
    expect(
      fieldsForVersion(versionedFile(), "1.21.1").map((field) => field.key),
    ).toEqual(["always", "modern"]);
  });
});
