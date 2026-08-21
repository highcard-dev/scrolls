import type { ConfigValue, FieldSchema, ValidationIssue } from "./model.js";

const issue = (
  field: FieldSchema,
  code: string,
  message: string,
): ValidationIssue => ({
  code,
  message,
  severity: "error",
  fieldKey: field.key,
});

const scalarText = (input: unknown): string => {
  if (input === null) return "";
  if (typeof input === "string") return input;
  if (typeof input === "number" || typeof input === "boolean") return String(input);
  throw new TypeError("Value must be a scalar.");
};

export const coerceFieldValue = (
  field: FieldSchema,
  input: unknown,
): ConfigValue => {
  if (field.type === "string" || field.type === "secret" || field.type === "enum") {
    return scalarText(input);
  }
  if (field.type === "boolean") {
    if (input === true || input === false) return input;
    if (input === "true") return true;
    if (input === "false") return false;
    throw new TypeError("Value must be true or false.");
  }

  if (typeof input === "number") {
    if (!Number.isFinite(input)) throw new TypeError("Value must be a finite number.");
    if (field.type === "integer" && !Number.isSafeInteger(input)) {
      throw new TypeError("Value must be an integer.");
    }
    return input;
  }

  const text = scalarText(input).trim();
  if (field.type === "integer") {
    if (!/^[+-]?\d+$/.test(text)) throw new TypeError("Value must be an integer.");
    const value = Number(text);
    if (!Number.isSafeInteger(value)) throw new TypeError("Value must be an integer.");
    return value;
  }
  if (!/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$/.test(text)) {
    throw new TypeError("Value must be a finite number.");
  }
  const value = Number(text);
  if (!Number.isFinite(value)) throw new TypeError("Value must be a finite number.");
  return value;
};

export const validateField = (
  field: FieldSchema,
  value: ConfigValue,
): ValidationIssue[] => {
  const issues: ValidationIssue[] = [];

  if (field.type === "integer" && (typeof value !== "number" || !Number.isSafeInteger(value))) {
    issues.push(issue(field, "type", "Value must be an integer."));
    return issues;
  }
  if (field.type === "number" && (typeof value !== "number" || !Number.isFinite(value))) {
    issues.push(issue(field, "type", "Value must be a finite number."));
    return issues;
  }
  if (field.type === "boolean" && typeof value !== "boolean") {
    issues.push(issue(field, "type", "Value must be true or false."));
    return issues;
  }
  if (
    (field.type === "string" || field.type === "secret" || field.type === "enum") &&
    typeof value !== "string"
  ) {
    issues.push(issue(field, "type", "Value must be text."));
    return issues;
  }

  if (typeof value === "number") {
    const below = field.min !== undefined && value < field.min;
    const above = field.max !== undefined && value > field.max;
    if (below || above) {
      let message: string;
      if (field.min !== undefined && field.max !== undefined) {
        message = `Value must be between ${field.min} and ${field.max} (inclusive).`;
      } else if (field.min !== undefined) message = `Value must be at least ${field.min}.`;
      else message = `Value must be at most ${field.max}.`;
      issues.push(issue(field, "range", message));
    }
  }

  if (field.type === "enum" && typeof value === "string" && !field.values?.includes(value)) {
    issues.push(
      issue(field, "enum", `Value must be one of: ${(field.values ?? []).join(", ")}.`),
    );
  }

  if (field.pattern !== undefined && typeof value === "string") {
    const pattern = new RegExp(`^(?:${field.pattern})$`);
    if (!pattern.test(value)) {
      issues.push(issue(field, "pattern", "Value does not match the required format."));
    }
  }

  return issues;
};
