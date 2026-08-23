import { describe, expect, it } from "vitest";
import { javaPropertiesAdapter } from "./java-properties.js";

const source =
  "#Minecraft server properties\r\nmax-players = 20\r\nunknown.future:key\\ value\r\n\r\n";

describe("javaPropertiesAdapter", () => {
  it("serializes an untouched properties file byte-for-byte", () => {
    const document = javaPropertiesAdapter.parse(source);
    expect(javaPropertiesAdapter.serialize(document)).toBe(source);
  });

  it("changes only the selected value token", () => {
    const document = javaPropertiesAdapter.parse(source);
    const changed = javaPropertiesAdapter.set(document, "max-players", 50);
    expect(javaPropertiesAdapter.serialize(changed)).toBe(
      "#Minecraft server properties\r\nmax-players = 50\r\nunknown.future:key\\ value\r\n\r\n",
    );
  });

  it("uses the last duplicate key as the effective value and retains every line", () => {
    const duplicateSource = "motd=old\nmotd=new\n";
    const document = javaPropertiesAdapter.parse(duplicateSource);
    expect(javaPropertiesAdapter.get(document, "motd")).toBe("new");
    expect(
      javaPropertiesAdapter.serialize(
        javaPropertiesAdapter.set(document, "motd", "final"),
      ),
    ).toBe("motd=old\nmotd=final\n");
  });

  it.each([
    ["equals", "name=value\n", "name", "value"],
    ["colon", "name:value\n", "name", "value"],
    ["whitespace", "name   value\n", "name", "value"],
    ["empty", "name=\n", "name", ""],
    ["escaped separator", "part\\=name=value\n", "part=name", "value"],
    ["escaped space", "server\\ name=Druid\n", "server name", "Druid"],
    ["unicode", "motd=Grüße 世界\n", "motd", "Grüße 世界"],
    ["unicode escape", "motd=Hello\\u0020World\n", "motd", "Hello World"],
  ])("reads %s syntax", (_name, input, key, expected) => {
    expect(javaPropertiesAdapter.get(javaPropertiesAdapter.parse(input), key)).toBe(
      expected,
    );
  });

  it.each([
    "key=value",
    "key=value\n",
    "key=value\r\n",
    "# comment\n! another\n\nkey=value\n",
    "continued=first\\\n  second\n",
  ])("preserves untouched source %j", (input) => {
    expect(
      javaPropertiesAdapter.serialize(javaPropertiesAdapter.parse(input)),
    ).toBe(input);
  });

  it("decodes a continued logical value without changing its source", () => {
    const input = "motd=Hello \\\r\n  Druid\\ World\r\n";
    const document = javaPropertiesAdapter.parse(input);
    expect(javaPropertiesAdapter.get(document, "motd")).toBe(
      "Hello Druid World",
    );
    expect(javaPropertiesAdapter.serialize(document)).toBe(input);
  });

  it("appends a missing key using the existing line ending and final-newline policy", () => {
    expect(
      javaPropertiesAdapter.serialize(
        javaPropertiesAdapter.set(
          javaPropertiesAdapter.parse("motd=Druid\r\n"),
          "max-players",
          20,
        ),
      ),
    ).toBe("motd=Druid\r\nmax-players=20\r\n");

    expect(
      javaPropertiesAdapter.serialize(
        javaPropertiesAdapter.set(
          javaPropertiesAdapter.parse("motd=Druid"),
          "max-players",
          20,
        ),
      ),
    ).toBe("motd=Druid\nmax-players=20");
  });

  it("escapes a newly appended key and string value safely", () => {
    const changed = javaPropertiesAdapter.set(
      javaPropertiesAdapter.parse(""),
      "server name",
      "Druid\\Realm\nLine",
    );
    expect(javaPropertiesAdapter.serialize(changed)).toBe(
      "server\\ name=Druid\\\\Realm\\nLine",
    );
    expect(javaPropertiesAdapter.get(changed, "server name")).toBe(
      "Druid\\Realm\nLine",
    );
  });
});
