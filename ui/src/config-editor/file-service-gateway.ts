import type { FileGateway, SaveResult } from "./gateway.js";

export interface DeploymentFileService {
  ensureFileService(watch?: boolean): Promise<void>;
  loadFile(path: string): Promise<string>;
  saveFile(path: string, content: string): Promise<void>;
  fingerprint(content: string): Promise<string>;
}

/**
 * Ensures WebDAV once before using deployment files. A failed startup is not
 * cached, allowing the next user action to retry it.
 */
export const createFileServiceGateway = (service: DeploymentFileService): FileGateway => {
  let ready: Promise<void> | undefined;

  const ensureReady = (): Promise<void> => {
    if (ready) return ready;

    let startup: Promise<void>;
    startup = Promise.resolve()
      .then(() => service.ensureFileService(false))
      .catch((error: unknown) => {
        if (ready === startup) ready = undefined;
        throw error;
      });
    ready = startup;
    return startup;
  };

  return {
    async load(path) {
      await ensureReady();
      return service.loadFile(path);
    },
    async save(path, content): Promise<SaveResult> {
      await ensureReady();
      await service.saveFile(path, content);
      return { status: "saved", fingerprint: await service.fingerprint(content) };
    },
  };
};
