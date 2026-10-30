use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeSet;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Manifest {
    pub version: u32,
    pub server: Server,
    pub files: Vec<File>,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Server {
    pub family: String,
    pub display_name: String,
    #[serde(default)]
    pub app_version: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct File {
    pub path: String,
    pub format: String,
    pub label: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub documentation: String,
    pub sections: Vec<Section>,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct Section {
    pub id: String,
    pub label: String,
    #[serde(default)]
    pub description: String,
    pub fields: Vec<Field>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Field {
    pub key: String,
    pub label: String,
    #[serde(default)]
    pub description: String,
    #[serde(rename = "type")]
    pub kind: String,
    #[serde(default)]
    pub documentation: String,
    #[serde(default)]
    pub values: Vec<String>,
    pub min: Option<f64>,
    pub max: Option<f64>,
    pub pattern: Option<String>,
    #[serde(default)]
    pub sensitive: bool,
    #[serde(default)]
    pub restart_required: bool,
    pub default_value: Option<Value>,
    pub since: Option<String>,
    pub until: Option<String>,
}
pub fn safe_path(path: &str) -> bool {
    !path.is_empty()
        && !path.starts_with('/')
        && !path.contains(['\\', ':', '\0'])
        && path
            .split('/')
            .all(|segment| !segment.is_empty() && segment != "." && segment != "..")
}
fn version(input: &str) -> Result<Vec<u32>, String> {
    input
        .split('.')
        .map(|part| {
            part.parse()
                .map_err(|_| "Invalid manifest version range.".into())
        })
        .collect()
}
fn compare(left: &str, right: &str) -> Result<std::cmp::Ordering, String> {
    let (mut l, mut r) = (version(left)?, version(right)?);
    let len = l.len().max(r.len());
    l.resize(len, 0);
    r.resize(len, 0);
    Ok(l.cmp(&r))
}
pub fn parse(source: &str) -> Result<Manifest, String> {
    let mut manifest: Manifest =
        serde_json::from_str(source).map_err(|_| "Invalid configuration manifest.".to_string())?;
    if manifest.version != 1
        || !matches!(
            manifest.server.family.as_str(),
            "minecraft" | "minecraft-cuberite"
        )
        || manifest.server.display_name.trim().is_empty()
        || manifest.files.is_empty()
    {
        return Err("Unsupported Minecraft configuration manifest.".into());
    }
    let mut paths = BTreeSet::new();
    for file in &mut manifest.files {
        if !safe_path(&file.path)
            || !paths.insert(file.path.clone())
            || file.label.trim().is_empty()
            || !matches!(file.format.as_str(), "java-properties" | "ini" | "raw")
        {
            return Err("Invalid or duplicate configuration file.".into());
        }
        let mut ids = BTreeSet::new();
        let mut keys = BTreeSet::new();
        for section in &mut file.sections {
            if section.id.is_empty() || section.label.is_empty() || !ids.insert(section.id.clone())
            {
                return Err("Invalid or duplicate section.".into());
            }
            for field in &section.fields {
                if field.key.is_empty()
                    || field.label.is_empty()
                    || !keys.insert(field.key.clone())
                    || !matches!(
                        field.kind.as_str(),
                        "string" | "secret" | "integer" | "number" | "boolean" | "enum"
                    )
                    || matches!((field.min, field.max), (Some(min), Some(max)) if min > max)
                    || (field.kind == "enum" && field.values.is_empty())
                    || field
                        .default_value
                        .as_ref()
                        .is_some_and(|v| v.is_array() || v.is_object())
                {
                    return Err("Invalid or duplicate configuration field.".into());
                }
                if let Some(pattern) = &field.pattern {
                    regex_lite::Regex::new(&format!("^(?:{pattern})$"))
                        .map_err(|_| "Invalid field pattern.")?;
                }
                if let Some(since) = &field.since {
                    version(since)?;
                }
                if let Some(until) = &field.until {
                    version(until)?;
                }
                if let (Some(since), Some(until)) = (&field.since, &field.until) {
                    if compare(since, until)?.is_gt() {
                        return Err("Invalid field version range.".into());
                    }
                }
            }
            let mut filtered = Vec::new();
            for field in section.fields.drain(..) {
                let include = (field.since.is_none()
                    || compare(&manifest.server.app_version, field.since.as_ref().unwrap())?
                        .is_ge())
                    && (field.until.is_none()
                        || compare(&manifest.server.app_version, field.until.as_ref().unwrap())?
                            .is_le());
                if include {
                    filtered.push(field);
                }
            }
            section.fields = filtered;
        }
        file.sections.retain(|section| !section.fields.is_empty());
    }
    Ok(manifest)
}
#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    pub fn source() -> Value {
        json!({"version":1,"server":{"family":"minecraft","displayName":"Minecraft Server","appVersion":"1.21.7"},"files":[{"path":"server.properties","format":"java-properties","label":"server.properties","sections":[{"id":"players","label":"Players","fields":[{"key":"max-players","label":"Max players","type":"integer","min":1,"max":1000}]}]}]})
    }
    #[test]
    fn validates_paths_and_duplicates() {
        assert!(parse(&source().to_string()).is_ok());
        for path in [
            "../secret",
            "/absolute",
            "C:/file",
            "data/../secret",
            "a\\b",
        ] {
            let mut value = source();
            value["files"][0]["path"] = json!(path);
            assert!(parse(&value.to_string()).is_err());
        }
        let mut value = source();
        let duplicate = value["files"][0].clone();
        value["files"].as_array_mut().unwrap().push(duplicate);
        assert!(parse(&value.to_string()).is_err());
    }
    #[test]
    fn filters_versions() {
        let mut value = source();
        value["files"][0]["sections"][0]["fields"][0]["until"] = json!("1.20.4");
        assert!(parse(&value.to_string()).unwrap().files[0]
            .sections
            .is_empty());
    }
}
