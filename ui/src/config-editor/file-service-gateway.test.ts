import { describe, expect, it, vi } from "vitest";
import { createFileServiceGateway } from "./file-service-gateway.js";

const createService = () => ({
  ensureFileService: vi.fn<() => Promise<void>>().mockResolvedValue(undefined),
  loadFile: vi.fn<(path: string) => Promise<string>>().mockResolvedValue("content"),
  saveFile: vi.fn<(path: string, content: string) => Promise<void>>().mockResolvedValue(undefined),
  fingerprint: vi.fn<(content: string) => Promise<string>>().mockResolvedValue("fingerprint"),
});

describe("createFileServiceGateway", () => {
  it("ensures the file service before loading and saving files", async () => {
    const service = createService();
    const gateway = createFileServiceGateway(service);

    await expect(gateway.load("server.properties")).resolves.toBe("content");
    await expect(gateway.save("server.properties", "content", "ignored")).resolves.toEqual({
      status: "saved",
      fingerprint: "fingerprint",
    });

    expect(service.ensureFileService).toHaveBeenCalledOnce();
    expect(service.ensureFileService).toHaveBeenCalledWith(false);
    expect(service.loadFile).toHaveBeenCalledWith("server.properties");
    expect(service.saveFile).toHaveBeenCalledWith("server.properties", "content");
  });

  it("shares startup between concurrent file operations", async () => {
    let resolveStartup: (() => void) | undefined;
    const service = createService();
    service.ensureFileService.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveStartup = resolve;
        })
    );
    const gateway = createFileServiceGateway(service);

    const load = gateway.load("server.properties");
    const save = gateway.save("server.properties", "content", "ignored");
    await vi.waitFor(() => expect(service.ensureFileService).toHaveBeenCalledOnce());
    resolveStartup?.();

    await Promise.all([load, save]);
    expect(service.ensureFileService).toHaveBeenCalledWith(false);
  });

  it("retries startup after a failure", async () => {
    const service = createService();
    service.ensureFileService.mockRejectedValueOnce(new Error("unavailable"));
    const gateway = createFileServiceGateway(service);

    await expect(gateway.load("server.properties")).rejects.toThrow("unavailable");
    await expect(gateway.load("server.properties")).resolves.toBe("content");

    expect(service.ensureFileService).toHaveBeenCalledTimes(2);
  });
});
