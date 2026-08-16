import type { Event } from "@druid-ui/component";
import type { FieldSchema } from "../model.js";
import type { FieldSnapshot } from "../store.js";
import { copy } from "../copy.js";

export interface FieldControlProps {
  field: FieldSchema;
  state: FieldSnapshot;
  onChange(key: string, value: string): void;
}

export const FieldControl = ({ field, state, onChange }: FieldControlProps) => {
  const inputId = `config-field-${field.key.replace(/[^a-zA-Z0-9_-]/g, "-")}`;
  const descriptionId = `${inputId}-description`;
  const input = field.type === "enum" ? (
    <select
      id={inputId}
      class="field-input"
      aria-label={field.label}
      aria-describedby={descriptionId}
      aria-invalid={state.issues.length > 0 ? "true" : "false"}
      value={state.displayValue}
      onChange={(event: Event) => onChange(field.key, event.value())}
    >
      {(field.values ?? []).map((value) => (
        <option value={value} selected={state.displayValue === value}>{value}</option>
      ))}
    </select>
  ) : field.type === "boolean" ? (
    <span class="boolean-control">
      <input
        id={inputId}
        type="checkbox"
        aria-label={field.label}
        aria-describedby={descriptionId}
        checked={state.displayValue === "true"}
        onChange={(event: Event) =>
          onChange(field.key, event.checked() ? "true" : "false")
        }
      />
      <span>{state.displayValue === "true" ? "Enabled" : "Disabled"}</span>
    </span>
  ) : (
    <input
      id={inputId}
      class="field-input"
      aria-label={field.label}
      aria-describedby={descriptionId}
      type={field.type === "secret" || field.sensitive ? "password" : field.type === "integer" || field.type === "number" ? "number" : "text"}
      step={field.type === "integer" ? "1" : field.type === "number" ? "any" : undefined}
      min={field.min}
      max={field.max}
      value={state.displayValue}
      aria-invalid={state.issues.length > 0 ? "true" : "false"}
      autocomplete={field.type === "secret" || field.sensitive ? "new-password" : "off"}
      onInput={(event: Event) => onChange(field.key, event.value())}
    />
  );

  return (
    <div class="field-control">
      <label for={inputId}>{field.label}</label>
      <p id={descriptionId} class="field-description">{field.description}</p>
      {input}
      <div class="field-meta">
        {field.restartRequired ? <span class="restart-chip">{copy.restartRequired}</span> : false}
        <span>{field.key}</span>
      </div>
      {state.issues.map((issue) => (
        <p class="field-error" role="alert">{issue.message}</p>
      ))}
    </div>
  );
};
