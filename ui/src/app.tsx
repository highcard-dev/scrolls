import {
  createConfigEditorComponent,
  withMissingFileFallback,
  type FileGateway,
  type SaveResult,
} from "./config-editor/index.js";
import {
  loadFileFromDeployment,
  saveFileToDeploymentIfMatch,
} from "@druid-ui/plattform";

const requireString = (value: unknown, operation: string): string => {
  if (typeof value !== "string") {
    throw new TypeError(`${operation} returned a non-string response.`);
  }
  return value;
};

const parseSaveResult = (value: unknown): SaveResult => {
  const result: unknown = JSON.parse(
    requireString(value, "saveFileToDeploymentIfMatch"),
  );
  if (!result || typeof result !== "object") {
    throw new TypeError("Invalid configuration save response.");
  }
  const candidate = result as Record<string, unknown>;
  const validSaved =
    candidate.status === "saved" && typeof candidate.fingerprint === "string";
  const validConflict =
    candidate.status === "conflict" &&
    typeof candidate.remote === "string" &&
    typeof candidate.fingerprint === "string";
  if (!validSaved && !validConflict) {
    throw new TypeError("Invalid configuration save response.");
  }
  return result as SaveResult;
};

const gateway = withMissingFileFallback({
  async load(path) {
    return requireString(await loadFileFromDeployment(path), "loadFileFromDeployment");
  },
  async save(path, content, expectedFingerprint) {
    return parseSaveResult(
      await saveFileToDeploymentIfMatch(path, content, expectedFingerprint),
    );
  },
} satisfies FileGateway);

export const component = createConfigEditorComponent({
  manifestPath: "private/config-editor.manifest.json",
  gateway,
});
