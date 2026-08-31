export const EDITOR_STYLES = `
:host {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
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
.druid-wrapper { width: 100%; height: 100%; min-height: 0; overflow: hidden; }
* { box-sizing: border-box; }
button, input, select, textarea { font: inherit; }
button, input, select, textarea, a { outline: none; }
button:focus-visible, input:focus-visible, select:focus-visible, textarea:focus-visible, a:focus-visible {
  box-shadow: 0 0 0 3px rgba(168, 239, 156, .28);
  border-color: var(--druid-accent);
}
.config-editor { display: grid; grid-template-rows: auto minmax(0, 1fr); width: 100%; height: 100%; min-height: 0; overflow: hidden; background: radial-gradient(circle at 75% 0%, #173326 0, transparent 32%), var(--druid-bg); color: var(--druid-text); }
.editor-header { min-height: 76px; display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 16px 22px; border-bottom: 1px solid var(--druid-border); }
.eyebrow { color: var(--druid-accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; text-transform: uppercase; }
.editor-title { margin: 3px 0 0; font-size: clamp(20px, 2.4vw, 30px); line-height: 1.1; }
.header-controls { display: flex; align-items: center; justify-content: flex-end; gap: 18px; }
.server-version { color: var(--druid-muted); font-size: 13px; }
.editor-grid { display: grid; grid-template-columns: minmax(210px, 250px) minmax(0, 1fr); width: 100%; height: 100%; min-height: 0; overflow: hidden; }
.file-rail { min-height: 0; overflow-y: auto; padding: 18px 14px; border-right: 1px solid var(--druid-border); background: rgba(11, 21, 17, .8); }
.rail-heading { margin: 0 0 12px; font-size: 12px; color: var(--druid-muted); letter-spacing: .08em; text-transform: uppercase; }
.file-list { display: grid; gap: 8px; }
.file-button { width: 100%; min-height: 52px; display: grid; gap: 3px; padding: 10px 12px; text-align: left; border: 1px solid transparent; border-radius: 10px; background: transparent; color: var(--druid-text); cursor: pointer; }
.file-button:hover { background: var(--druid-panel); }
.file-button[aria-current="true"] { border-color: var(--druid-border-strong); background: var(--druid-panel-raised); }
.file-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 700; }
.file-format { color: var(--druid-muted); font-size: 12px; text-transform: uppercase; }
.editor-main { min-width: 0; min-height: 0; overflow-y: auto; padding: 18px 22px 28px; }
:is(.editor-main, .file-rail, .raw-textarea) {
  scrollbar-width: thin;
  scrollbar-color: var(--druid-border-strong) transparent;
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar-track {
  background: transparent;
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar-thumb {
  min-height: 40px;
  border: 3px solid transparent;
  border-radius: 999px;
  background-color: var(--druid-border-strong);
  background-clip: content-box;
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar-thumb:hover {
  background-color: var(--druid-muted);
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar-button {
  display: none;
  width: 0;
  height: 0;
}
:is(.editor-main, .file-rail, .raw-textarea)::-webkit-scrollbar-corner {
  background: transparent;
}
.tabs { display: inline-flex; gap: 4px; padding: 4px; margin-bottom: 18px; border: 1px solid var(--druid-border); border-radius: 10px; background: #0b1511; }
.tab { min-height: 40px; min-width: 92px; border: 0; border-radius: 7px; background: transparent; color: var(--druid-muted); cursor: pointer; }
.tab[aria-selected="true"] { background: var(--druid-panel-raised); color: var(--druid-text); }
.form-editor { display: grid; gap: 18px; }
.field-section { min-width: 0; margin: 0; padding: 18px; border: 1px solid var(--druid-border); border-radius: 14px; background: rgba(16, 27, 23, .84); content-visibility: auto; contain-intrinsic-size: auto 360px; }
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
.field-meta > span:last-child { min-width: 0; max-width: 100%; overflow-wrap: anywhere; word-break: break-word; }
.restart-chip { color: var(--druid-warning); }
.field-error { margin: 0; color: var(--druid-error); font-size: 12px; }
.raw-editor { display: grid; gap: 10px; }
.raw-textarea { width: 100%; min-height: 440px; resize: vertical; padding: 16px; border: 1px solid var(--druid-border); border-radius: 12px; background: #050b08; color: #dce8e1; font: 13px/1.55 "Cascadia Code", "SFMono-Regular", Consolas, monospace; tab-size: 2; }
.status-card { padding: 14px; border: 1px solid var(--druid-border); border-radius: 12px; background: var(--druid-panel); }
.action-bar { display: flex; align-items: center; justify-content: flex-end; gap: 14px; }
.action-feedback { display: grid; gap: 3px; text-align: right; }
.save-button { min-height: 46px; padding: 0 18px; border: 0; border-radius: 9px; background: var(--druid-accent); color: var(--druid-accent-ink); font-weight: 850; cursor: pointer; }
.save-button:disabled { opacity: .45; cursor: not-allowed; }
.action-status { color: var(--druid-muted); font-size: 12px; }
.restart-notice { color: var(--druid-warning); font-size: 12px; }
.error-shell { margin: 20px; padding: 18px; border: 1px solid var(--druid-error); border-radius: 12px; background: #28120f; color: #ffd8d2; }
@media (max-width: 900px) {
  .editor-grid { grid-template-columns: 190px minmax(0, 1fr); }
  .header-controls { gap: 12px; }
}
@media (max-width: 560px) {
  .editor-header { align-items: flex-start; flex-wrap: wrap; gap: 12px; padding: 14px 16px; }
  .header-controls { width: 100%; justify-content: space-between; }
  .editor-grid { grid-template-columns: 1fr; grid-template-rows: auto minmax(0, 1fr); }
  .file-rail { overflow-x: auto; overflow-y: hidden; padding: 12px 16px; border-right: 0; border-bottom: 1px solid var(--druid-border); }
  .file-list { display: flex; width: max-content; }
  .file-button { width: 190px; }
  .editor-main { padding: 16px 16px 28px; }
  .tabs { display: flex; }
  .tab { flex: 1; }
  .field-section { padding: 14px; }
  .field-grid { grid-template-columns: 1fr; }
  .raw-textarea { min-height: 360px; }
  .action-bar { flex: 1; }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { scroll-behavior: auto !important; transition: none !important; animation: none !important; }
}
`;
