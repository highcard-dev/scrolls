import { describe, expect, it } from "vitest";
import { iniAdapter, keyValueAdapter, unrealIniAdapter } from "./ini.js";

describe("iniAdapter", () => {
  it("changes only a value in the selected section", () => {
    const source =
      "; generated\r\n[Server]\r\nName = Old ; keep this\r\nPlayers=20\r\n\r\n[Other]\r\nName=Untouched\r\n";
    const document = iniAdapter.parse(source);
    expect(iniAdapter.get(document, "Server.Name")).toBe("Old");
    const changed = iniAdapter.set(document, "Server.Name", "Druid");
    expect(iniAdapter.serialize(changed)).toBe(
      source.replace("Name = Old", "Name = Druid"),
    );
  });

  it("uses the last duplicate key and keeps all other lines", () => {
    const source = "[Server]\nName=old\nName=new\n";
    const changed = iniAdapter.set(iniAdapter.parse(source), "Server.Name", "final");
    expect(iniAdapter.get(changed, "Server.Name")).toBe("final");
    expect(iniAdapter.serialize(changed)).toBe(
      "[Server]\nName=old\nName=final\n",
    );
  });

  it("preserves an untouched document byte-for-byte", () => {
    const source = "# comment\nroot=value\n[Section]\nempty=\n";
    expect(iniAdapter.serialize(iniAdapter.parse(source))).toBe(source);
  });
});

describe("unrealIniAdapter", () => {
  it("changes only an Unreal INI value in the selected section", () => {
    const source =
      "[/Script/ShooterGame.ShooterGameUserSettings]\r\nServerPassword=\r\nDifficultyOffset=0.2\r\n";
    const changed = unrealIniAdapter.set(
      unrealIniAdapter.parse(source),
      "/Script/ShooterGame.ShooterGameUserSettings.DifficultyOffset",
      1,
    );
    expect(unrealIniAdapter.serialize(changed)).toBe(
      source.replace("0.2", "1"),
    );
  });

  it("retains Unreal array prefixes", () => {
    const source = "[Rules]\n+AllowedClasses=One\n+AllowedClasses=Two\n";
    const changed = unrealIniAdapter.set(
      unrealIniAdapter.parse(source),
      "Rules.+AllowedClasses",
      "Three",
    );
    expect(unrealIniAdapter.serialize(changed)).toBe(
      "[Rules]\n+AllowedClasses=One\n+AllowedClasses=Three\n",
    );
  });
});

describe("keyValueAdapter", () => {
  it("preserves comments and inline spacing while changing one value", () => {
    const source = "// server config\nhostname   Druid\nmaxplayers = 20 // capacity\n";
    const document = keyValueAdapter.parse(source);
    expect(keyValueAdapter.get(document, "hostname")).toBe("Druid");
    expect(
      keyValueAdapter.serialize(keyValueAdapter.set(document, "maxplayers", 64)),
    ).toBe("// server config\nhostname   Druid\nmaxplayers = 64 // capacity\n");
  });

  it("preserves shell-style quotes while editing their contents", () => {
    const source = 'servername="Palworld Server"\n';
    const document = keyValueAdapter.parse(source);

    expect(keyValueAdapter.get(document, "servername")).toBe("Palworld Server");
    expect(
      keyValueAdapter.serialize(
        keyValueAdapter.set(document, "servername", "Druid Server"),
      ),
    ).toBe('servername="Druid Server"\n');
  });

  it("escapes matching quote characters inside preserved double quotes", () => {
    const document = keyValueAdapter.parse('servername="Palworld Server"\n');

    const changed = keyValueAdapter.set(document, "servername", 'Druid "EU"');
    expect(keyValueAdapter.serialize(changed)).toBe('servername="Druid \\"EU\\""\n');
    expect(keyValueAdapter.get(changed, "servername")).toBe('Druid "EU"');
  });

  it("uses POSIX-safe quoting inside preserved shell single quotes", () => {
    const document = keyValueAdapter.parse("servername='Druid Server'\n");
    const changed = keyValueAdapter.set(document, "servername", "Druid's Server");
    expect(keyValueAdapter.serialize(changed)).toBe("servername='Druid'\\''s Server'\n");
    expect(keyValueAdapter.get(changed, "servername")).toBe("Druid's Server");
  });

  it("roundtrips POSIX apostrophes without consuming inline comments", () => {
    const document = keyValueAdapter.parse("servername='Old' # public name\n");
    const changed = keyValueAdapter.set(document, "servername", "Druid's");
    const serialized = keyValueAdapter.serialize(changed);

    expect(serialized).toBe("servername='Druid'\\''s' # public name\n");
    expect(keyValueAdapter.get(keyValueAdapter.parse(serialized), "servername")).toBe("Druid's");
    expect(
      keyValueAdapter.serialize(
        keyValueAdapter.set(keyValueAdapter.parse(serialized), "servername", "Druid's EU"),
      ),
    ).toBe("servername='Druid'\\''s EU' # public name\n");
  });

  it("does not treat comment markers inside quotes as inline comments", () => {
    const document = keyValueAdapter.parse('servername="Druid #1" # public name\n');

    expect(keyValueAdapter.get(document, "servername")).toBe("Druid #1");
    expect(
      keyValueAdapter.serialize(keyValueAdapter.set(document, "servername", "Druid #2")),
    ).toBe('servername="Druid #2" # public name\n');
  });

  it("preserves DayZ quotes and statement terminators", () => {
    const document = keyValueAdapter.parse('hostname = "Druid Server";\nmaxPlayers = 60;\n');

    expect(keyValueAdapter.get(document, "hostname")).toBe("Druid Server");
    expect(keyValueAdapter.get(document, "maxPlayers")).toBe("60");
    expect(
      keyValueAdapter.serialize(keyValueAdapter.set(document, "hostname", "Druid EU")),
    ).toBe('hostname = "Druid EU";\nmaxPlayers = 60;\n');
  });
});

describe("unrealIniAdapter OptionSettings", () => {
  it("discovers and edits Palworld tuple options independently", () => {
    const source =
      '[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(Difficulty=None,ServerName="Palworld, EU",bIsPvP=false,DayTimeSpeedRate=1.000000)\n';
    const document = unrealIniAdapter.parse(source);

    expect(unrealIniAdapter.entries(document)).toEqual([
      { key: "/Script/Pal.PalGameWorldSettings.OptionSettings.Difficulty", value: "None" },
      { key: "/Script/Pal.PalGameWorldSettings.OptionSettings.ServerName", value: "Palworld, EU" },
      { key: "/Script/Pal.PalGameWorldSettings.OptionSettings.bIsPvP", value: "false" },
      { key: "/Script/Pal.PalGameWorldSettings.OptionSettings.DayTimeSpeedRate", value: "1.000000" },
    ]);

    const updated = unrealIniAdapter.set(
      document,
      "/Script/Pal.PalGameWorldSettings.OptionSettings.ServerName",
      "Druid, EU",
    );
    expect(unrealIniAdapter.serialize(updated)).toBe(
      source.replace('ServerName="Palworld, EU"', 'ServerName="Druid, EU"'),
    );
  });
});
