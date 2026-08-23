import type { ConfigFormat, ConfigValue, FileSchema, ValidationIssue } from "../model.js";
import { hasFinalNewline, preferredLineEnding, splitSourceLines } from "./line-document.js";
import type {
  ConfigAdapter,
  LineNode,
  ParsedDocument,
  SourceSpan,
} from "./types.js";

interface IniEntry {
  key: string;
  value: string;
  valueSpan: SourceSpan;
  line: number;
  quote?: '"' | "'";
  container?: boolean;
}

interface IniSection {
  name: string;
  line: number;
}

export interface IniDocument extends ParsedDocument {
  format: "ini" | "unreal-ini" | "key-value";
  entries: readonly IniEntry[];
  sections: readonly IniSection[];
  lineEnding: "\n" | "\r\n";
  finalNewline: boolean;
}

const inlineCommentAt = (value: string): number => {
  let quote: '"' | "'" | undefined;
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index]!;
    if (quote) {
      if (quote === "'" && value.startsWith("'\\''", index)) {
        index += 3;
        continue;
      }
      if (character === quote && value[index - 1] !== "\\") quote = undefined;
      continue;
    }
    if (character === '"' || character === "'") {
      quote = character;
      continue;
    }
    const precededBySpace = index > 0 && /\s/.test(value[index - 1]!);
    if (!precededBySpace) continue;
    if (character === ";" || character === "#") return index;
    if (character === "/" && value[index + 1] === "/") return index;
  }
  return value.length;
};

