import { describe, expect, it } from "vitest";
import { javaPropertiesAdapter } from "./adapters/java-properties.js";
import { keyValueAdapter } from "./adapters/ini.js";
import { rawAdapter } from "./adapters/raw.js";
import { jsonAdapter } from "./adapters/json.js";
import type { FieldSchema, FileSchema } from "./model.js";
import { ConfigEditorStore, MASKED_SECRET } from "./store.js";

const baseField = (overrides: Partial<FieldSchema> = {}): FieldSchema => ({
  key: "max-players",
  label: "Max players",
  description: "Maximum player count.",
  documentation: "https://minecraft.wiki/w/Server.properties",
  type: "integer",
  min: 1,
  max: 1000,
  ...overrides,
});

const fileSchema = (fields: FieldSchema[]): FileSchema => ({
  path: "server.properties",
  format: "java-properties",
  label: "server.properties",
  sections: [{ id: "general", label: "General", fields }],
});

const loadedStore = (source: string, schema: FileSchema): ConfigEditorStore =>
  ConfigEditorStore.fromLoadedFile(schema, javaPropertiesAdapter.parse(source));

describe("ConfigEditorStore", () => {
  it("does not serialize an unchanged masked secret", () => {
    const store = loadedStore(
      "rcon.password=actual-secret\n",
      fileSchema([
        baseField({
          key: "rcon.password",
          label: "RCON password",
          type: "secret",
          min: undefined,
          max: undefined,
          sensitive: true,
        }),
      ]),
    );
    expect(store.snapshot().fields["rcon.password"]!.displayValue).toBe(
      MASKED_SECRET,
    );
    store.setDisplayValue("rcon.password", MASKED_SECRET);
    expect(store.serializeSelectedFile()).toBe(
      "rcon.password=actual-secret\n",
    );
  });

  it("tracks changes and restart impact by stable key", () => {
    const store = loadedStore(
      "view-distance=10\n",
      fileSchema([
        baseField({
          key: "view-distance",
          label: "View distance",
          min: 1,
          max: 32,
          restartRequired: true,
        }),
      ]),
    );
    store.setDisplayValue("view-distance", "14");
    expect(store.snapshot().changes).toEqual([
      expect.objectContaining({
        key: "view-distance",
        before: 10,
        after: 14,
        restartRequired: true,
      }),
    ]);
    expect(store.snapshot().restartRequired).toBe(true);
    expect(store.serializeSelectedFile()).toBe("view-distance=14\n");
  });

  it("keeps invalid display input without corrupting serialized content", () => {
    const store = loadedStore("max-players=20\n", fileSchema([baseField()]));
    store.setDisplayValue("max-players", "0");
    const snapshot = store.snapshot();
    expect(snapshot.fields["max-players"]!.displayValue).toBe("0");
    expect(snapshot.fields["max-players"]!.issues).toEqual([
      expect.objectContaining({ code: "range" }),
    ]);
    expect(snapshot.dirty).toBe(false);
    expect(store.serializeSelectedFile()).toBe("max-players=20\n");
  });

  it("redacts secret values from change summaries and immutable snapshots", () => {
    const store = loadedStore(
      "password=old-secret\n",
      fileSchema([
        baseField({
          key: "password",
          type: "secret",
          min: undefined,
          max: undefined,
          sensitive: true,
        }),
      ]),
    );
    store.setDisplayValue("password", "new-secret");
    const snapshot = store.snapshot();
    expect(JSON.stringify(snapshot)).not.toContain("old-secret");
    expect(JSON.stringify(snapshot)).not.toContain("new-secret");
    expect(snapshot.changes[0]).toEqual(
      expect.objectContaining({ before: null, after: null, sensitive: true }),
    );
    expect(Object.isFrozen(snapshot)).toBe(true);
    expect(Object.isFrozen(snapshot.fields)).toBe(true);
  });

  it("redacts secrets in raw display while preserving them through raw edits", () => {
    const store = loadedStore(
      "motd=hello\nrcon.password=actual-secret\nfuture=preserved\n",
      fileSchema([
        baseField({ key: "motd", type: "string", min: undefined, max: undefined }),
        baseField({
          key: "rcon.password",
          type: "secret",
          min: undefined,
          max: undefined,
          sensitive: true,
        }),
      ]),
    );

    const display = store.serializeForDisplay();
    expect(display).toContain(`rcon.password=${MASKED_SECRET}`);
    expect(display).not.toContain("actual-secret");

    store.replaceWorkingSource(display.replace("motd=hello", "motd=updated"));

    expect(store.serializeSelectedFile()).toContain("motd=updated");
    expect(store.serializeSelectedFile()).toContain("rcon.password=actual-secret");
    expect(store.serializeSelectedFile()).toContain("future=preserved");
  });

  it("redacts and restores every duplicate secret occurrence", () => {
    const store = loadedStore(
      "password=first-secret\npassword=second-secret\nmotd=hello\n",
      fileSchema([]),
    );

    const display = store.serializeForDisplay();
    expect(display).not.toContain("first-secret");
    expect(display).not.toContain("second-secret");
    expect(display.match(new RegExp(MASKED_SECRET, "g"))).toHaveLength(2);

    store.replaceWorkingSource(display.replace("motd=hello", "motd=updated"));
    expect(store.serializeSelectedFile()).toBe(
      "password=first-secret\npassword=second-secret\nmotd=updated\n",
    );
  });

  it("restores untouched secret source tokens byte-for-byte after raw edits", () => {
    const schema: FileSchema = {
      ...fileSchema([]),
      format: "key-value",
    };
    const store = ConfigEditorStore.fromLoadedFile(
      schema,
      keyValueAdapter.parse('password="a\\"b"\nmotd=hello\n'),
    );

    const display = store.serializeForDisplay();
    store.replaceWorkingSource(display.replace("motd=hello", "motd=updated"));
    expect(store.serializeSelectedFile()).toBe('password="a\\"b"\nmotd=updated\n');
  });

  it("redacts every duplicate secret in JSON raw mode", () => {
    const schema: FileSchema = {
      path: "config.json",
      format: "json",
      label: "config.json",
      sections: [],
    };
    const store = ConfigEditorStore.fromLoadedFile(
      schema,
      jsonAdapter.parse('{"password":"first-secret","password":"second-secret","name":"Druid"}\n'),
    );
    const display = store.serializeForDisplay();
    expect(display).not.toContain("first-secret");
    expect(display).not.toContain("second-secret");
    store.replaceWorkingSource(display.replace('"Druid"', '"Druid EU"'));
    expect(store.serializeSelectedFile()).toBe(
      '{"password":"first-secret","password":"second-secret","name":"Druid EU"}\n',
    );
  });

  it("marks raw-only document edits dirty so they can be saved", () => {
    const schema: FileSchema = {
      path: "serverconfig.xml",
      format: "raw",
      label: "serverconfig.xml",
      sections: [],
    };
    const store = ConfigEditorStore.fromLoadedFile(
      schema,
      rawAdapter.parse("<ServerSettings />\n"),
    );

    store.replaceWorkingSource("<ServerSettings Name=\"Druid\" />\n");

    expect(store.snapshot().dirty).toBe(true);
    expect(store.snapshot().unstructuredChanges).toBe(true);
    expect(store.snapshot().restartRequired).toBe(true);
  });

  it("marks unknown-key raw edits dirty in a typed document", () => {
    const store = loadedStore(
      "motd=hello\nfuture-setting=before\n",
      fileSchema([
        baseField({ key: "motd", type: "string", min: undefined, max: undefined }),
      ]),
    );

    store.replaceWorkingSource("motd=hello\nfuture-setting=after\n");

    expect(store.snapshot().dirty).toBe(true);
  });

  it("discovers loaded unknown keys as typed form fields", () => {
    const store = loadedStore(
      "motd=hello\nenable-feature=true\nworker-count=4\nratio=1.5\n",
      fileSchema([
        baseField({ key: "motd", type: "string", min: undefined, max: undefined }),
      ]),
    );

    const detected = store.schema.sections.find(
      (section) => section.id === "detected-settings",
    );

    expect(detected?.fields.map(({ key, type }) => ({ key, type }))).toEqual([
      { key: "enable-feature", type: "boolean" },
      { key: "worker-count", type: "string" },
      { key: "ratio", type: "string" },
    ]);
  });

  it("keeps undeclared numeric-looking strings lossless", () => {
    const store = loadedStore(
      "seed=000012340000123400001234\nsteamid=76561198012345678\n",
      fileSchema([]),
    );

    expect(store.snapshot().fields.seed).toEqual(
      expect.objectContaining({ type: "string", value: "000012340000123400001234" }),
    );
    expect(store.snapshot().fields.steamid).toEqual(
      expect.objectContaining({ type: "string", value: "76561198012345678" }),
    );
  });

  it("discovers keys added in raw mode without duplicating declared fields", () => {
    const store = loadedStore(
      "motd=hello\n",
      fileSchema([
        baseField({ key: "motd", type: "string", min: undefined, max: undefined }),
      ]),
    );

    store.replaceWorkingSource("motd=updated\nnew-option=enabled\n");

    expect(
      store.schema.sections.flatMap((section) => section.fields.map((field) => field.key)),
    ).toEqual(["motd", "new-option"]);
  });

  it("masks automatically discovered secret fields", () => {
    const store = loadedStore(
      "rcon.password=actual-secret\n",
      fileSchema([]),
    );

    const secret = store.snapshot().fields["rcon.password"];

    expect(secret).toEqual(
      expect.objectContaining({
        type: "secret",
        sensitive: true,
        displayValue: MASKED_SECRET,
      }),
    );
    expect(JSON.stringify(secret)).not.toContain("actual-secret");
    expect(store.serializeForDisplay()).not.toContain("actual-secret");
  });

  it("detects common game-server credentials without masking unrelated keys", () => {
    const store = loadedStore(
      "steampass=steam-secret\ngslt=token-value\nserverPassword=join-secret\npass=short-secret\ncompass=north\n",
      fileSchema([]),
    );

    for (const key of ["steampass", "gslt", "serverPassword", "pass"]) {
      expect(store.snapshot().fields[key]).toEqual(
        expect.objectContaining({ type: "secret", sensitive: true }),
      );
    }
    expect(store.snapshot().fields.compass).toEqual(
      expect.objectContaining({ type: "string", sensitive: false, value: "north" }),
    );
    const display = store.serializeForDisplay();
    expect(display).not.toContain("steam-secret");
    expect(display).not.toContain("token-value");
    expect(display).not.toContain("join-secret");
    expect(display).not.toContain("short-secret");
    expect(display).toContain("compass=north");
  });

  it("can add a declared JSON scalar that is absent from the loaded file", () => {
    const schema: FileSchema = {
      path: "config.json",
      format: "json",
      label: "config.json",
      sections: [{
        id: "server",
        label: "Server",
        fields: [baseField({
          key: "/port",
          label: "Port",
          type: "integer",
          min: 1,
          max: 65535,
        })],
      }],
    };
    const store = ConfigEditorStore.fromLoadedFile(schema, jsonAdapter.parse("{}\n"));

    store.setDisplayValue("/port", "5520");

    expect(store.serializeSelectedFile()).toBe('{\n  "port": 5520\n}\n');
  });

  it("shows a declared default for a missing setting without changing the file", () => {
    const store = loadedStore(
      "motd=Druid\n",
      fileSchema([
        baseField({
          key: "online-mode",
          label: "Online mode",
          type: "boolean",
          min: undefined,
          max: undefined,
          defaultValue: true,
        }),
      ]),
    );

    expect(store.snapshot().fields["online-mode"]).toEqual(
      expect.objectContaining({
        displayValue: "true",
        value: true,
        dirty: false,
        issues: [],
      }),
    );
    expect(store.serializeSelectedFile()).toBe("motd=Druid\n");
  });
});
