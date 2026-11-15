use crate::{
    manifest::{Field, File, Section},
    properties::{escape, Document},
};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};

pub const MASK: &str = "••••••••";
pub fn fingerprint(source: &str) -> String {
    format!("{:x}", Sha256::digest(source.as_bytes()))
}
pub fn sensitive(key: &str) -> bool {
    let normalized = regex_lite::Regex::new(r"([a-z\d])([A-Z])")
        .unwrap()
        .replace_all(key, "$1.$2")
        .to_lowercase();
    [
        "password",
        "passwd",
        "apikey",
        "api-key",
        "api_key",
        "api.key",
        "privatekey",
        "private-key",
        "gslt",
        "steampass",
    ]
    .iter()
    .any(|word| normalized.contains(word))
        || normalized
            .split(|c: char| !c.is_alphanumeric())
            .any(|part| matches!(part, "pass" | "secret" | "token"))
}
fn secret(field: &Field) -> bool {
    field.sensitive || field.kind == "secret" || sensitive(&field.key)
}
fn scalar(value: &Value) -> String {
    value.as_str().map(str::to_owned).unwrap_or_else(|| {
        if value.is_null() {
            String::new()
        } else {
            value.to_string()
        }
    })
}
fn placeholder(value: &str) -> bool {
    value.starts_with("__DRUID_")
        && value.ends_with("__")
        && value
            .chars()
            .all(|c| c.is_ascii_uppercase() || c.is_ascii_digit() || c == '_')
}
fn validate(field: &Field, value: Option<&str>) -> Result<String, String> {
    let Some(raw) = value
        .map(str::to_owned)
        .or_else(|| field.default_value.as_ref().map(scalar))
    else {
        return Ok(String::new());
    };
    let text = raw.trim();
    let normalized = match field.kind.as_str() {
        "boolean" => match text.to_lowercase().as_str() {
            "true" => "true".to_string(),
            "false" => "false".to_string(),
            _ => return Err("Value must be true or false.".into()),
        },
        "integer" | "number" => {
            if placeholder(&raw) {
                return Ok(String::new());
            }
            if field.kind == "integer"
                && !regex_lite::Regex::new(r"^[+-]?\d+(?:\.0+)?$")
                    .unwrap()
                    .is_match(text)
            {
                return Err("Value must be an integer.".into());
            }
            let number: f64 = text.parse().map_err(|_| "Value must be a finite number.")?;
            if !number.is_finite() {
                return Err("Value must be a finite number.".into());
            }
            if field.kind == "integer"
                && (number.fract() != 0.0 || number.abs() > 9007199254740991.0)
            {
                return Err("Value must be an integer.".into());
            }
            if field.min.is_some_and(|min| number < min)
                || field.max.is_some_and(|max| number > max)
            {
                return Err(match (field.min, field.max) {
                    (Some(min), Some(max)) => {
                        format!("Value must be between {min} and {max} (inclusive).")
                    }
                    (Some(min), _) => format!("Value must be at least {min}."),
                    (_, Some(max)) => format!("Value must be at most {max}."),
                    _ => unreachable!(),
                });
            }
            number.to_string()
        }
        "enum" if !field.values.contains(&raw) => {
            return Err(format!(
                "Value must be one of: {}.",
                field.values.join(", ")
            ))
        }
        _ => raw,
    };
    if let Some(pattern) = &field.pattern {
        if !regex_lite::Regex::new(&format!("^(?:{pattern})$"))
            .map_err(|_| "Invalid field pattern.")?
            .is_match(&normalized)
        {
            return Err("Value does not match the required format.".into());
        }
    }
    Ok(normalized)
}

#[derive(Clone)]
pub struct Editor {
    pub schema: File,
    pub original: String,
    pub document: Document,
    pub resolved_path: String,
    displays: BTreeMap<String, String>,
    errors: BTreeMap<String, String>,
}

