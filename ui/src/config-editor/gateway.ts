import { createAdapterRegistry } from "./adapters/registry.js";
import { fingerprint } from "./fingerprint.js";
import type { ConfigEditorManifest, FileSchema } from "./model.js";
import { ConfigEditorStore } from "./store.js";

export type SaveResult =
  | { status: "saved"; fingerprint: string }
  | { status: "conflict"; remote: string; fingerprint: string };

export interface FileGateway {
  load(path: string): Promise<string>;
  save(
    path: string,
    content: string,
    expectedFingerprint: string,
  ): Promise<SaveResult>;
}

export const MISSING_FILE_FINGERPRINT = "missing";

const isMissingFileError = (error: unknown): boolean => {
  const message = error instanceof Error ? error.message : String(error);
  return /(?:\b404\b|not[ -]?found|missing)/i.test(message);
};

export const withMissingFileFallback = (
  gateway: FileGateway,
  suffixes: string | readonly string[] = [".scroll_template", ".default"],
): FileGateway => {
  const resolvedPaths = new Map<string, string>();
  const fallbackSuffixes = typeof suffixes === "string" ? [suffixes] : suffixes;

  return {
    async load(path) {
      try {
        const content = await gateway.load(path);
        resolvedPaths.set(path, path);
        return content;
      } catch (error) {
        if (!isMissingFileError(error)) throw error;
        const resolved = resolvedPaths.get(path);
        if (resolved && resolved !== path) {
          try {
            return await gateway.load(resolved);
          } catch (fallbackError) {
            if (!isMissingFileError(fallbackError)) throw fallbackError;
          }
        }
        for (const suffix of fallbackSuffixes) {
          const fallback = `${path}${suffix}`;
          if (fallback === resolved) continue;
          try {
            const content = await gateway.load(fallback);
            resolvedPaths.set(path, fallback);
            return content;
          } catch (fallbackError) {
            if (!isMissingFileError(fallbackError)) throw fallbackError;
          }
        }
        throw new Error(`Missing configuration file: ${path}`);
      }
    },
    async save(path, content, expectedFingerprint) {
      const resolved = resolvedPaths.get(path) ?? path;
      if (resolved !== path) {
        const result = await gateway.save(
          path,
          content,
          MISSING_FILE_FINGERPRINT,
        );
        if (result.status === "saved") resolvedPaths.set(path, path);
        return result;
      }
      return await gateway.save(resolved, content, expectedFingerprint);
    },
  };
};

const schemaForPath = (
  manifest: ConfigEditorManifest,
  path: string | undefined,
): FileSchema => {
  const schema = path
    ? manifest.files.find((candidate) => candidate.path === path)
    : manifest.files[0];
  if (!schema) throw new TypeError(path ? `Manifest has no file "${path}".` : "Manifest has no files.");
  return schema;
};

export const loadEditor = async (
  gateway: FileGateway,
  manifest: ConfigEditorManifest,
  path?: string,
): Promise<ConfigEditorStore> => {
  const schema = schemaForPath(manifest, path);
  const adapter = createAdapterRegistry().get(schema.format);
  if (!adapter) throw new TypeError(`No adapter registered for "${schema.format}".`);
  const source = await gateway.load(schema.path);
  return ConfigEditorStore.fromLoadedFile(schema, adapter.parse(source));
};

export const saveSelectedFile = async (
  store: ConfigEditorStore,
  gateway: FileGateway,
): Promise<SaveResult> => {
  if (store.snapshot().issues.some((issue) => issue.severity === "error")) {
    throw new Error("Configuration has validation errors and cannot be saved.");
  }

  const content = store.serializeSelectedFile();
  const serializedIssues = store.validateSerializedSource(content);
  if (serializedIssues.some((issue) => issue.severity === "error")) {
    throw new Error("Serialized configuration failed validation.");
  }

  const expectedFingerprint = await fingerprint(store.loadedSource());
  const result = await gateway.save(store.schema.path, content, expectedFingerprint);
  if (result.status === "conflict") return result;

  const contentFingerprint = await fingerprint(content);
  if (result.fingerprint !== contentFingerprint) {
    throw new Error("Save verification failed: gateway fingerprint does not match content.");
  }
  const remote = await gateway.load(store.schema.path);
  const remoteFingerprint = await fingerprint(remote);
  if (remoteFingerprint !== result.fingerprint || remote !== content) {
    throw new Error("Save verification failed: reloaded content differs from the saved content.");
  }

  store.acceptSavedSource(remote);
  return result;
};
