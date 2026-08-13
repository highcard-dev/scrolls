import {
  createConfigEditorComponent,
  fingerprint,
  type FileGateway,
} from "@druid-ui/config-editor";
import {
  loadFileFromDeployment,
  saveFileToDeployment,
} from "@druid-ui/plattform";

const gateway: FileGateway = {
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
};

export const component = createConfigEditorComponent({
  manifestPath: "private/config-editor.manifest.json",
  gateway,
});
