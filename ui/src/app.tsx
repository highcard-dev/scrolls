import {
  createConfigEditorComponent,
  withMissingFileFallback,
  type FileGateway,
} from "@druid-ui/config-editor";
import {
  loadFileFromDeployment,
  saveFileToDeploymentIfMatch,
} from "@druid-ui/plattform";

const gateway = withMissingFileFallback({
  async load(path) {
    return await loadFileFromDeployment(path);
  },
  async save(path, content, expectedFingerprint) {
    return JSON.parse(
      await saveFileToDeploymentIfMatch(path, content, expectedFingerprint),
    );
  },
} satisfies FileGateway);

export const component = createConfigEditorComponent({
  manifestPath: "private/config-editor.manifest.json",
  gateway,
});
