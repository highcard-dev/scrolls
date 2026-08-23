export const EDITOR_STYLES = `
:host {
  display: block;
  width: 100%;
  --druid-bg: #07100d;
  --druid-panel: #101b17;
  --druid-panel-raised: #15231d;
  --druid-border: #294137;
  --druid-border-strong: #3f6655;
  --druid-text: #f4f7f5;
  --druid-muted: #9eb0a8;
  --druid-accent: #a8ef9c;
  --druid-accent-ink: #10210f;
  --druid-warning: #f0c96b;
  --druid-error: #ff8b7f;
  color: var(--druid-text);
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color-scheme: dark;
}
* { box-sizing: border-box; }
button, input, select, textarea { font: inherit; }
button, input, select, textarea, a { outline: none; }
button:focus-visible, input:focus-visible, select:focus-visible, textarea:focus-visible, a:focus-visible {
  box-shadow: 0 0 0 3px rgba(168, 239, 156, .28);
  border-color: var(--druid-accent);
}
.config-editor { min-height: 640px; background: radial-gradient(circle at 75% 0%, #173326 0, transparent 32%), var(--druid-bg); color: var(--druid-text); }
.editor-header { min-height: 76px; display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 16px 22px; border-bottom: 1px solid var(--druid-border); }
.eyebrow { color: var(--druid-accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; text-transform: uppercase; }
.editor-title { margin: 3px 0 0; font-size: clamp(20px, 2.4vw, 30px); line-height: 1.1; }
.server-version { color: var(--druid-muted); font-size: 13px; }
.editor-grid { display: grid; grid-template-columns: minmax(210px, 250px) minmax(420px, 1fr) minmax(260px, 320px); min-height: 564px; }
.file-rail, .inspector { background: rgba(11, 21, 17, .8); }
.file-rail { padding: 18px 14px; border-right: 1px solid var(--druid-border); }
.rail-heading, .inspector h2 { margin: 0 0 12px; font-size: 12px; color: var(--druid-muted); letter-spacing: .08em; text-transform: uppercase; }
.file-list { display: grid; gap: 8px; }
.file-button { width: 100%; min-height: 52px; display: grid; gap: 3px; padding: 10px 12px; text-align: left; border: 1px solid transparent; border-radius: 10px; background: transparent; color: var(--druid-text); cursor: pointer; }
.file-button:hover { background: var(--druid-panel); }
.file-button[aria-current="true"] { border-color: var(--druid-border-strong); background: var(--druid-panel-raised); }
.file-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 700; }
.file-format { color: var(--druid-muted); font-size: 12px; text-transform: uppercase; }
.editor-main { min-width: 0; padding: 18px 22px 92px; }
.tabs { display: inline-flex; gap: 4px; padding: 4px; margin-bottom: 18px; border: 1px solid var(--druid-border); border-radius: 10px; background: #0b1511; }
.tab { min-height: 40px; min-width: 92px; border: 0; border-radius: 7px; background: transparent; color: var(--druid-muted); cursor: pointer; }
.tab[aria-selected="true"] { background: var(--druid-panel-raised); color: var(--druid-text); }
.form-editor { display: grid; gap: 18px; }
.field-section { min-width: 0; margin: 0; padding: 18px; border: 1px solid var(--druid-border); border-radius: 14px; background: rgba(16, 27, 23, .84); }
.field-section legend { padding: 0 8px; color: var(--druid-accent); font-weight: 800; }
.section-description { margin: 0 0 16px; color: var(--druid-muted); font-size: 13px; }
.field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.field-control { min-width: 0; display: grid; align-content: start; gap: 7px; }
.field-control label { font-size: 13px; font-weight: 750; }
.field-description { min-height: 34px; margin: 0; color: var(--druid-muted); font-size: 12px; line-height: 1.45; }
.field-input { width: 100%; min-height: 44px; padding: 10px 12px; border: 1px solid var(--druid-border); border-radius: 9px; background: #09130f; color: var(--druid-text); }
.boolean-control { min-height: 44px; display: flex; align-items: center; gap: 10px; padding: 8px 11px; border: 1px solid var(--druid-border); border-radius: 9px; background: #09130f; }
.boolean-control input { width: 20px; height: 20px; accent-color: var(--druid-accent); }
.field-meta { display: flex; flex-wrap: wrap; gap: 8px; color: var(--druid-muted); font-size: 11px; }
.restart-chip { color: var(--druid-warning); }
.field-error { margin: 0; color: var(--druid-error); font-size: 12px; }
.raw-editor { display: grid; gap: 10px; }
.raw-textarea { width: 100%; min-height: 440px; resize: vertical; padding: 16px; border: 1px solid var(--druid-border); border-radius: 12px; background: #050b08; color: #dce8e1; font: 13px/1.55 "Cascadia Code", "SFMono-Regular", Consolas, monospace; tab-size: 2; }
.inspector { display: flex; flex-direction: column; gap: 18px; padding: 18px; border-left: 1px solid var(--druid-border); }
.status-card { padding: 14px; border: 1px solid var(--druid-border); border-radius: 12px; background: var(--druid-panel); }
.status-card p { margin: 4px 0; color: var(--druid-muted); font-size: 13px; }
.change-list, .issue-list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
.change-item { padding: 10px; border: 1px solid var(--druid-border); border-radius: 9px; background: #0b1511; font-size: 13px; }
.change-value { display: block; margin-top: 4px; color: var(--druid-muted); word-break: break-word; }
.issue-item { color: var(--druid-error); font-size: 12px; }
.action-bar { position: sticky; bottom: 0; display: flex; align-items: center; justify-content: space-between; gap: 16px; margin: auto 0 0; padding-top: 14px; background: linear-gradient(transparent, rgba(11, 21, 17, .98) 32%); }
.save-button { min-height: 46px; padding: 0 18px; border: 0; border-radius: 9px; background: var(--druid-accent); color: var(--druid-accent-ink); font-weight: 850; cursor: pointer; }
.save-button:disabled { opacity: .45; cursor: not-allowed; }
.action-status { color: var(--druid-muted); font-size: 12px; }
.error-shell { margin: 20px; padding: 18px; border: 1px solid var(--druid-error); border-radius: 12px; background: #28120f; color: #ffd8d2; }
@media (max-width: 900px) {
  .editor-grid { grid-template-columns: 190px minmax(0, 1fr); }
  .inspector { grid-column: 1 / -1; border-left: 0; border-top: 1px solid var(--druid-border); }
}
@media (max-width: 560px) {
  .config-editor { min-height: 100%; }
  .editor-header { align-items: flex-start; padding: 14px 16px; }
  .editor-grid { display: block; }
  .file-rail { padding: 12px 16px; border-right: 0; border-bottom: 1px solid var(--druid-border); overflow-x: auto; }
  .file-list { display: flex; width: max-content; }
  .file-button { width: 190px; }
  .editor-main { padding: 16px 16px 28px; }
  .tabs { display: flex; }
  .tab { flex: 1; }
  .field-section { padding: 14px; }
  .field-grid { grid-template-columns: 1fr; }
  .raw-textarea { min-height: 360px; }
  .inspector { padding: 16px; }
  .action-bar { position: static; }
  .save-button { width: 100%; }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { scroll-behavior: auto !important; transition: none !important; animation: none !important; }
}
`;
