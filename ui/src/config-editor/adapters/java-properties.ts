import type { ConfigValue, FileSchema, ValidationIssue } from "../model.js";
import { hasFinalNewline, preferredLineEnding, splitSourceLines } from "./line-document.js";
import type {
  ConfigAdapter,
  LineNode,
  ParsedDocument,
  SourceLine,
  SourceSpan,
} from "./types.js";

interface JavaPropertiesEntry {
  key: string;
  value: string;
  valueSpan: SourceSpan;
  firstLine: number;
  lastLine: number;
}

export interface JavaPropertiesDocument extends ParsedDocument {
  format: "java-properties";
  entries: readonly JavaPropertiesEntry[];
  lineEnding: "\n" | "\r\n";
  finalNewline: boolean;
}

interface LogicalLine {
  text: string;
  offsets: number[];
  end: number;
  firstLine: number;
  lastLine: number;
}

const trailingBackslashes = (value: string): number => {
  let count = 0;
  for (let index = value.length - 1; index >= 0 && value[index] === "\\"; index -= 1) {
    count += 1;
  }
  return count;
};

const logicalLineAt = (
  lines: readonly SourceLine[],
  firstLine: number,
): LogicalLine => {
  let text = "";
  const offsets: number[] = [];
  let lineIndex = firstLine;
  let continued = false;

  for (; lineIndex < lines.length; lineIndex += 1) {
    const line = lines[lineIndex]!;
    const trim = continued ? line.raw.match(/^[ \t\f]*/)?.[0].length ?? 0 : 0;
    const content = line.raw.slice(trim);
    const continues = trailingBackslashes(content) % 2 === 1;
    const logicalContent = continues ? content.slice(0, -1) : content;

    for (let index = 0; index < logicalContent.length; index += 1) {
      text += logicalContent[index]!;
      offsets.push(line.start + trim + index);
    }

    continued = continues;
    if (!continues) break;
  }

  const lastLine = Math.min(lineIndex, lines.length - 1);
  return {
    text,
    offsets,
    end: lines[lastLine]?.end ?? 0,
    firstLine,
    lastLine,
  };
};

const isEscaped = (value: string, index: number): boolean => {
  let slashes = 0;
  for (let cursor = index - 1; cursor >= 0 && value[cursor] === "\\"; cursor -= 1) {
    slashes += 1;
  }
  return slashes % 2 === 1;
};

const decodeEscapes = (value: string): string => {
  let decoded = "";
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index]!;
    if (character !== "\\" || index === value.length - 1) {
      decoded += character;
      continue;
    }

    const escaped = value[(index += 1)]!;
    if (escaped === "t") decoded += "\t";
    else if (escaped === "n") decoded += "\n";
    else if (escaped === "r") decoded += "\r";
    else if (escaped === "f") decoded += "\f";
    else if (escaped === "u") {
      while (value[index + 1] === "u") index += 1;
      const hexadecimal = value.slice(index + 1, index + 5);
      if (/^[\da-f]{4}$/i.test(hexadecimal)) {
        decoded += String.fromCharCode(Number.parseInt(hexadecimal, 16));
        index += 4;
      } else {
        decoded += "u";
      }
    } else decoded += escaped;
  }
  return decoded;
};

const parseEntry = (logical: LogicalLine): JavaPropertiesEntry | undefined => {
  const { text } = logical;
  let start = 0;
  while (start < text.length && /[ \t\f]/.test(text[start]!)) start += 1;
  if (start === text.length || text[start] === "#" || text[start] === "!") {
    return undefined;
  }

  let keyEnd = start;
  while (keyEnd < text.length) {
    const character = text[keyEnd]!;
    if (!isEscaped(text, keyEnd) && (character === "=" || character === ":" || /[ \t\f]/.test(character))) {
      break;
    }
    keyEnd += 1;
  }

  let valueStart = keyEnd;
  while (valueStart < text.length && /[ \t\f]/.test(text[valueStart]!)) {
    valueStart += 1;
  }
  if (valueStart < text.length && (text[valueStart] === "=" || text[valueStart] === ":")) {
    valueStart += 1;
  }
  while (valueStart < text.length && /[ \t\f]/.test(text[valueStart]!)) {
    valueStart += 1;
  }

  return {
    key: decodeEscapes(text.slice(start, keyEnd)),
    value: decodeEscapes(text.slice(valueStart)),
    valueSpan: {
      start: logical.offsets[valueStart] ?? logical.end,
      end: logical.end,
    },
    firstLine: logical.firstLine,
    lastLine: logical.lastLine,
  };
};

