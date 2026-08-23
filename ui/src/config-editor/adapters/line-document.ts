import type { SourceLine } from "./types.js";

export const splitSourceLines = (source: string): SourceLine[] => {
  const lines: SourceLine[] = [];
  let start = 0;

  while (start < source.length) {
    const newline = source.indexOf("\n", start);
    if (newline === -1) {
      lines.push({ raw: source.slice(start), ending: "", start, end: source.length });
      break;
    }

    const hasCarriageReturn = newline > start && source[newline - 1] === "\r";
    const rawEnd = hasCarriageReturn ? newline - 1 : newline;
    lines.push({
      raw: source.slice(start, rawEnd),
      ending: hasCarriageReturn ? "\r\n" : "\n",
      start,
      end: rawEnd,
    });
    start = newline + 1;
  }

  return lines;
};

export const preferredLineEnding = (
  lines: readonly SourceLine[],
): "\n" | "\r\n" =>
  lines.find((line) => line.ending !== "")?.ending || "\n";

export const hasFinalNewline = (source: string): boolean => source.endsWith("\n");
