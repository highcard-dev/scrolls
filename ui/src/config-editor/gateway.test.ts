import { describe, expect, it } from "vitest";
import { fingerprint } from "./fingerprint.js";
import {
  loadEditor,
  saveSelectedFile,
  withMissingFileFallback,
  MISSING_FILE_FINGERPRINT,
  type FileGateway,
  type SaveResult,
} from "./gateway.js";
import { validateManifest } from "./manifest.js";
import { validManifest } from "./test-fixtures.js";

const manifestFor = (path: string) => validateManifest(validManifest(path));

const memoryGateway = (initial: Record<string, string>) => {
  const files = new Map(Object.entries(initial));
  const gateway = {
    async load(path: string) {
      const value = files.get(path);
      if (value === undefined) throw new Error(`Missing fixture file: ${path}`);
      return value;
    },
    async save(
      path: string,
      content: string,
      expectedFingerprint: string,
    ): Promise<SaveResult> {
      const remote = files.get(path);
      if (remote === undefined && expectedFingerprint === MISSING_FILE_FINGERPRINT) {
        files.set(path, content);
        return { status: "saved", fingerprint: await fingerprint(content) };
      }
      if (remote === undefined) throw new Error(`Missing fixture file: ${path}`);
      const remoteFingerprint = await fingerprint(remote);
      if (remoteFingerprint !== expectedFingerprint) {
        return { status: "conflict", remote, fingerprint: remoteFingerprint };
      }
      files.set(path, content);
      return { status: "saved", fingerprint: await fingerprint(content) };
    },
    replace(path: string, content: string) {
      files.set(path, content);
    },
    current(path: string) {
      return files.get(path);
    },
  } satisfies FileGateway & {
    replace(path: string, content: string): void;
    current(path: string): string | undefined;
  };
  return gateway;
};

describe("saveSelectedFile", () => {
  it("blocks overwrite when the remote fingerprint changed", async () => {
    const gateway = memoryGateway({ "server.properties": "max-players=20\n" });
    const store = await loadEditor(gateway, manifestFor("server.properties"));
    store.setDisplayValue("max-players", "50");
    gateway.replace("server.properties", "max-players=30\n");

    await expect(saveSelectedFile(store, gateway)).resolves.toEqual(
      expect.objectContaining({
        status: "conflict",
        remote: "max-players=30\n",
      }),
    );
    expect(gateway.current("server.properties")).toBe("max-players=30\n");
    expect(store.snapshot().dirty).toBe(true);
  });

  it("reparses and verifies content after a successful save", async () => {
    const gateway = memoryGateway({ "server.properties": "max-players=20\n" });
    const store = await loadEditor(gateway, manifestFor("server.properties"));
    store.setDisplayValue("max-players", "50");
    await expect(saveSelectedFile(store, gateway)).resolves.toEqual(
      expect.objectContaining({ status: "saved" }),
    );
    expect(gateway.current("server.properties")).toBe("max-players=50\n");
    expect(store.snapshot().dirty).toBe(false);
  });

  it("refuses to save while the visible form has validation errors", async () => {
    const gateway = memoryGateway({ "server.properties": "max-players=20\n" });
    const store = await loadEditor(gateway, manifestFor("server.properties"));
    store.setDisplayValue("max-players", "0");
    await expect(saveSelectedFile(store, gateway)).rejects.toThrow(
      /validation errors/,
    );
    expect(gateway.current("server.properties")).toBe("max-players=20\n");
  });

  it("detects a gateway that acknowledges a save but returns different content", async () => {
    const gateway = memoryGateway({ "server.properties": "max-players=20\n" });
    const brokenGateway: FileGateway = {
      load: gateway.load.bind(gateway),
      async save(path, _content, _expectedFingerprint) {
        gateway.replace(path, "max-players=999\n");
        return { status: "saved", fingerprint: await fingerprint("max-players=50\n") };
      },
    };
    const store = await loadEditor(brokenGateway, manifestFor("server.properties"));
    store.setDisplayValue("max-players", "50");
    await expect(saveSelectedFile(store, brokenGateway)).rejects.toThrow(
      /verification failed/,
    );
  });
});

describe("withMissingFileFallback", () => {
  it("edits a packaged default before the active configuration exists", async () => {
    const memory = memoryGateway({
      "data/server.properties.default": "max-players=20\n",
    });
    const gateway = withMissingFileFallback(memory);
    const store = await loadEditor(gateway, manifestFor("data/server.properties"));

    store.setDisplayValue("max-players", "42");
    await expect(saveSelectedFile(store, gateway)).resolves.toEqual(
      expect.objectContaining({ status: "saved" }),
    );

    expect(memory.current("data/server.properties")).toBe("max-players=42\n");
    expect(memory.current("data/server.properties.default")).toBe("max-players=20\n");
  });

  it("edits a scroll template before the active configuration exists", async () => {
    const memory = memoryGateway({
      "data/server.properties.scroll_template": "max-players=20\n",
    });
    const gateway = withMissingFileFallback(memory);
    const store = await loadEditor(gateway, manifestFor("data/server.properties"));

    store.setDisplayValue("max-players", "42");
    await expect(saveSelectedFile(store, gateway)).resolves.toEqual(
      expect.objectContaining({ status: "saved" }),
    );

    expect(memory.current("data/server.properties")).toBe("max-players=42\n");
    expect(memory.current("data/server.properties.scroll_template")).toBe("max-players=20\n");
  });

  it("prefers an active configuration over its template", async () => {
    const memory = memoryGateway({
      "data/server.properties": "max-players=30\n",
      "data/server.properties.scroll_template": "max-players=20\n",
    });
    const store = await loadEditor(
      withMissingFileFallback(memory),
      manifestFor("data/server.properties"),
    );

    expect(store.snapshot().fields["max-players"]?.displayValue).toBe("30");
  });

  it("switches from a fallback to an active file when the runtime creates it", async () => {
    const memory = memoryGateway({
      "data/server.properties.default": "max-players=20\n",
    });
    const gateway = withMissingFileFallback(memory);

    await expect(gateway.load("data/server.properties")).resolves.toBe("max-players=20\n");
    memory.replace("data/server.properties", "max-players=30\n");
    await expect(gateway.load("data/server.properties")).resolves.toBe("max-players=30\n");
  });

  it("fails closed when an active file appears before a fallback save", async () => {
    const memory = memoryGateway({
      "data/server.properties.default": "max-players=20\n",
    });
    const gateway = withMissingFileFallback(memory);
    const store = await loadEditor(gateway, manifestFor("data/server.properties"));
    store.setDisplayValue("max-players", "42");
    memory.replace("data/server.properties", "max-players=30\n");

    await expect(saveSelectedFile(store, gateway)).resolves.toEqual(
      expect.objectContaining({ status: "conflict", remote: "max-players=30\n" }),
    );
    expect(memory.current("data/server.properties")).toBe("max-players=30\n");
    expect(memory.current("data/server.properties.default")).toBe("max-players=20\n");
  });

  it("does not hide non-missing-file load failures", async () => {
    const gateway: FileGateway = {
      async load() {
        throw new Error("permission denied");
      },
      async save() {
        throw new Error("unexpected save");
      },
    };

    await expect(
      withMissingFileFallback(gateway).load("data/server.properties"),
    ).rejects.toThrow("permission denied");
  });

  it("reports the declared path after every fallback is missing", async () => {
    const gateway = withMissingFileFallback(memoryGateway({}));

    await expect(gateway.load("data/generated.cfg")).rejects.toThrow(
      "Missing configuration file: data/generated.cfg",
    );
  });
});
