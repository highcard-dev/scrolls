import type { ConfigValue, FileSchema, ValidationIssue } from "../model.js";
import { splitSourceLines } from "./line-document.js";
import type { ConfigAdapter, LineNode, ParsedDocument, SourceSpan } from "./types.js";

interface JsonScalar {
  value: ConfigValue;
  span: SourceSpan;
  unsafeInteger?: boolean;
}

interface JsonContainer {
  kind: "object" | "array";
  open: number;
  close: number;
}

export interface JsonDocument extends ParsedDocument {
  format: "json";
  scalars: ReadonlyMap<string, JsonScalar>;
  scalarOccurrences: ReadonlyMap<string, readonly JsonScalar[]>;
  containers: ReadonlyMap<string, JsonContainer>;
}

class JsonParseError extends Error {
  readonly position: number;

  constructor(position: number, message: string) {
    super(message);
    this.position = position;
  }
}

const pointerSegment = (value: string): string =>
  value.replace(/~/g, "~0").replace(/\//g, "~1");

const locationAt = (source: string, position: number): { line: number; column: number } => {
  const before = source.slice(0, position);
  const lines = before.split("\n");
  return { line: lines.length, column: lines.at(-1)!.length + 1 };
};

const jsonNodes = (source: string): LineNode[] =>
  splitSourceLines(source).map((line) => ({
    ...line,
    kind: line.raw.trim() === "" ? "blank" : "invalid",
  }));

const parseJson = (source: string): JsonDocument => {
  const scalars = new Map<string, JsonScalar>();
  const scalarOccurrences = new Map<string, JsonScalar[]>();
  const containers = new Map<string, JsonContainer>();
  let cursor = 0;
  const addScalar = (pointer: string, scalar: JsonScalar): void => {
    scalars.set(pointer, scalar);
    const occurrences = scalarOccurrences.get(pointer) ?? [];
    occurrences.push(scalar);
    scalarOccurrences.set(pointer, occurrences);
  };

  const fail = (message: string): never => {
    throw new JsonParseError(cursor, message);
  };
  const skipWhitespace = (): void => {
    while (cursor < source.length && /\s/.test(source[cursor]!)) cursor += 1;
  };
  const consume = (character: string): void => {
    if (source[cursor] !== character) fail(`Expected "${character}".`);
    cursor += 1;
  };
  const parseString = (): { value: string; span: SourceSpan } => {
    const start = cursor;
    consume('"');
    while (cursor < source.length) {
      const character = source[cursor]!;
      if (character === '"') {
        cursor += 1;
        const token = source.slice(start, cursor);
        try {
          return { value: JSON.parse(token) as string, span: { start, end: cursor } };
        } catch {
          fail("Invalid JSON string.");
        }
      }
      if (character === "\\") {
        cursor += 2;
        continue;
      }
      if (character.charCodeAt(0) < 0x20) fail("Unescaped control character.");
      cursor += 1;
    }
    return fail("Unterminated JSON string.");
  };
  const parseValue = (pointer: string): void => {
    skipWhitespace();
    const start = cursor;
    const character = source[cursor];
    if (character === '"') {
      const string = parseString();
      addScalar(pointer, { value: string.value, span: string.span });
      return;
    }
    if (character === "{") {
      const open = cursor;
      cursor += 1;
      skipWhitespace();
      if (source[cursor] === "}") {
        containers.set(pointer, { kind: "object", open, close: cursor });
        cursor += 1;
        return;
      }
      while (cursor < source.length) {
        skipWhitespace();
        if (source[cursor] !== '"') fail("Expected an object key.");
        const key = parseString().value;
        skipWhitespace();
        consume(":");
        parseValue(`${pointer}/${pointerSegment(key)}`);
        skipWhitespace();
        if (source[cursor] === "}") {
          containers.set(pointer, { kind: "object", open, close: cursor });
          cursor += 1;
          return;
        }
        consume(",");
      }
      fail("Unterminated JSON object.");
    }
    if (character === "[") {
      const open = cursor;
      cursor += 1;
      skipWhitespace();
      if (source[cursor] === "]") {
        containers.set(pointer, { kind: "array", open, close: cursor });
        cursor += 1;
        return;
      }
      let index = 0;
      while (cursor < source.length) {
        parseValue(`${pointer}/${index}`);
        index += 1;
        skipWhitespace();
        if (source[cursor] === "]") {
          containers.set(pointer, { kind: "array", open, close: cursor });
          cursor += 1;
          return;
        }
        consume(",");
      }
      fail("Unterminated JSON array.");
    }

    const remainder = source.slice(cursor);
    const literal = remainder.match(/^(?:true|false|null)(?![\w])/);
    if (literal) {
      cursor += literal[0].length;
      addScalar(pointer, {
        value: JSON.parse(literal[0]) as ConfigValue,
        span: { start, end: cursor },
      });
      return;
    }
    const number = remainder.match(/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/);
    if (number) {
      cursor += number[0].length;
      const unsafeInteger = /^-?\d+$/.test(number[0]) &&
        !Number.isSafeInteger(Number(number[0]));
      addScalar(pointer, {
        value: unsafeInteger ? number[0] : Number(number[0]),
        span: { start, end: cursor },
        ...(unsafeInteger ? { unsafeInteger: true } : {}),
      });
      return;
    }
    fail("Expected a JSON scalar, object, or array.");
  };

  try {
    parseValue("");
    skipWhitespace();
    if (cursor !== source.length) fail("Unexpected content after the JSON value.");
    return {
      format: "json",
      source,
      nodes: jsonNodes(source),
      issues: [],
      scalars,
      scalarOccurrences,
      containers,
    };
  } catch (error) {
    if (!(error instanceof JsonParseError)) throw error;
    const location = locationAt(source, error.position);
    return {
      format: "json",
      source,
      nodes: jsonNodes(source),
      issues: [
        {
          code: "invalid-json",
          message: error.message,
          severity: "error",
          ...location,
        },
      ],
      scalars: new Map(),
      scalarOccurrences: new Map(),
      containers: new Map(),
    };
  }
};

const decodePointerSegment = (value: string): string =>
  value.replace(/~1/g, "/").replace(/~0/g, "~");

const insertJsonProperty = (
  document: JsonDocument,
  pointer: string,
  value: ConfigValue,
): JsonDocument => {
  const separator = pointer.lastIndexOf("/");
  if (separator < 0) throw new Error(`JSON pointer "${pointer}" does not exist.`);
  const parentPointer = pointer.slice(0, separator);
  const property = decodePointerSegment(pointer.slice(separator + 1));
  const parent = document.containers.get(parentPointer);
  if (!parent || parent.kind !== "object") {
    throw new Error(`JSON pointer "${pointer}" does not exist.`);
  }

  const content = document.source.slice(parent.open + 1, parent.close);
  const contentEnd = parent.close - (content.match(/\s*$/)?.[0].length ?? 0);
  const closingLine = document.source.slice(0, parent.close).split("\n").at(-1) ?? "";
  const closingIndent = closingLine.match(/^\s*/)?.[0] ?? "";
  const serialized = `${JSON.stringify(property)}: ${JSON.stringify(value)}`;
  const newline = document.source.includes("\r\n") ? "\r\n" : "\n";
  let insertion: string;
  if (content.trim() === "") {
    insertion = `${newline}${closingIndent}  ${serialized}${newline}${closingIndent}`;
    return parseJson(
      document.source.slice(0, parent.open + 1) +
        insertion +
        document.source.slice(parent.close),
    );
  }

  const multiline = content.includes("\n");
  if (!multiline) insertion = `, ${serialized}`;
  else {
    const existingLine = document.source.slice(0, contentEnd).split("\n").at(-1) ?? "";
    const childIndent = existingLine.match(/^\s*/)?.[0] ?? `${closingIndent}  `;
    insertion = `,${newline}${childIndent}${serialized}`;
  }
  return parseJson(
    document.source.slice(0, contentEnd) +
      insertion +
      document.source.slice(contentEnd),
  );
};

export const jsonAdapter: ConfigAdapter<JsonDocument> = {
  parse: parseJson,
  entries(document) {
    return [...document.scalars].map(([key, scalar]) => ({
      key,
      value: scalar.value,
    }));
  },
  get(document, key) {
    return document.scalars.get(key)?.value;
  },
  getAll(document, key) {
    return (document.scalarOccurrences.get(key) ?? []).map((scalar) => scalar.value);
  },
  getAllRaw(document, key) {
    return (document.scalarOccurrences.get(key) ?? []).map((scalar) =>
      document.source.slice(scalar.span.start, scalar.span.end)
    );
  },
  set(document, key, value) {
    if (document.issues.length > 0) throw new Error("Invalid JSON cannot be edited in form mode.");
    const scalar = document.scalars.get(key);
    if (!scalar) {
      if (document.containers.has(key)) throw new Error(`JSON pointer "${key}" is not a scalar.`);
      return insertJsonProperty(document, key, value);
    }
    const serialized = scalar.unsafeInteger && typeof value === "string" && /^-?\d+$/.test(value)
      ? value
      : JSON.stringify(value);
    return parseJson(
      document.source.slice(0, scalar.span.start) +
        serialized +
        document.source.slice(scalar.span.end),
    );
  },
  setAll(document, key, values) {
    const occurrences = document.scalarOccurrences.get(key) ?? [];
    const expected = occurrences.length;
    if (values.length !== expected) {
      throw new Error(`Expected ${expected} values for JSON pointer "${key}".`);
    }
    let source = document.source;
    for (let index = occurrences.length - 1; index >= 0; index -= 1) {
      const scalar = occurrences[index]!;
      const value = values[index]!;
      const serialized = scalar.unsafeInteger && typeof value === "string" && /^-?\d+$/.test(value)
        ? value
        : JSON.stringify(value);
      source = source.slice(0, scalar.span.start) + serialized + source.slice(scalar.span.end);
    }
    return parseJson(source);
  },
  setAllRaw(document, key, values) {
    const occurrences = document.scalarOccurrences.get(key) ?? [];
    const expected = occurrences.length;
    if (values.length !== expected) {
      throw new Error(`Expected ${expected} raw values for JSON pointer "${key}".`);
    }
    let source = document.source;
    for (let index = occurrences.length - 1; index >= 0; index -= 1) {
      const scalar = occurrences[index]!;
      source = source.slice(0, scalar.span.start) + values[index]! + source.slice(scalar.span.end);
    }
    return parseJson(source);
  },
  validate(document, schema: FileSchema): ValidationIssue[] {
    void schema;
    return [...document.issues];
  },
  serialize(document) {
    return document.source;
  },
};
