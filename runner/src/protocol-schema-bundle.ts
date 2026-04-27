/**
 * Protocol schema bundle loader for untrusted runner validation.
 *
 * This module only reads the allowed cross-boundary schema inventory under
 * protocol/schemas and provides fail-closed schema validation helpers.
 */

import { readFile } from "node:fs/promises";
import path from "node:path";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";

type JsonObject = Record<string, unknown>;

type SchemaManifestEntry = {
  path: string;
  schema_id: string;
  schema_version: string;
};

type SchemaManifest = {
  schema_files: SchemaManifestEntry[];
};

export type SchemaValidationResult =
  | { ok: true }
  | { ok: false; reason: string };

function schemaKey(schemaId: string, schemaVersion: string): string {
  return `${schemaId}@${schemaVersion}`;
}

export class ProtocolSchemaBundle {
  private readonly ajv: Ajv2020;

  private readonly schemaPathByRuntimeKey: Map<string, string>;

  private constructor(ajv: Ajv2020, schemaPathByRuntimeKey: Map<string, string>) {
    this.ajv = ajv;
    this.schemaPathByRuntimeKey = schemaPathByRuntimeKey;
  }

  static async fromProtocolSchemasRoot(protocolSchemasRoot: string): Promise<ProtocolSchemaBundle> {
    const manifestPath = path.join(protocolSchemasRoot, "manifest.json");
    const manifest = await readJsonFile<SchemaManifest>(manifestPath);

    const ajv = new Ajv2020({ allErrors: true, strict: false });
    addFormats(ajv);

    const schemaPathByRuntimeKey = new Map<string, string>();
    for (const entry of manifest.schema_files) {
      const schemaPath = path.join(protocolSchemasRoot, entry.path);
      const schema = await readJsonFile<JsonObject>(schemaPath);
      ajv.addSchema(schema);
      schemaPathByRuntimeKey.set(schemaKey(entry.schema_id, entry.schema_version), entry.path);
    }

    return new ProtocolSchemaBundle(ajv, schemaPathByRuntimeKey);
  }

  validateByRuntimeKey(schemaId: string, schemaVersion: string, value: unknown): SchemaValidationResult {
    const runtimeKey = schemaKey(schemaId, schemaVersion);
    const schemaPath = this.schemaPathByRuntimeKey.get(runtimeKey);
    if (!schemaPath) {
      return { ok: false, reason: `schema ${runtimeKey} not found in manifest` };
    }

    const validate = this.ajv.getSchema(`https://runecode.dev/protocol/schemas/${schemaPath}`);
    if (!validate) {
      return { ok: false, reason: `validator for ${runtimeKey} is unavailable` };
    }

    if (validate(value)) {
      return { ok: true };
    }

    return { ok: false, reason: JSON.stringify(validate.errors ?? []) };
  }
}

async function readJsonFile<T>(filePath: string): Promise<T> {
  const text = await readFile(filePath, "utf8");
  return JSON.parse(text) as T;
}
