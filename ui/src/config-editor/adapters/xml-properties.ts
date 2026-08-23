import type { ConfigValue, FileSchema, ValidationIssue } from "../model.js";
import { splitSourceLines } from "./line-document.js";
import type { ConfigAdapter, LineNode, ParsedDocument, SourceSpan } from "./types.js";

interface XmlProperty {
  key: string;
  value: string;
  valueSpan: SourceSpan;
  quote: '"' | "'";
}

export interface XmlPropertiesDocument extends ParsedDocument {
  format: "xml-properties";
  properties: readonly XmlProperty[];
}

const decodeXml = (value: string): string =>
  value.replace(/&(?:#x[\da-f]+|#\d+|amp|quot|apos|lt|gt);/gi, (entity) => {
    const token = entity.slice(1, -1);
    if (token.toLowerCase().startsWith("#x")) {
      return String.fromCodePoint(Number.parseInt(token.slice(2), 16));
    }
    if (token.startsWith("#")) {
      return String.fromCodePoint(Number.parseInt(token.slice(1), 10));
    }
    return { amp: "&", quot: '"', apos: "'", lt: "<", gt: ">" }[
      token.toLowerCase()
    ]!;
  });

const encodeXml = (value: ConfigValue, quote: '"' | "'"): string => {
  let encoded = (value === null ? "" : String(value))
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  encoded = quote === '"'
    ? encoded.replace(/"/g, "&quot;")
    : encoded.replace(/'/g, "&apos;");
  return encoded;
};

const parseXmlProperties = (source: string): XmlPropertiesDocument => {
  const properties: XmlProperty[] = [];
  const propertyPattern = /<property\b[^>]*>/gi;
  for (const match of source.matchAll(propertyPattern)) {
    const tag = match[0];
    const tagStart = match.index;
    const attributes = new Map<string, { value: string; span: SourceSpan; quote: '"' | "'" }>();
    const attributePattern = /\b(name|value)\s*=\s*(["'])(.*?)\2/gi;
    for (const attribute of tag.matchAll(attributePattern)) {
      const quote = attribute[2] as '"' | "'";
      const rawValue = attribute[3]!;
      const rawStart = tagStart + attribute.index + attribute[0].indexOf(quote) + 1;
      attributes.set(attribute[1]!.toLowerCase(), {
        value: decodeXml(rawValue),
        span: { start: rawStart, end: rawStart + rawValue.length },
        quote,
      });
    }
    const name = attributes.get("name");
    const value = attributes.get("value");
    if (name && value) {
      properties.push({
        key: name.value,
        value: value.value,
        valueSpan: value.span,
        quote: value.quote,
      });
    }
  }
  const nodes: LineNode[] = splitSourceLines(source).map((line) => ({
    ...line,
    kind: line.raw.trim() === "" ? "blank" : "invalid",
  }));
  return { format: "xml-properties", source, nodes, issues: [], properties };
};

const effectiveProperty = (
  document: XmlPropertiesDocument,
  key: string,
): XmlProperty | undefined => {
  for (let index = document.properties.length - 1; index >= 0; index -= 1) {
    const property = document.properties[index]!;
    if (property.key === key) return property;
  }
  return undefined;
};

export const xmlPropertiesAdapter: ConfigAdapter<XmlPropertiesDocument> = {
  parse: parseXmlProperties,
  entries(document) {
    const entries = new Map<string, ConfigValue>();
    for (const property of document.properties) entries.set(property.key, property.value);
    return [...entries].map(([key, value]) => ({ key, value }));
  },
  get(document, key) {
    return effectiveProperty(document, key)?.value;
  },
  getAll(document, key) {
    return document.properties
      .filter((property) => property.key === key)
      .map((property) => property.value);
  },
  getAllRaw(document, key) {
    return document.properties
      .filter((property) => property.key === key)
      .map((property) => document.source.slice(property.valueSpan.start, property.valueSpan.end));
  },
  set(document, key, value) {
    const property = effectiveProperty(document, key);
    if (!property) throw new Error(`XML property "${key}" does not exist.`);
    return parseXmlProperties(
      document.source.slice(0, property.valueSpan.start) +
        encodeXml(value, property.quote) +
        document.source.slice(property.valueSpan.end),
    );
  },
  setAll(document, key, values) {
    const properties = document.properties.filter((property) => property.key === key);
    if (properties.length !== values.length) {
      throw new Error(`Expected ${properties.length} values for XML property "${key}".`);
    }
    let source = document.source;
    for (let index = properties.length - 1; index >= 0; index -= 1) {
      const property = properties[index]!;
      source =
        source.slice(0, property.valueSpan.start) +
        encodeXml(values[index]!, property.quote) +
        source.slice(property.valueSpan.end);
    }
    return parseXmlProperties(source);
  },
  setAllRaw(document, key, values) {
    const properties = document.properties.filter((property) => property.key === key);
    if (properties.length !== values.length) {
      throw new Error(`Expected ${properties.length} raw values for XML property "${key}".`);
    }
    let source = document.source;
    for (let index = properties.length - 1; index >= 0; index -= 1) {
      const property = properties[index]!;
      source = source.slice(0, property.valueSpan.start) + values[index]! + source.slice(property.valueSpan.end);
    }
    return parseXmlProperties(source);
  },
  validate(document, schema: FileSchema): ValidationIssue[] {
    void schema;
    return [...document.issues];
  },
  serialize(document) {
    return document.source;
  },
};
