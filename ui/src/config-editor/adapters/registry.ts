import type { ConfigFormat } from "../model.js";
import { iniAdapter, keyValueAdapter, unrealIniAdapter } from "./ini.js";
import { javaPropertiesAdapter } from "./java-properties.js";
import { jsonAdapter } from "./json.js";
import { rawAdapter } from "./raw.js";
import { xmlPropertiesAdapter } from "./xml-properties.js";
import type { ConfigAdapter } from "./types.js";

export type AdapterRegistry = ReadonlyMap<ConfigFormat, ConfigAdapter>;

export const createAdapterRegistry = (): AdapterRegistry =>
  new Map<ConfigFormat, ConfigAdapter>([
    ["java-properties", javaPropertiesAdapter],
    ["ini", iniAdapter],
    ["unreal-ini", unrealIniAdapter],
    ["key-value", keyValueAdapter],
    ["json", jsonAdapter],
    ["xml-properties", xmlPropertiesAdapter],
    ["raw", rawAdapter],
  ]);
