import type { FileSchema } from "../model.js";
import type { EditorSnapshot } from "../store.js";
import { FieldControl } from "./FieldControl.js";

export interface FormEditorProps {
  schema: FileSchema;
  snapshot: EditorSnapshot;
  onChange(key: string, value: string): void;
}

export const FormEditor = ({ schema, snapshot, onChange }: FormEditorProps) => (
  <div class="form-editor">
    {schema.sections.map((section) => (
      <fieldset class="field-section">
        <legend>{section.label}</legend>
        {section.description ? <p class="section-description">{section.description}</p> : false}
        <div class="field-grid">
          {section.fields.map((field) => {
            const state = snapshot.fields[field.key];
            return state ? (
              <FieldControl field={field} state={state} onChange={onChange} />
            ) : false;
          })}
        </div>
      </fieldset>
    ))}
  </div>
);
