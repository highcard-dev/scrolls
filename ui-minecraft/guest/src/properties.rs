use std::ops::Range;

#[derive(Clone, Debug)]
pub struct Entry {
    pub key: String,
    pub value: String,
    pub span: Range<usize>,
    pub quote: Option<char>,
}
#[derive(Clone, Debug)]
pub struct Document {
    pub source: String,
    pub entries: Vec<Entry>,
    pub ini: bool,
}

fn whitespace(c: char) -> bool {
    matches!(c, ' ' | '\t' | '\x0c')
}

fn decode(input: &str) -> String {
    let mut chars = input.chars().peekable();
    let mut units = Vec::<u16>::new();
    while let Some(c) = chars.next() {
        let decoded = if c != '\\' {
            c
        } else {
            match chars.next() {
                Some('t') => '\t',
                Some('n') => '\n',
                Some('r') => '\r',
                Some('f') => '\x0c',
                Some('u') => {
                    while chars.peek() == Some(&'u') {
                        chars.next();
                    }
                    let mut probe = chars.clone();
                    let hex: String = probe.by_ref().take(4).collect();
                    if hex.len() == 4 && hex.chars().all(|ch| ch.is_ascii_hexdigit()) {
                        units.push(u16::from_str_radix(&hex, 16).unwrap());
                        chars = probe;
                        continue;
                    }
                    'u'
                }
                Some(other) => other,
                None => '\\',
            }
        };
        let mut buf = [0; 2];
        units.extend_from_slice(decoded.encode_utf16(&mut buf));
    }
    String::from_utf16_lossy(&units)
}

pub fn escape(value: &str, key: bool) -> String {
    let mut result = String::new();
    let mut leading = true;
    for c in value.chars() {
        match c {
            '\\' => result.push_str("\\\\"),
            '\n' => result.push_str("\\n"),
            '\r' => result.push_str("\\r"),
            '\t' => result.push_str("\\t"),
            '\x0c' => result.push_str("\\f"),
            ' ' if key || leading => result.push_str("\\ "),
            '=' | ':' | '#' | '!' if key => {
                result.push('\\');
                result.push(c);
            }
            _ => result.push(c),
        }
        if c != ' ' {
            leading = false;
        }
    }
    result
}

impl Document {
    pub fn parse(source: String) -> Self {
        // Physical spans retain original UTF-8 byte positions, including mixed line endings.
        let mut lines = Vec::new();
        let bytes = source.as_bytes();
        let mut start = 0;
        let mut i = 0;
        while i < bytes.len() {
            if bytes[i] == b'\r' || bytes[i] == b'\n' {
                let end = i;
                if bytes[i] == b'\r' && bytes.get(i + 1) == Some(&b'\n') {
                    i += 1;
                }
                i += 1;
                lines.push((start, end));
                start = i;
            } else {
                i += 1;
            }
        }
        if start < source.len() {
            lines.push((start, source.len()));
        }
        let mut entries = Vec::new();
        let mut index = 0;
        while index < lines.len() {
            let (start, stop) = lines[index];
            // Comments are physical lines, even when their text ends with a backslash.
            if source[start..stop]
                .trim_start_matches(whitespace)
                .starts_with(['#', '!'])
            {
                index += 1;
                continue;
            }
            let mut logical = Vec::<(char, usize)>::new();
            let mut continued = false;
            let end;
            loop {
                let (start, stop) = lines[index];
                let raw = &source[start..stop];
                let trim = if continued {
                    raw.len() - raw.trim_start_matches(whitespace).len()
                } else {
                    0
                };
                let content = &raw[trim..];
                let more = content.chars().rev().take_while(|c| *c == '\\').count() % 2 == 1;
                let kept = if more {
                    &content[..content.len() - 1]
                } else {
                    content
                };
                logical.extend(
                    kept.char_indices()
                        .map(|(offset, c)| (c, start + trim + offset)),
                );
                index += 1;
                if !more || index == lines.len() {
                    end = stop;
                    break;
                }
                continued = true;
            }
            let mut pos = 0;
            while pos < logical.len() && whitespace(logical[pos].0) {
                pos += 1;
            }
            if pos == logical.len() || matches!(logical[pos].0, '#' | '!') {
                continue;
            }
            let key_start = pos;
            let mut escaped = false;
            while pos < logical.len() {
                let c = logical[pos].0;
                if !escaped && (whitespace(c) || matches!(c, '=' | ':')) {
                    break;
                }
                escaped = !escaped && c == '\\';
                pos += 1;
            }
            let key: String = logical[key_start..pos].iter().map(|(c, _)| c).collect();
            while pos < logical.len() && whitespace(logical[pos].0) {
                pos += 1;
            }
            if pos < logical.len() && matches!(logical[pos].0, '=' | ':') {
                pos += 1;
            }
            while pos < logical.len() && whitespace(logical[pos].0) {
                pos += 1;
            }
            let value: String = logical[pos..].iter().map(|(c, _)| c).collect();
            entries.push(Entry {
                key: decode(&key),
                value: decode(&value),
                span: logical.get(pos).map(|p| p.1).unwrap_or(end)..end,
                quote: None,
            });
        }
        Self {
            source,
            entries,
            ini: false,
        }
    }

