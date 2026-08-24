import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import { EDITOR_STYLES } from "./styles.js";

function parsedStyleRules(): CSSStyleRule[] {
  const dom = new JSDOM(`<style>${EDITOR_STYLES}</style>`);
  const sheet = dom.window.document.styleSheets.item(0);

  if (!sheet) {
    throw new Error("Expected the config editor stylesheet to be parsed");
  }

  return Array.from(sheet.cssRules).filter(
    (rule): rule is CSSStyleRule => "selectorText" in rule,
  );
}

function findRule(rules: CSSStyleRule[], selectorFragment: string): CSSStyleRule | undefined {
  return rules.find((rule) => rule.selectorText.includes(selectorFragment));
}

describe("config editor scrollbar theme", () => {
  it("themes every scrollable editor surface with the shared Druid scrollbar", () => {
    const rules = parsedStyleRules();
    const sharedRule = findRule(rules, ".editor-main, .file-rail, .raw-textarea");
    const webkitScrollbar = findRule(rules, "::-webkit-scrollbar");
    const track = findRule(rules, "::-webkit-scrollbar-track");
    const thumb = findRule(rules, "::-webkit-scrollbar-thumb");
    const button = findRule(rules, "::-webkit-scrollbar-button");
    const corner = findRule(rules, "::-webkit-scrollbar-corner");

    expect(sharedRule).toBeDefined();
    expect(sharedRule?.style.getPropertyValue("scrollbar-width")).toBe("thin");
    expect(sharedRule?.style.getPropertyValue("scrollbar-color")).toBe(
      "var(--druid-border-strong) transparent",
    );

    expect(webkitScrollbar?.style.width).toBe("10px");
    expect(webkitScrollbar?.style.height).toBe("10px");
    expect(track?.style.background).toBe("transparent");
    expect(thumb?.style.getPropertyValue("background-color")).toBe(
      "var(--druid-border-strong)",
    );
    expect(thumb?.style.getPropertyValue("border-radius")).toBe("999px");
    expect(thumb?.style.getPropertyValue("background-clip")).toBe("content-box");
    expect(button?.style.display).toBe("none");
    expect(button?.style.width).toBe("0");
    expect(button?.style.height).toBe("0");
    expect(corner?.style.background).toBe("transparent");
  });
});