const classifyNodes = (
  lines: readonly SourceLine[],
  entries: readonly JavaPropertiesEntry[],
): LineNode[] => {
  const firstLines = new Map(entries.map((entry) => [entry.firstLine, entry]));
  const continuationLines = new Set<number>();
  for (const entry of entries) {
    for (let index = entry.firstLine + 1; index <= entry.lastLine; index += 1) {
      continuationLines.add(index);
    }
  }

  return lines.map((line, index) => {
    const entry = firstLines.get(index);
    if (entry) {
      return {
        ...line,
        kind: "entry",
        key: entry.key,
        value: entry.value,
        valueSpan: entry.valueSpan,
      };
    }
    if (continuationLines.has(index)) return { ...line, kind: "continuation" };
    const trimmed = line.raw.trimStart();
    if (trimmed === "") return { ...line, kind: "blank" };
    if (trimmed.startsWith("#") || trimmed.startsWith("!")) {
      return { ...line, kind: "comment" };
    }
    return { ...line, kind: "invalid" };
  });
};

const parseJavaProperties = (source: string): JavaPropertiesDocument => {
  const lines = splitSourceLines(source);
  const entries: JavaPropertiesEntry[] = [];

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
    const logical = logicalLineAt(lines, lineIndex);
    const entry = parseEntry(logical);
    if (entry) entries.push(entry);
    lineIndex = logical.lastLine;
  }

  return {
    format: "java-properties",
    source,
    nodes: classifyNodes(lines, entries),
    entries,
    issues: [],
    lineEnding: preferredLineEnding(lines),
    finalNewline: hasFinalNewline(source),
  };
};

const escapeKey = (value: string): string =>
  value.replace(/[\\ \t\f=:#!]/g, (character) => {
    if (character === "\t") return "\\t";
    if (character === "\f") return "\\f";
    return `\\${character}`;
  });

const escapeValue = (value: ConfigValue): string => {
  const text = value === null ? "" : String(value);
  return text.replace(/[\\\t\n\r\f]/g, (character) => {
    if (character === "\\") return "\\\\";
    if (character === "\t") return "\\t";
    if (character === "\n") return "\\n";
    if (character === "\r") return "\\r";
    return "\\f";
  }).replace(/^ +/, (spaces) => "\\ ".repeat(spaces.length));
};

const effectiveEntry = (
  document: JavaPropertiesDocument,
  key: string,
): JavaPropertiesEntry | undefined => {
  for (let index = document.entries.length - 1; index >= 0; index -= 1) {
    const entry = document.entries[index]!;
    if (entry.key === key) return entry;
  }
  return undefined;
};

const appendEntry = (
  document: JavaPropertiesDocument,
  key: string,
  value: ConfigValue,
): string => {
  const entry = `${escapeKey(key)}=${escapeValue(value)}`;
  if (document.source === "") return entry;
  if (document.finalNewline) return `${document.source}${entry}${document.lineEnding}`;
  return `${document.source}${document.lineEnding}${entry}`;
};

export const javaPropertiesAdapter: ConfigAdapter<JavaPropertiesDocument> = {
  parse: parseJavaProperties,
  entries(document) {
    const entries = new Map<string, ConfigValue>();
    for (const entry of document.entries) entries.set(entry.key, entry.value);
    return [...entries].map(([key, value]) => ({ key, value }));
  },
  get(document, key) {
    return effectiveEntry(document, key)?.value;
  },
  getAll(document, key) {
    return document.entries
      .filter((entry) => entry.key === key)
      .map((entry) => entry.value);
  },
  getAllRaw(document, key) {
    return document.entries
      .filter((entry) => entry.key === key)
      .map((entry) => document.source.slice(entry.valueSpan.start, entry.valueSpan.end));
  },
  set(document, key, value) {
    const entry = effectiveEntry(document, key);
    if (!entry) return parseJavaProperties(appendEntry(document, key, value));
    const source =
      document.source.slice(0, entry.valueSpan.start) +
      escapeValue(value) +
      document.source.slice(entry.valueSpan.end);
    return parseJavaProperties(source);
  },
  setAll(document, key, values) {
    const entries = document.entries.filter((entry) => entry.key === key);
    if (entries.length !== values.length) {
      throw new Error(`Expected ${entries.length} values for Java property "${key}".`);
    }
    let source = document.source;
    for (let index = entries.length - 1; index >= 0; index -= 1) {
      const entry = entries[index]!;
      source =
        source.slice(0, entry.valueSpan.start) +
        escapeValue(values[index]!) +
        source.slice(entry.valueSpan.end);
    }
    return parseJavaProperties(source);
  },
  setAllRaw(document, key, values) {
    const entries = document.entries.filter((entry) => entry.key === key);
    if (entries.length !== values.length) {
      throw new Error(`Expected ${entries.length} raw values for Java property "${key}".`);
    }
    let source = document.source;
    for (let index = entries.length - 1; index >= 0; index -= 1) {
      const entry = entries[index]!;
      source = source.slice(0, entry.valueSpan.start) + values[index]! + source.slice(entry.valueSpan.end);
    }
    return parseJavaProperties(source);
  },
  validate(document, schema: FileSchema): ValidationIssue[] {
    void schema;
    return [...document.issues];
  },
  serialize(document) {
    return document.source;
  },
};
