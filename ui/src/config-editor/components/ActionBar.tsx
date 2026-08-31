import { copy } from "../copy.js";

export interface ActionBarProps {
  dirty: boolean;
  invalid: boolean;
  restartRequired: boolean;
  saving: boolean;
  status: string;
  onSave(): Promise<void> | void;
}

export const ActionBar = ({
  dirty,
  invalid,
  restartRequired,
  saving,
  status,
  onSave,
}: ActionBarProps) => (
  <div class="action-bar">
    <span class="action-feedback">
      <span class="action-status" aria-live="polite">{status}</span>
      {restartRequired ? (
        <span class="restart-notice">{copy.restartAfterSave}</span>
      ) : false}
    </span>
    <button
      type="button"
      class="save-button"
      disabled={!dirty || invalid || saving}
      onClick={() => onSave()}
    >
      {saving ? copy.saving : copy.save}
    </button>
  </div>
);
