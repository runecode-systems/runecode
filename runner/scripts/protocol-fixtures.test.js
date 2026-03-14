const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')

const Ajv2020 = require('ajv/dist/2020').default
const addFormats = require('ajv-formats')

const repoRoot = path.resolve(__dirname, '..', '..')
const schemaRoot = path.join(repoRoot, 'protocol/schemas')
const fixtureRoot = path.join(repoRoot, 'protocol/fixtures')

function loadJson(filePath) {
  // JSON.parse is intentional here because the MVP canonicalization profile rejects
  // numbers outside the shared Go/TS safe-integer range before hashing.
  return JSON.parse(fs.readFileSync(filePath, 'utf8'))
}

function loadCanonicalizationPayload(filePath) {
  const raw = fs.readFileSync(filePath, 'utf8')
  assertCanonicalNumberLexemes(raw, filePath)
  return JSON.parse(raw)
}

function assertCanonicalNumberLexemes(raw, filePath) {
  let index = 0
  let inString = false
  let escaped = false

  while (index < raw.length) {
    const char = raw[index]
    if (inString) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        inString = false
      }
      index += 1
      continue
    }

    if (char === '"') {
      inString = true
      index += 1
      continue
    }

    if (char === '-' || isDigit(char)) {
      const { token, nextIndex } = readJsonNumberToken(raw, index)
      if (token !== null) {
        if (token.includes('.') || token.includes('e') || token.includes('E')) {
          throw new Error(`${filePath} contains non-integer numeric literal ${token}`)
        }
        index = nextIndex
        continue
      }
    }

    index += 1
  }
}

function readJsonNumberToken(raw, start) {
  let index = start
  if (raw[index] === '-') {
    index += 1
  }
  if (index >= raw.length || !isDigit(raw[index])) {
    return { token: null, nextIndex: start + 1 }
  }

  if (raw[index] === '0') {
    index += 1
  } else {
    while (index < raw.length && isDigit(raw[index])) {
      index += 1
    }
  }

  if (raw[index] === '.') {
    index += 1
    while (index < raw.length && isDigit(raw[index])) {
      index += 1
    }
  }

  if (raw[index] === 'e' || raw[index] === 'E') {
    index += 1
    if (raw[index] === '+' || raw[index] === '-') {
      index += 1
    }
    while (index < raw.length && isDigit(raw[index])) {
      index += 1
    }
  }

  return { token: raw.slice(start, index), nextIndex: index }
}

function isDigit(char) {
  return char >= '0' && char <= '9'
}

function loadBundle() {
  const manifest = loadJson(path.join(schemaRoot, 'manifest.json'))
  // strict: false is required because the shared schemas carry custom metadata
  // such as x-data-class that Ajv does not recognize.
  const ajv = new Ajv2020({ allErrors: true, strict: false })
  addFormats(ajv)

  const schemaPathToId = new Map()
  for (const entry of manifest.schema_files) {
    const schema = loadJson(path.join(schemaRoot, entry.path))
    ajv.addSchema(schema)
    schemaPathToId.set(entry.path, schema.$id)
  }

  return { ajv, schemaPathToId }
}

function validatorForPath(bundle, schemaPath) {
  const schemaId = bundle.schemaPathToId.get(schemaPath)
  assert.ok(schemaId, `missing schema id for ${schemaPath}`)
  const validate = bundle.ajv.getSchema(schemaId)
  assert.ok(validate, `missing validator for ${schemaPath}`)
  return validate
}

function canonicalize(value) {
  if (value === null) {
    return 'null'
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false'
  }
  if (typeof value === 'string') {
    return JSON.stringify(value)
  }
  if (typeof value === 'number') {
    if (!Number.isSafeInteger(value)) {
      throw new Error(`number ${value} is outside the MVP canonicalization profile`)
    }
    return JSON.stringify(value)
  }
  if (Array.isArray(value)) {
    return `[${value.map(canonicalize).join(',')}]`
  }
  if (value && typeof value === 'object') {
    const keys = Object.keys(value).sort()
    for (const key of keys) {
      if (!isAsciiString(key)) {
        throw new Error(`object key ${JSON.stringify(key)} is outside the MVP ASCII-only canonicalization profile`)
      }
    }
    return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalize(value[key])}`).join(',')}}`
  }
  throw new Error(`unsupported JSON type ${typeof value}`)
}

function isAsciiString(text) {
  for (let index = 0; index < text.length; index += 1) {
    if (text.charCodeAt(index) > 0x7f) {
      return false
    }
  }
  return true
}

function sha256Hex(text) {
  return require('node:crypto').createHash('sha256').update(text, 'utf8').digest('hex')
}

function digestIdentity(digest) {
  return `${digest.hash_alg}:${digest.hash}`
}

function validateRuntimeInvariant(rule, fixture) {
  switch (rule) {
    case 'llm_request_unique_artifact_digests':
      return requireUniqueArtifactDigests(fixture.input_artifacts)
    case 'llm_request_unique_tool_identities':
      return requireUniqueToolIdentities(fixture.tool_allowlist)
    case 'llm_response_unique_output_artifact_digests':
      return requireUniqueArtifactDigests(fixture.output_artifacts)
    case 'llm_response_unique_tool_call_ids':
      return requireUniqueToolCallIds(fixture.proposed_tool_calls)
    default:
      throw new Error(`unknown runtime invariant rule ${rule}`)
  }
}