    pub fn get(&self, key: &str) -> Option<&str> {
        self.entries
            .iter()
            .rev()
            .find(|entry| entry.key == key)
            .map(|entry| entry.value.as_str())
    }

    pub fn set(&mut self, key: &str, value: &str) {
        if let Some(entry) = self.entries.iter().rev().find(|entry| entry.key == key) {
            let escaped = if self.ini {
                let value = value.replace('\r', "\\r").replace('\n', "\\n");
                match entry.quote {
                    Some('"') => value.replace('\\', "\\\\").replace('"', "\\\""),
                    Some('\'') => value.replace('\'', "'\\''"),
                    _ => value,
                }
            } else {
                escape(value, false)
            };
            self.source.replace_range(entry.span.clone(), &escaped);
        } else {
            let escaped = escape(value, false);
            let newline = if self.source.contains("\r\n") {
                "\r\n"
            } else if self.source.contains('\r') {
                "\r"
            } else {
                "\n"
            };
            let final_newline = self.source.ends_with(['\r', '\n']);
            if !self.source.is_empty() && !final_newline {
                self.source.push_str(newline);
            }
            self.source
                .push_str(&format!("{}={escaped}", escape(key, true)));
            if final_newline {
                self.source.push_str(newline);
            }
        }
        *self = Self::parse_format(std::mem::take(&mut self.source), self.ini);
    }

    pub fn replace_tokens(&self, replacements: Vec<(Range<usize>, String)>) -> Self {
        let mut source = self.source.clone();
        for (span, value) in replacements.into_iter().rev() {
            source.replace_range(span, &value);
        }
        Self::parse_format(source, self.ini)
    }

    pub fn parse_format(source: String, ini: bool) -> Self {
        if ini {
            crate::ini::parse(source)
        } else {
            Self::parse(source)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn lossless_and_targeted_change() {
        let source = "#Minecraft\r\nmax-players = 20\r\nunknown.future:key\\ value\r\n\r\n";
        let mut doc = Document::parse(source.into());
        assert_eq!(doc.source, source);
        doc.set("max-players", "50");
        assert_eq!(doc.source, source.replace("20", "50"));
    }
    #[test]
    fn properties_syntax_and_unicode() {
        for (source, key, expected) in [
            ("name=value\n", "name", "value"),
            ("name:value\n", "name", "value"),
            ("name   value\n", "name", "value"),
            ("name=\n", "name", ""),
            ("part\\=name=value\n", "part=name", "value"),
            ("server\\ name=Druid\n", "server name", "Druid"),
            ("motd=Grüße 世界\n", "motd", "Grüße 世界"),
            ("motd=Hello\\u0020World\n", "motd", "Hello World"),
            ("motd=\\uD83D\\uDE00\n", "motd", "😀"),
            (
                "motd=Hello \\\r\n  Druid\\ World\r\n",
                "motd",
                "Hello Druid World",
            ),
        ] {
            assert_eq!(Document::parse(source.into()).get(key), Some(expected));
        }
    }
    #[test]
    fn duplicates_and_appending() {
        let mut doc = Document::parse("motd=old\nmotd=new\n".into());
        assert_eq!(doc.get("motd"), Some("new"));
        doc.set("motd", "final");
        assert_eq!(doc.source, "motd=old\nmotd=final\n");
        for (source, expected) in [
            ("motd=Druid\r\n", "motd=Druid\r\nmax-players=20\r\n"),
            ("motd=Druid", "motd=Druid\nmax-players=20"),
        ] {
            let mut doc = Document::parse(source.into());
            doc.set("max-players", "20");
            assert_eq!(doc.source, expected);
        }
        let mut empty = Document::parse("".into());
        empty.set("server name", "Druid\\Realm\nLine");
        assert_eq!(empty.source, "server\\ name=Druid\\\\Realm\\nLine");
    }
    #[test]
    fn comments_do_not_consume_following_settings() {
        let source = "# comment\\\r\nmax-players=20\r\n! second\\\nmotd=Hello";
        let mut doc = Document::parse(source.into());
        assert_eq!(doc.get("max-players"), Some("20"));
        assert_eq!(doc.get("motd"), Some("Hello"));
        doc.set("max-players", "21");
        assert_eq!(doc.source, source.replace("20", "21"));
    }
}
