import type { ConfigValue, FileSchema, ValidationIssue } from "../model.js";
import { splitSourceLines } from "./line-document.js";
import type { ConfigAdapter, LineNode, ParsedDocument } from "./types.js";

export interface RawDocument extends ParsedDocument {
  format: "raw";
}

const parseRaw = (source: string): RawDocument => {
  const nodes: LineNode[] = splitSourceLines(source).map((line) => ({
    ...line,
    kind: line.raw.trim() === "" ? "blank" : "invalid",
  }));
  return { format: "raw", source, nodes, issues: [] };
};

export const rawAdapter: ConfigAdapter<RawDocument> = {
  parse: parseRaw,
  entries() {
    return [];
  },
  get() {
    return undefined;
  },
  getAll() {
    return [];
  },
  getAllRaw() {
    return [];
  },
  set(_document, _key, _value: ConfigValue) {
    throw new Error("Raw mode does not support typed field writes.");
  },
  setAll(document, _key, values) {
    if (values.length > 0) throw new Error("Raw mode does not support typed field writes.");
    return document;
  },
  setAllRaw(document, _key, values) {
    if (values.length > 0) throw new Error("Raw mode does not support typed field writes.");
    return document;
  },
  validate(document, schema: FileSchema): ValidationIssue[] {
    void schema;
    return [...document.issues];
  },
  serialize(document) {
    return document.source;
  },
};