function requireUniqueArtifactDigests(artifacts) {
  const seen = new Set()
  for (const artifact of artifacts) {
    const identity = digestIdentity(artifact.digest)
    if (seen.has(identity)) {
      return new Error(`duplicate artifact digest ${identity}`)
    }
    seen.add(identity)
  }
  return null
}

function requireUniqueToolIdentities(tools) {
  const seen = new Set()
  for (const tool of tools) {
    const identity = `${tool.tool_name}|${tool.arguments_schema_id}|${tool.arguments_schema_version}`
    if (seen.has(identity)) {
      return new Error(`duplicate tool identity ${identity}`)
    }
    seen.add(identity)
  }
  return null
}

function requireUniqueToolCallIds(toolCalls = []) {
  const seen = new Set()
  for (const toolCall of toolCalls) {
    if (seen.has(toolCall.tool_call_id)) {
      return new Error(`duplicate tool_call_id ${toolCall.tool_call_id}`)
    }
    seen.add(toolCall.tool_call_id)
  }
  return null
}

function validateStreamSequence(events) {
  if (events.length === 0) {
    return new Error('stream sequence must contain at least one event')
  }

  const first = events[0]
  if (first.seq !== 1) {
    return new Error('first stream event must use seq=1')
  }
  if (events[events.length - 1].event_type !== 'response_terminal') {
    return new Error('stream must contain exactly one terminal event')
  }

  const streamId = first.stream_id
  const requestHash = digestIdentity(first.request_hash)
  let lastSeq = 0

  for (let index = 0; index < events.length; index += 1) {
    const event = events[index]
    if (event.stream_id !== streamId) {
      return new Error('stream_id must remain constant across a stream')
    }
    if (digestIdentity(event.request_hash) !== requestHash) {
      return new Error('request_hash must remain constant across a stream')
    }
    if (event.seq <= lastSeq) {
      return new Error('stream sequence numbers must be strictly monotonic')
    }
    if (index < events.length - 1 && event.event_type === 'response_terminal') {
      return new Error('response_terminal must be the final event in the stream')
    }
    lastSeq = event.seq
  }
  return null
}

test('schema fixtures validate against manifest-defined schemas', async (t) => {
  const bundle = loadBundle()
  const manifest = loadJson(path.join(fixtureRoot, 'manifest.json'))

  for (const entry of manifest.schema_fixtures) {
    await t.test(entry.id, () => {
      const fixture = loadJson(path.join(fixtureRoot, entry.fixture_path))
      const validate = validatorForPath(bundle, entry.schema_path)
      const valid = validate(fixture)
      assert.equal(valid, entry.expect_valid, `${entry.id} validation mismatch: ${JSON.stringify(validate.errors)}`)
    })
  }
})

test('stream sequence fixtures validate schema and runtime semantics', async (t) => {
  const bundle = loadBundle()
  const manifest = loadJson(path.join(fixtureRoot, 'manifest.json'))

  for (const entry of manifest.stream_sequence_fixtures) {
    await t.test(entry.id, () => {
      const events = loadJson(path.join(fixtureRoot, entry.fixture_path))
      const validate = validatorForPath(bundle, entry.event_schema_path)

      let schemaValid = true
      for (const event of events) {
        if (!validate(event)) {
          schemaValid = false
          break
        }
      }

      const runtimeError = validateStreamSequence(events)
      const valid = schemaValid && runtimeError === null
      assert.equal(valid, entry.expect_valid, `${entry.id} stream mismatch: ${runtimeError || JSON.stringify(validate.errors)}`)
    })
  }
})

test('runtime invariant fixtures fail closed identically in JS', async (t) => {
  const bundle = loadBundle()
  const manifest = loadJson(path.join(fixtureRoot, 'manifest.json'))

  for (const entry of manifest.runtime_invariant_fixtures) {
    await t.test(entry.id, () => {
      const fixture = loadJson(path.join(fixtureRoot, entry.fixture_path))
      const validate = validatorForPath(bundle, entry.schema_path)
      assert.equal(validate(fixture), true, `${entry.id} must be schema-valid before runtime checks: ${JSON.stringify(validate.errors)}`)
      const runtimeError = validateRuntimeInvariant(entry.rule, fixture)
      assert.equal(runtimeError === null, entry.expect_valid, `${entry.id} runtime mismatch: ${runtimeError}`)
    })
  }
})

test('canonicalization fixtures match golden bytes and hashes', async (t) => {
  const manifest = loadJson(path.join(fixtureRoot, 'manifest.json'))

  for (const entry of manifest.canonicalization_fixtures) {
    await t.test(entry.id, () => {
      if (!entry.expect_valid) {
        assert.throws(() => canonicalize(loadCanonicalizationPayload(path.join(fixtureRoot, entry.payload_path))))
        return
      }

      const payload = loadCanonicalizationPayload(path.join(fixtureRoot, entry.payload_path))
      const golden = fs.readFileSync(path.join(fixtureRoot, entry.canonical_json_path), 'utf8').replace(/\n$/, '')
      const canonical = canonicalize(payload)

      assert.equal(canonical, golden)
      assert.equal(sha256Hex(canonical), entry.sha256)
    })
  }
})
