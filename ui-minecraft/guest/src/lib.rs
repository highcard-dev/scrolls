pub mod editor;
pub mod ini;
pub mod manifest;
pub mod properties;
#[cfg(target_arch = "wasm32")]
mod tools;

#[cfg(target_arch = "wasm32")]
wit_bindgen::generate!({
    path: env!("DRUID_WIT_PATH"),
    world: "openui-client",
});

#[cfg(target_arch = "wasm32")]
mod runtime {
    use super::druid::openui::state;
    use super::{
        editor::Editor,
        manifest::{self, Manifest},
        tools,
    };
    use serde_json::{json, Value};
    use std::{cell::RefCell, collections::BTreeMap};

    #[derive(Default)]
    struct Dashboard {
        manifest: Option<Manifest>,
        editors: BTreeMap<String, Editor>,
        selected: String,
        mode: String,
        busy: bool,
        status: String,
        error: Option<String>,
    }
    thread_local! { static DASHBOARD: RefCell<Dashboard> = RefCell::new(Dashboard::default()); }
    fn render() {
        let view = DASHBOARD.with_borrow(|dashboard| {
            let editor = dashboard.editors.get(&dashboard.selected);
            let mut view = editor.map(Editor::view).unwrap_or_else(||json!({}));
            view["manifest"] = json!(dashboard.manifest.as_ref().map(|m| &m.server).map(|server| json!({"server":server})).unwrap_or(Value::Null));
            view["selectedPath"] = json!(dashboard.selected);
            view["mode"] = json!(dashboard.mode);
            view["saving"] = json!(dashboard.busy);
            view["ready"] = json!(editor.is_some());
            view["status"] = json!(dashboard.status);
            view["error"] = json!(dashboard.error);
            view["files"] = json!(dashboard.manifest.as_ref().map(|m| m.files.iter().map(|f|json!({"value":f.path,"label":f.label})).collect::<Vec<_>>()).unwrap_or_default());
            view["tabs"] = json!([{"value":"form","label":"Form","disabled":editor.is_none_or(|e| !e.has_fields())},{"value":"raw","label":"Raw"}]);
            view["canSave"] = json!(editor.is_some_and(|e| e.document.source != e.original && e.errors().is_empty()) && !dashboard.busy);
            view["copy"] = json!({
                "loading":"Loading configuration…","formTab":"Form","rawTab":"Raw","modeHeading":"Editor mode",
                "filesHeading":"Configuration files","restartRequired":"Restart required","restartAfterSave":"Restart required after saving",
                "save":"Save changes","saving":"Saving…","rawLabel":"Raw configuration"
            });
            view
        });
        state::set_value("$editor", &view.to_string());
    }
    async fn select(path: &str) -> Result<(), String> {
        let file = DASHBOARD.with_borrow(|s| {
            s.manifest
                .as_ref()
                .and_then(|m| m.files.iter().find(|f| f.path == path))
                .cloned()
                .ok_or("Manifest has no selected file.".to_string())
        })?;
        let exists = DASHBOARD.with_borrow(|s| s.editors.contains_key(path));
        if !exists {
            let editor = tools::load(file).await?;
            DASHBOARD.with_borrow_mut(|s| {
                s.editors.insert(path.to_owned(), editor);
            });
        }
        DASHBOARD.with_borrow_mut(|s| {
            s.selected = path.to_owned();
            s.mode = if !s.editors[path].has_fields() {
                "raw"
            } else {
                "form"
            }
            .into();
        });
        Ok(())
    }
    pub async fn init() {
        DASHBOARD.with_borrow_mut(|s| *s = Dashboard::default());
        state::set_value("$access", "\"services\"");
        state::set_value("$section", "\"dashboard\"");
        render();
        if let Err(error) = initialize().await {
            DASHBOARD.with_borrow_mut(|s| s.error = Some(error));
        }
        render();
    }
    async fn initialize() -> Result<(), String> {
        tools::call("ensure_file_service", json!({"watch":false})).await?;
        let manifest = manifest::parse(&tools::text(".druid/config-editor.manifest.json").await?)?;
        let path = manifest.files[0].path.clone();
        DASHBOARD.with_borrow_mut(|s| s.manifest = Some(manifest));
        select(&path).await
    }
    pub async fn action(action: String, params: String) {
        let params: Value = match serde_json::from_str(&params) {
            Ok(value) => value,
            Err(_) => return,
        };
        if action == "SetAccessTab" || action == "SetSectionTab" {
            let (key, allowed): (&str, &[&str]) = if action == "SetAccessTab" {
                ("$access", &["services", "developer-tools"])
            } else {
                ("$section", &["dashboard", "configuration", "backups"])
            };
            if params["value"]
                .as_str()
                .is_some_and(|v| allowed.contains(&v))
            {
                state::set_value(key, &params["value"].to_string());
            }
            return;
        }
        let accepted = DASHBOARD.with_borrow_mut(|s| {
            if s.busy || s.editors.is_empty() {
                return false;
            }
            s.busy = true;
            s.status.clear();
            true
        });
        if !accepted {
            return;
        }
        render();
        let result = perform(&action, &params).await;
        DASHBOARD.with_borrow_mut(|s| {
            s.busy = false;
            if let Err(error) = result {
                s.status = error;
            }
        });
        render();
    }
    async fn perform(action: &str, params: &Value) -> Result<(), String> {
        let text = |key: &str| {
            params[key]
                .as_str()
                .ok_or_else(|| format!("Missing string parameter: {key}"))
        };
        match action {
            "SelectFile" => select(text("value")?).await?,
            "SetMode" => {
                let mode = text("value")?;
                if mode != "form" && mode != "raw" {
                    return Err("Invalid editor mode.".into());
                }
                DASHBOARD.with_borrow_mut(|s| s.mode = mode.into());
            }
            "SetField" | "SetRaw" => {
                DASHBOARD.with_borrow_mut(|s| {
                    let selected = s.selected.clone();
                    let editor = s
                        .editors
                        .get_mut(&selected)
                        .ok_or("No configuration file loaded.")?;
                    if action == "SetRaw" {
                        editor.set_raw(text("value")?.to_owned())
                    } else {
                        let input = match &params["value"] {
                            Value::String(value) => value.clone(),
                            Value::Bool(value) => value.to_string(),
                            Value::Number(value) => value.to_string(),
                            _ => return Err("Missing configuration field value.".into()),
                        };
                        editor.set_field(text("key")?, &input)
                    }
                })?;
            }
            "Save" => {
                let (path, editor) =
                    DASHBOARD.with_borrow(|s| (s.selected.clone(), s.editors[&s.selected].clone()));
                DASHBOARD.with_borrow_mut(|s| s.status = "Saving…".into());
                render();
                let remote = tools::save(&editor).await?;
                DASHBOARD.with_borrow_mut(|s| {
                    s.editors.get_mut(&path).unwrap().accept_saved(remote);
                    s.status = "Saved and verified".into();
                });
            }
            _ => {}
        }
        Ok(())
    }
}

#[cfg(target_arch = "wasm32")]
struct MinecraftDashboard;
#[cfg(target_arch = "wasm32")]
impl exports::druid::openui::client::Guest for MinecraftDashboard {
    async fn init() {
        runtime::init().await;
    }
    async fn handle_action(action: String, params: String) {
        runtime::action(action, params).await;
    }
}
#[cfg(target_arch = "wasm32")]
export!(MinecraftDashboard);
