import type { ConfigEditorManifest } from "../model.js";
import { copy } from "../copy.js";

export interface FileRailProps {
  manifest: ConfigEditorManifest;
  selectedPath: string;
  onSelect(path: string): Promise<void> | void;
}

export const FileRail = ({ manifest, selectedPath, onSelect }: FileRailProps) => (
  <nav class="file-rail" aria-label={copy.filesHeading}>
    <h2 class="rail-heading">{copy.filesHeading}</h2>
    <div class="file-list">
      {manifest.files.map((file) => (
        <button
          type="button"
          class="file-button"
          aria-current={file.path === selectedPath ? "true" : "false"}
          onClick={() => onSelect(file.path)}
        >
          <span class="file-path">{file.label}</span>
          <span class="file-format">{file.format}</span>
        </button>
      ))}
    </div>
  </nav>
);
