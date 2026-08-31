import {
  createConfigEditorComponent,
  fingerprint,
  withMissingFileFallback,
  type FileGateway,
} from "./config-editor/index.js";
import {
  loadFileFromDeployment,
  saveFileToDeployment,
} from "@druid-ui/plattform";

const requireString = (value: unknown, operation: string): string => {
  if (typeof value !== "string") {
    throw new TypeError(`${operation} returned a non-string response.`);
  }
  return value;
};

const gateway = withMissingFileFallback({
  async load(path) {
    return requireString(await loadFileFromDeployment(path), "loadFileFromDeployment");
  },
  async save(path, content) {
    await saveFileToDeployment(path, content);
    return { status: "saved", fingerprint: await fingerprint(content) };
  },
} satisfies FileGateway);

export const component = createConfigEditorComponent({
  manifestPath: ".druid/config-editor.manifest.json",
  gateway,
});