const decodeQuotedValue = (value: string, quote?: '"' | "'"): string => {
  if (quote === '"') return value.replace(/\\(["\\])/g, "$1");
  if (quote === "'") return value.replace(/'\\''/g, "'");
  return value;
};

const parseEntryLine = (
  raw: string,
  start: number,
  section: string | undefined,
  line: number,
  format: IniDocument["format"],
): IniEntry | undefined => {
  let keyStart = 0;
  while (keyStart < raw.length && /[ \t]/.test(raw[keyStart]!)) keyStart += 1;
  if (keyStart === raw.length) return undefined;

  let keyEnd = keyStart;
  while (
    keyEnd < raw.length &&
    raw[keyEnd] !== "=" &&
    raw[keyEnd] !== ":" &&
    !/[ \t]/.test(raw[keyEnd]!)
  ) {
    keyEnd += 1;
  }
  if (keyEnd === keyStart) return undefined;

  let valueStart = keyEnd;
  while (valueStart < raw.length && /[ \t]/.test(raw[valueStart]!)) valueStart += 1;
  if (raw[valueStart] === "=" || raw[valueStart] === ":") valueStart += 1;
  while (valueStart < raw.length && /[ \t]/.test(raw[valueStart]!)) valueStart += 1;

  const comment = inlineCommentAt(raw.slice(valueStart));
  let valueEnd = valueStart + comment;
  while (valueEnd > valueStart && /[ \t]/.test(raw[valueEnd - 1]!)) valueEnd -= 1;
  if (format === "key-value" && raw[valueEnd - 1] === ";") {
    valueEnd -= 1;
    while (valueEnd > valueStart && /[ \t]/.test(raw[valueEnd - 1]!)) valueEnd -= 1;
  }
  const quote = raw[valueStart];
  let valueQuote: '"' | "'" | undefined;
  if (
    (quote === '"' || quote === "'") &&
    valueEnd > valueStart &&
    raw[valueEnd - 1] === quote
  ) {
    valueQuote = quote;
    valueStart += 1;
    valueEnd -= 1;
  }
  const localKey = raw.slice(keyStart, keyEnd).trim();
  return {
    key: section ? `${section}.${localKey}` : localKey,
    value: decodeQuotedValue(raw.slice(valueStart, valueEnd), valueQuote),
    valueSpan: { start: start + valueStart, end: start + valueEnd },
    line,
    ...(valueQuote ? { quote: valueQuote } : {}),
  };
};

const tupleEntries = (entry: IniEntry): IniEntry[] => {
  if (!entry.value.startsWith("(") || !entry.value.endsWith(")")) return [];
  const inner = entry.value.slice(1, -1);
  const segments: Array<{ start: number; end: number }> = [];
  let segmentStart = 0;
  let depth = 0;
  let quote: '"' | "'" | undefined;
  for (let index = 0; index <= inner.length; index += 1) {
    const character = inner[index];
    if (quote) {
      if (character === quote && inner[index - 1] !== "\\") quote = undefined;
      continue;
    }
    if (character === '"' || character === "'") {
      quote = character;
      continue;
    }
    if (character === "(" || character === "[" || character === "{") depth += 1;
    else if (character === ")" || character === "]" || character === "}") depth -= 1;
    if ((character === "," && depth === 0) || index === inner.length) {
      segments.push({ start: segmentStart, end: index });
      segmentStart = index + 1;
    }
  }

  const result: IniEntry[] = [];
  for (const segment of segments) {
    const raw = inner.slice(segment.start, segment.end);
    let equals = -1;
    depth = 0;
    quote = undefined;
    for (let index = 0; index < raw.length; index += 1) {
      const character = raw[index]!;
      if (quote) {
        if (character === quote && raw[index - 1] !== "\\") quote = undefined;
        continue;
      }
      if (character === '"' || character === "'") quote = character;
      else if (character === "(" || character === "[" || character === "{") depth += 1;
      else if (character === ")" || character === "]" || character === "}") depth -= 1;
      else if (character === "=" && depth === 0) {
        equals = index;
        break;
      }
    }
    if (equals < 1) continue;
    const key = raw.slice(0, equals).trim();
    let localStart = equals + 1;
    while (localStart < raw.length && /\s/.test(raw[localStart]!)) localStart += 1;
    let localEnd = raw.length;
    while (localEnd > localStart && /\s/.test(raw[localEnd - 1]!)) localEnd -= 1;
    const valueQuote = raw[localStart];
    let preservedQuote: '"' | "'" | undefined;
    if (
      (valueQuote === '"' || valueQuote === "'") &&
      localEnd > localStart &&
      raw[localEnd - 1] === valueQuote
    ) {
      preservedQuote = valueQuote;
      localStart += 1;
      localEnd -= 1;
    }
    const absoluteStart = entry.valueSpan.start + 1 + segment.start + localStart;
    result.push({
      key: `${entry.key}.${key}`,
      value: decodeQuotedValue(raw.slice(localStart, localEnd), preservedQuote),
      valueSpan: {
        start: absoluteStart,
        end: absoluteStart + localEnd - localStart,
      },
      line: entry.line,
      ...(preservedQuote ? { quote: preservedQuote } : {}),
    });
  }
  return result;
};

const parseIni = (
  source: string,
  format: IniDocument["format"],
): IniDocument => {
  const lines = splitSourceLines(source);
  const entries: IniEntry[] = [];
  const sections: IniSection[] = [];
  const nodes: LineNode[] = [];
  let section: string | undefined;

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]!;
    const trimmed = line.raw.trimStart();
    if (trimmed === "") {
      nodes.push({ ...line, kind: "blank" });
      continue;
    }
    if (
      trimmed.startsWith(";") ||
      trimmed.startsWith("#") ||
      trimmed.startsWith("//")
    ) {
      nodes.push({ ...line, kind: "comment" });
      continue;
    }

    const sectionMatch = format === "key-value" ? null : trimmed.match(/^\[([^\]]+)](?:\s*(?:[;#].*)?)$/);
    if (sectionMatch) {
      section = sectionMatch[1]!.trim();
      sections.push({ name: section, line: index });
      nodes.push({ ...line, kind: "section" });
      continue;
    }

    const entry = parseEntryLine(line.raw, line.start, section, index, format);
    if (entry) {
      entries.push(entry);
      if (format === "unreal-ini") {
        const children = tupleEntries(entry);
        if (children.length > 0) {
          entry.container = true;
          entries.push(...children);
        }
      }
      nodes.push({
        ...line,
        kind: "entry",
        key: entry.key,
        value: entry.value,
        valueSpan: entry.valueSpan,
      });
    } else nodes.push({ ...line, kind: "invalid" });
  }

  return {
    format,
    source,
    nodes,
    entries,
    sections,
    issues: [],
    lineEnding: preferredLineEnding(lines),
    finalNewline: hasFinalNewline(source),
  };
};

const effectiveEntry = (document: IniDocument, key: string): IniEntry | undefined => {
  for (let index = document.entries.length - 1; index >= 0; index -= 1) {
    const entry = document.entries[index]!;
    if (entry.key === key) return entry;
  }
  return undefined;
};

const formatValue = (value: ConfigValue, quote?: '"' | "'"): string => {
  let formatted = (value === null ? "" : String(value)).replace(
    /[\r\n]/g,
    (character) => character === "\r" ? "\\r" : "\\n",
  );
  if (quote === '"') formatted = formatted.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  else if (quote === "'") formatted = formatted.replace(/'/g, "'\\''");
  return formatted;
};

const appendIniEntry = (
  document: IniDocument,
  key: string,
  value: ConfigValue,
): string => {
  const split = document.format === "key-value" ? -1 : key.lastIndexOf(".");
  const section = split < 0 ? undefined : key.slice(0, split);
  const localKey = split < 0 ? key : key.slice(split + 1);
  const assignment = `${localKey}=${formatValue(value)}`;
  const prefix = document.source === "" || document.finalNewline ? "" : document.lineEnding;
  const suffix = document.finalNewline ? document.lineEnding : "";

  if (!section || document.sections.some((candidate) => candidate.name === section)) {
    return `${document.source}${prefix}${assignment}${suffix}`;
  }
  const sectionHeader = `[${section}]${document.lineEnding}`;
  return `${document.source}${prefix}${sectionHeader}${assignment}${suffix}`;
};

const createIniAdapter = (
  format: IniDocument["format"],
): ConfigAdapter<IniDocument> => ({
  parse(source) {
    return parseIni(source, format);
  },
  entries(document) {
    const entries = new Map<string, ConfigValue>();
    for (const entry of document.entries) {
      if (!entry.container) entries.set(entry.key, entry.value);
    }
    return [...entries].map(([key, value]) => ({ key, value }));
  },
  get(document, key) {
    return effectiveEntry(document, key)?.value;
  },
  getAll(document, key) {
    return document.entries
      .filter((entry) => !entry.container && entry.key === key)
      .map((entry) => entry.value);
  },
  getAllRaw(document, key) {
    return document.entries
      .filter((entry) => !entry.container && entry.key === key)
      .map((entry) => document.source.slice(entry.valueSpan.start, entry.valueSpan.end));
  },
  set(document, key, value) {
    const entry = effectiveEntry(document, key);
    if (!entry) return parseIni(appendIniEntry(document, key, value), format);
    return parseIni(
      document.source.slice(0, entry.valueSpan.start) +
        formatValue(value, entry.quote) +
        document.source.slice(entry.valueSpan.end),
      format,
    );
  },
  setAll(document, key, values) {
    const entries = document.entries.filter(
      (entry) => !entry.container && entry.key === key,
    );
    if (entries.length !== values.length) {
      throw new Error(`Expected ${entries.length} values for setting "${key}".`);
    }
    let source = document.source;
    for (let index = entries.length - 1; index >= 0; index -= 1) {
      const entry = entries[index]!;
      source =
        source.slice(0, entry.valueSpan.start) +
        formatValue(values[index]!, entry.quote) +
        source.slice(entry.valueSpan.end);
    }
    return parseIni(source, format);
  },
  setAllRaw(document, key, values) {
    const entries = document.entries.filter(
      (entry) => !entry.container && entry.key === key,
    );
    if (entries.length !== values.length) {
      throw new Error(`Expected ${entries.length} raw values for setting "${key}".`);
    }
    let source = document.source;
    for (let index = entries.length - 1; index >= 0; index -= 1) {
      const entry = entries[index]!;
      source = source.slice(0, entry.valueSpan.start) + values[index]! + source.slice(entry.valueSpan.end);
    }
    return parseIni(source, format);
  },
  validate(document, schema: FileSchema): ValidationIssue[] {
    void schema;
    return [...document.issues];
  },
  serialize(document) {
    return document.source;
  },
});

export const iniAdapter = createIniAdapter("ini");
export const unrealIniAdapter = createIniAdapter("unreal-ini");
export const keyValueAdapter = createIniAdapter("key-value");

export const isIniFormat = (format: ConfigFormat): boolean =>
  format === "ini" || format === "unreal-ini" || format === "key-value";
