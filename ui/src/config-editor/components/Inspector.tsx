import { copy } from "../copy.js";
import type { EditorSnapshot } from "../store.js";

export interface InspectorProps {
  snapshot: EditorSnapshot;
}

const printable = (value: unknown): string =>
  value === undefined ? "Not set" : value === null ? "Protected" : String(value);

export const Inspector = ({ snapshot }: InspectorProps) => (
  <aside class="inspector" aria-label={copy.inspectorHeading}>
    <section>
      <h2>{copy.inspectorHeading}</h2>
      {snapshot.unstructuredChanges ? (
        <div class="status-card"><p>{copy.rawChanged}</p></div>
      ) : snapshot.changes.length === 0 ? (
        <div class="status-card"><p>{copy.noChanges}</p></div>
      ) : (
        <ul class="change-list">
          {snapshot.changes.map((change) => (
            <li class="change-item">
              <strong>{change.label}</strong>
              <span class="change-value">
                {change.sensitive
                  ? copy.secretChanged
                  : `${printable(change.before)} → ${printable(change.after)}`}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
    <section class="status-card">
      <h2>{copy.validationHeading}</h2>
      {snapshot.issues.length === 0 ? <p>{copy.valid}</p> : (
        <ul class="issue-list" aria-live="polite">
          {snapshot.issues.map((issue) => <li class="issue-item">{issue.message}</li>)}
        </ul>
      )}
      <p>{copy.unknownKeys}</p>
      <p>{snapshot.restartRequired ? copy.restartRequired : copy.noRestartRequired}</p>
      {snapshot.restartRequired ? <p>{copy.stopBeforeSave}</p> : false}
    </section>
  </aside>
);
