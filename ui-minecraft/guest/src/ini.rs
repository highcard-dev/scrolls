//! Cuberite's native INI files retain section names, quoting, comments and byte spans.
use crate::properties::{Document, Entry};

pub fn parse(source: String) -> Document {
    let mut entries = Vec::new();
    let mut section = String::new();
    let mut offset = 0;
    for line in source.split_inclusive(['\r', '\n']) {
        let raw = line.trim_end_matches(['\r', '\n']);
        let trimmed = raw.trim_start();
        if trimmed.starts_with('[') {
            if let Some(end) = trimmed.find(']') {
                section = trimmed[1..end].trim().into();
            }
        } else if !trimmed.is_empty()
            && !trimmed.starts_with([';', '#'])
            && !trimmed.starts_with("//")
        {
            let start = raw.len() - trimmed.len();
            let key_end = trimmed
                .find(|c: char| c == '=' || c == ':' || c.is_ascii_whitespace())
                .unwrap_or(trimmed.len())
                + start;
            let key = &raw[start..key_end];
            let mut value_start = key_end;
            while raw
                .as_bytes()
                .get(value_start)
                .is_some_and(|c| c.is_ascii_whitespace())
            {
                value_start += 1;
            }
            if raw
                .as_bytes()
                .get(value_start)
                .is_some_and(|c| *c == b'=' || *c == b':')
            {
                value_start += 1;
            }
            while raw
                .as_bytes()
                .get(value_start)
                .is_some_and(|c| c.is_ascii_whitespace())
            {
                value_start += 1;
            }
            let value = &raw[value_start..];
            let mut quote = None;
            let mut comment = value.len();
            let mut previous = None;
            for (i, c) in value.char_indices() {
                if let Some(q) = quote {
                    if c == q && previous != Some('\\') {
                        quote = None;
                    }
                } else if c == '"' || c == '\'' {
                    quote = Some(c);
                } else if previous.is_some_and(char::is_whitespace)
                    && (c == ';' || c == '#' || value[i..].starts_with("//"))
                {
                    comment = i;
                    break;
                }
                previous = Some(c);
            }
            let mut value_end = value_start + value[..comment].trim_end().len();
            let first = raw[value_start..value_end].chars().next();
            let quoted = first.filter(|c| {
                matches!(c, '"' | '\'')
                    && value_end > value_start + 1
                    && raw[..value_end].ends_with(*c)
            });
            if quoted.is_some() {
                value_start += 1;
                value_end -= 1;
            }
            let mut decoded = raw[value_start..value_end].to_owned();
            if quoted == Some('"') {
                decoded = decoded.replace("\\\"", "\"").replace("\\\\", "\\");
            }
            if quoted == Some('\'') {
                decoded = decoded.replace("'\\''", "'");
            }
            if !key.is_empty() {
                entries.push(Entry {
                    key: if section.is_empty() {
                        key.into()
                    } else {
                        format!("{section}.{key}")
                    },
                    value: decoded,
                    span: offset + value_start..offset + value_end,
                    quote: quoted,
                });
            }
        }
        offset += line.len();
    }
    Document {
        source,
        entries,
        ini: true,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn preserves_cuberite_sections_quotes_comments_and_duplicates() {
        let source = "; comment\r\n[Server]\r\nMaxPlayers=20 ; keep\r\nDescription=\"Druid\"\r\n[WebAdmin]\r\nPassword='secret'\r\nPassword='active'\r\n";
        let mut doc = parse(source.into());
        assert_eq!(doc.get("Server.MaxPlayers"), Some("20"));
        assert_eq!(doc.get("WebAdmin.Password"), Some("active"));
        doc.set("Server.MaxPlayers", "21");
        doc.set("Server.Description", "My \"server\"");
        assert_eq!(
            doc.source,
            source
                .replace("=20", "=21")
                .replace("\"Druid\"", "\"My \\\"server\\\"\"")
        );
        assert_eq!(doc.get("Server.Description"), Some("My \"server\""));
    }
}
