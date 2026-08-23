import type { Event } from "@druid-ui/component";
import { copy } from "../copy.js";

export interface RawEditorProps {
  source: string;
  onChange(source: string): void;
}

export const RawEditor = ({ source, onChange }: RawEditorProps) => (
  <div class="raw-editor">
    <label for="config-raw-source">{copy.rawLabel}</label>
    <textarea
      id="config-raw-source"
      class="raw-textarea"
      aria-label={copy.rawLabel}
      spellcheck="false"
      value={source}
      onInput={(event: Event) => onChange(event.value())}
    />
  </div>
);