impl Editor {
    pub fn new(schema: File, source: String, resolved_path: String) -> Self {
        let ini = schema.format == "ini";
        Self {
            schema,
            original: source.clone(),
            document: Document::parse_format(source, ini),
            resolved_path,
            displays: BTreeMap::new(),
            errors: BTreeMap::new(),
        }
    }
    fn sections(&self) -> Vec<Section> {
        let mut sections = self.schema.sections.clone();
        if self.schema.format == "raw" {
            return sections;
        }
        let mut seen: BTreeSet<String> = sections
            .iter()
            .flat_map(|s| s.fields.iter().map(|f| f.key.clone()))
            .collect();
        let mut fields = Vec::new();
        for entry in &self.document.entries {
            if !seen.insert(entry.key.clone()) {
                continue;
            }
            let secret = sensitive(&entry.key);
            let label = entry
                .key
                .split(['.', '/'])
                .next_back()
                .unwrap_or(&entry.key)
                .replace(['-', '_'], " ");
            let label = regex_lite::Regex::new(r"([a-z\d])([A-Z])")
                .unwrap()
                .replace_all(&label, "$1 $2")
                .to_string();
            let mut chars = label.chars();
            let label = chars
                .next()
                .map(|c| c.to_uppercase().to_string() + chars.as_str())
                .unwrap_or_default();
            let value = self.document.get(&entry.key).unwrap_or_default();
            fields.push(Field {
                key: entry.key.clone(),
                label,
                description: format!("Setting discovered in {}.", self.schema.label),
                kind: if secret {
                    "secret"
                } else if ["true", "false"].contains(&value.to_lowercase().as_str()) {
                    "boolean"
                } else {
                    "string"
                }
                .into(),
                sensitive: secret,
                restart_required: true,
                ..Field::default()
            });
        }
        if !fields.is_empty() {
            sections.push(Section {
                id: "detected-settings".into(),
                label: "Detected settings".into(),
                description: "Settings discovered from the loaded configuration file.".into(),
                fields,
            });
        }
        sections
    }
    pub fn has_fields(&self) -> bool {
        self.sections()
            .iter()
            .any(|section| !section.fields.is_empty())
    }
    pub fn set_field(&mut self, key: &str, input: &str) -> Result<(), String> {
        let field = self
            .sections()
            .into_iter()
            .flat_map(|s| s.fields)
            .find(|f| f.key == key)
            .ok_or("Unknown configuration field.")?;
        if secret(&field) && input == MASK {
            self.errors.remove(key);
            return Ok(());
        }
        self.displays.insert(key.into(), input.into());
        if input.trim().is_empty()
            && self.document.get(key).is_none()
            && matches!(field.kind.as_str(), "integer" | "number")
        {
            self.errors.remove(key);
            return Ok(());
        }
        match validate(&field, Some(input)) {
            Ok(value) => {
                self.document.set(key, &value);
                self.errors.remove(key);
                if secret(&field) {
                    self.displays.remove(key);
                }
            }
            Err(message) => {
                self.errors.insert(key.into(), message);
            }
        }
        Ok(())
    }
    fn secret_keys(&self) -> BTreeSet<String> {
        self.sections()
            .iter()
            .flat_map(|s| s.fields.iter().filter(|f| secret(f)).map(|f| f.key.clone()))
            .collect()
    }
    pub fn display_source(&self) -> String {
        let keys = self.secret_keys();
        self.document
            .replace_tokens(
                self.document
                    .entries
                    .iter()
                    .filter(|e| keys.contains(&e.key))
                    .map(|e| (e.span.clone(), escape(MASK, false)))
                    .collect(),
            )
            .source
    }
    pub fn set_raw(&mut self, source: String) -> Result<(), String> {
        let document = Document::parse_format(source, self.document.ini);
        let keys = self.secret_keys();
        let mut replacements = Vec::new();
        for key in &keys {
            let incoming: Vec<_> = document.entries.iter().filter(|e| &e.key == key).collect();
            if !incoming.iter().any(|e| e.value == MASK) {
                continue;
            }
            let originals: Vec<_> = self
                .document
                .entries
                .iter()
                .filter(|e| &e.key == key)
                .collect();
            if incoming.len() != originals.len() {
                return Err(format!(
                    "Secret occurrences for \"{key}\" cannot be added or removed in raw mode."
                ));
            }
            for (entry, original) in incoming.into_iter().zip(originals) {
                if entry.value == MASK {
                    replacements.push((
                        entry.span.clone(),
                        self.document.source[original.span.clone()].to_string(),
                    ));
                }
            }
        }
        replacements.sort_by_key(|(span, _)| span.start);
        self.document = document.replace_tokens(replacements);
        self.displays.clear();
        self.errors.clear();
        Ok(())
    }
    pub fn accept_saved(&mut self, source: String) {
        self.original = source.clone();
        self.document = Document::parse_format(source, self.document.ini);
        self.displays.clear();
        self.errors.clear();
        self.resolved_path = self.schema.path.clone();
    }
    pub fn errors(&self) -> Vec<String> {
        let mut issues: Vec<_> = self.errors.values().cloned().collect();
        for field in self.sections().iter().flat_map(|s| &s.fields) {
            if self.errors.contains_key(&field.key) {
                continue;
            }
            if let Err(error) = validate(field, self.document.get(&field.key)) {
                issues.push(error);
            }
        }
        issues
    }
    pub fn view(&self) -> Value {
        let sections: Vec<_> = self.sections().into_iter().map(|section| {
            let fields: Vec<_> = section.fields.iter().map(|field| {
                let result = validate(field, self.document.get(&field.key));
                let value = self.displays.get(&field.key).cloned().unwrap_or_else(|| result.clone().unwrap_or_else(|_| self.document.get(&field.key).unwrap_or_default().into()));
                let display = if secret(field) && !value.is_empty() { MASK.to_string() } else { value };
                let errors: Vec<_> = self.errors.get(&field.key).cloned().or_else(|| result.err()).into_iter().collect();
                json!({
                    "key":field.key,"label":field.label,"description":field.description,
                    "type":if secret(field) { "secret" } else { &field.kind },
                    "value":display,"checked":display == "true","errors":errors,
                    "min":field.min,"max":field.max,"step":if field.kind == "integer" { json!(1) } else { json!("any") },
                    "restartRequired":field.restart_required,
                    "options":field.values.iter().map(|value|json!({"value":value,"label":value})).collect::<Vec<_>>()
                })
            }).collect();
            json!({"id":section.id,"label":section.label,"description":section.description,"fields":fields})
        }).collect();
        let dirty = self.document.source != self.original;
        let original = Document::parse_format(self.original.clone(), self.document.ini);
        let fields: Vec<_> = self
            .sections()
            .into_iter()
            .flat_map(|section| section.fields)
            .collect();
        let mut structured = original.clone();
        let mut restart_required = false;
        for field in fields {
            let current = self.document.get(&field.key);
            if current != original.get(&field.key) {
                restart_required |= field.restart_required;
                if let Some(value) = current {
                    structured.set(&field.key, value);
                }
            }
        }
        restart_required |= dirty && structured.source != self.document.source;
        json!({
            "schema":self.schema,"source":self.display_source(),"sections":sections,
            "validation":self.errors().join("\n"),"dirty":dirty,
            "snapshot":{"dirty":dirty,"restartRequired":restart_required}
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn schema() -> File {
        serde_json::from_value(json!({"path":"server.properties","format":"java-properties","label":"server.properties","sections":[{"id":"general","label":"General","fields":[
            {"key":"max-players","label":"Max players","type":"integer","min":1,"max":1000},
            {"key":"rcon.password","label":"Password","type":"secret"},
            {"key":"online-mode","label":"Online mode","type":"boolean"},
            {"key":"difficulty","label":"Difficulty","type":"enum","values":["easy","hard"]}
        ]}]})).unwrap()
    }
    fn editor(source: &str) -> Editor {
        Editor::new(schema(), source.into(), "server.properties".into())
    }
    #[test]
    fn preserves_invalid_input_and_validates_raw() {
        let mut e = editor("# keep\nmax-players=20\ncustom=true\n");
        e.set_field("max-players", "not a number").unwrap();
        assert_eq!(e.document.source, e.original);
        assert!(!e.errors().is_empty());
        e.set_field("max-players", "30").unwrap();
        assert!(e.errors().is_empty());
        assert!(e.document.source.contains("max-players=30"));
        e.set_raw("max-players=0\n".into()).unwrap();
        assert!(!e.errors().is_empty());
    }
    #[test]
    fn masks_and_restores_duplicate_secret_tokens() {
        let source = "# comment\r\nrcon.password=one\\u0020secret\r\nrcon.password=two\\\\secret\r\nmax-players=20\r\n";
        let mut e = editor(source);
        let display = e.display_source();
        assert!(!display.contains("secret"));
        e.set_raw(display.replace("20", "25")).unwrap();
        assert_eq!(
            e.document.source,
            source.replace("max-players=20", "max-players=25")
        );
        assert!(!e.view().to_string().contains("two"));
        e.set_field("rcon.password", MASK).unwrap();
        assert!(e.document.source.contains("two\\\\secret"));
        assert!(e.set_raw("rcon.password=••••••••\n".into()).is_err());
    }
    #[test]
    fn detects_unknown_settings_and_secrets() {
        let mut e = editor("max-players=20\ncustom-setting=yes\napi-token=keep\nnew-bool=true\n");
        assert!(e.view().to_string().contains("Custom setting"));
        assert!(!e.view().to_string().contains("\"keep\""));
        e.set_field("custom-setting", "hello").unwrap();
        assert!(e.document.source.contains("custom-setting=hello"));
        assert!(sensitive("apiToken"));
        assert!(!sensitive("tokenizer"));
        assert!(!sensitive("secretary"));
    }
    #[test]
    fn cuberite_ini_keeps_form_edits_and_raw_secrets_lossless() {
        let file: File = serde_json::from_value(
            json!({"path":"settings.ini","format":"ini","label":"settings.ini","sections":[]}),
        )
        .unwrap();
        let source = "; keep\r\n[Server]\r\nMaxPlayers=20\r\nPassword='fixture' ; secret\r\n";
        let mut e = Editor::new(file, source.into(), "settings.ini".into());
        assert!(e.has_fields());
        assert!(!e.display_source().contains("fixture"));
        e.set_field("Server.MaxPlayers", "21").unwrap();
        e.set_raw(e.display_source()).unwrap();
        assert_eq!(e.document.source, source.replace("=20", "=21"));
    }
    #[test]
    fn restart_feedback_tracks_fields_and_unstructured_changes() {
        let mut e = editor("max-players=20\n");
        e.set_field("max-players", "21").unwrap();
        assert_eq!(e.view()["snapshot"]["restartRequired"], false);
        e.set_raw("# added comment\nmax-players=21\n".into())
            .unwrap();
        assert_eq!(e.view()["snapshot"]["restartRequired"], true);
        e.accept_saved(e.document.source.clone());
        assert_eq!(e.view()["snapshot"]["restartRequired"], false);
    }
    #[test]
    fn empty_numeric_and_runtime_placeholders_stay_absent() {
        let mut e = editor("motd=test\n");
        e.set_field("max-players", "").unwrap();
        assert_eq!(e.document.source, e.original);
        let e = editor("max-players=__DRUID_PLAYER_LIMIT__\n");
        assert!(e.errors().is_empty());
    }
    #[test]
    fn validates_scalar_types_enums_and_hashes() {
        let mut e = editor("max-players=20\nonline-mode=true\ndifficulty=easy\n");
        e.set_field("online-mode", "false").unwrap();
        e.set_field("difficulty", "wrong").unwrap();
        assert_eq!(e.document.get("online-mode"), Some("false"));
        assert!(!e.errors().is_empty());
        assert_eq!(
            fingerprint("abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }
}
