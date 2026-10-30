use crate::druid::openui::host;
use crate::{
    editor::{fingerprint, Editor},
    manifest::File,
};
use serde_json::{json, Value};

pub async fn call(operation: &str, params: Value) -> Result<Value, String> {
    let response = host::call(operation.to_owned(), params.to_string()).await?;
    serde_json::from_str(&response).map_err(|_| format!("Invalid response from {operation}."))
}
pub async fn text(path: &str) -> Result<String, String> {
    call("load_file", json!({"path":path}))
        .await?
        .as_str()
        .map(str::to_owned)
        .ok_or("File service returned non-text content.".into())
}
fn missing(error: &str) -> bool {
    let message = error.to_lowercase();
    message.contains("404")
        || message.contains("not found")
        || message.contains("not-found")
        || message.contains("missing")
}
pub async fn load(file: File) -> Result<Editor, String> {
    match text(&file.path).await {
        Ok(source) => {
            let path = file.path.clone();
            Ok(Editor::new(file, source, path))
        }
        Err(error) if missing(&error) => {
            for suffix in [".scroll_template", ".default"] {
                let path = format!("{}{suffix}", file.path);
                match text(&path).await {
                    Ok(source) => return Ok(Editor::new(file, source, path)),
                    Err(error) if missing(&error) => {}
                    Err(error) => return Err(error),
                }
            }
            Err(format!("Missing configuration file: {}", file.path))
        }
        Err(error) => Err(error),
    }
}
pub async fn save(editor: &Editor) -> Result<String, String> {
    if !editor.errors().is_empty() {
        return Err("Configuration has validation errors and cannot be saved.".into());
    }
    let source = &editor.document.source;
    call(
        "save_file",
        json!({"path":editor.schema.path,"content":source}),
    )
    .await?;
    let remote = text(&editor.schema.path).await?;
    if fingerprint(&remote) != fingerprint(source) || remote != *source {
        return Err(
            "Save verification failed: reloaded content differs from the saved content.".into(),
        );
    }
    Ok(remote)
}
