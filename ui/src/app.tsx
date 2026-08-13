import {
  createConfigEditorComponent,
  fingerprint,
  withMissingFileFallback,
  type FileGateway,
} from "@druid-ui/config-editor";
import {
  loadFileFromDeployment,
  saveFileToDeployment,
} from "@druid-ui/plattform";

const gateway = withMissingFileFallback({
  async load(path) {
    return await loadFileFromDeployment(path);
  },
  async save(path, content, expectedFingerprint) {
    const remote = await loadFileFromDeployment(path);
    const remoteFingerprint = await fingerprint(remote);
    if (remoteFingerprint !== expectedFingerprint) {
      return { status: "conflict", remote, fingerprint: remoteFingerprint };
    }
    await saveFileToDeployment(path, content);
    return { status: "saved", fingerprint: await fingerprint(content) };
  },
} satisfies FileGateway);

export const component = createConfigEditorComponent({
  manifestPath: "private/config-editor.manifest.json",
  gateway,
});
