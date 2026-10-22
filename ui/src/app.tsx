import {
  createConfigEditorComponent,
  fingerprint,
  withMissingFileFallback,
} from "./config-editor/index.js";
import {
  ensureFileService,
  loadFileFromDeployment,
  saveFileToDeployment,
} from "@druid-ui/plattform";
import { createFileServiceGateway } from "./config-editor/file-service-gateway.js";

const requireString = (value: unknown, operation: string): string => {
  if (typeof value !== "string") {
    throw new TypeError(`${operation} returned a non-string response.`);
  }
  return value;
};

const gateway = withMissingFileFallback(createFileServiceGateway({
  ensureFileService,
  async loadFile(path) {
    return requireString(await loadFileFromDeployment(path), "loadFileFromDeployment");
  },
  async saveFile(path, content) {
    await saveFileToDeployment(path, content);
  },
  async fingerprint(content) {
    return fingerprint(content);
  },
}));

export const component = createConfigEditorComponent({
  manifestPath: ".druid/config-editor.manifest.json",
  gateway,
});
