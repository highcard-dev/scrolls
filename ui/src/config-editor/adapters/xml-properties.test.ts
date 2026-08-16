import { describe, expect, it } from "vitest";
import { xmlPropertiesAdapter } from "./xml-properties.js";

describe("xmlPropertiesAdapter", () => {
  it("discovers 7 Days to Die property elements", () => {
    const document = xmlPropertiesAdapter.parse(
      '<ServerSettings>\n  <property name="ServerName" value="Druid"/>\n  <property name="MaxPlayerCount" value="8"/>\n</ServerSettings>\n',
    );

    expect(xmlPropertiesAdapter.entries(document)).toEqual([
      { key: "ServerName", value: "Druid" },
      { key: "MaxPlayerCount", value: "8" },
    ]);
  });

  it("changes only the selected XML attribute value", () => {
    const source = '<property name="ServerName" value="Druid &amp; Friends" />\n';
    const document = xmlPropertiesAdapter.parse(source);
    const updated = xmlPropertiesAdapter.set(document, "ServerName", 'Druid "EU" & Friends');

    expect(xmlPropertiesAdapter.get(document, "ServerName")).toBe("Druid & Friends");
    expect(xmlPropertiesAdapter.serialize(updated)).toBe(
      '<property name="ServerName" value="Druid &quot;EU&quot; &amp; Friends" />\n',
    );
  });

  it("uses the last duplicate as the effective setting", () => {
    const document = xmlPropertiesAdapter.parse(
      '<property name="ServerName" value="old"/>\n<property name="ServerName" value="new"/>\n',
    );

    expect(xmlPropertiesAdapter.get(document, "ServerName")).toBe("new");
    expect(xmlPropertiesAdapter.entries(document)).toEqual([
      { key: "ServerName", value: "new" },
    ]);
  });
});
