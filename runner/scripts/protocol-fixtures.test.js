const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')

const Ajv2020 = require('ajv/dist/2020').default
const addFormats = require('ajv-formats')
const crypto = require('node:crypto')

const repoRoot = path.resolve(__dirname, '..', '..')
const schemaRoot = path.join(repoRoot, 'protocol/schemas')
const fixtureRoot = path.join(repoRoot, 'protocol/fixtures')

function loadJson(filePath) {
	return JSON.parse(fs.readFileSync(filePath, 'utf8'))
}

function loadBundle() {
  const manifest = loadJson(path.join(schemaRoot, 'manifest.json'))
  // strict: false is required because the shared schemas carry custom metadata
  // such as x-data-class that Ajv does not recognize.
  const ajv = new Ajv2020({ allErrors: true, strict: false })
  addFormats(ajv)

  const schemaPathToId = new Map()
  const runtimeSchemaKeyToPath = new Map()
  for (const entry of manifest.schema_files) {
    const schema = loadJson(path.join(schemaRoot, entry.path))
    ajv.addSchema(schema)
    schemaPathToId.set(entry.path, schema.$id)
    runtimeSchemaKeyToPath.set(runtimeSchemaKey(entry.schema_id, entry.schema_version), entry.path)
  }

  return { ajv, schemaPathToId, runtimeSchemaKeyToPath }
}

function runtimeSchemaKey(schemaId, schemaVersion) {
  return `${schemaId}@${schemaVersion}`
}

function validatorForPath(bundle, schemaPath) {
  const schemaId = bundle.schemaPathToId.get(schemaPath)
  assert.ok(schemaId, `missing schema id for ${schemaPath}`)
  const validate = bundle.ajv.getSchema(schemaId)
  assert.ok(validate, `missing validator for ${schemaPath}`)
  return validate
}

function canonicalize(value) {
	return canonicalizeFromText(JSON.stringify(value))
}

function canonicalizeFromText(jsonText) {
	assertNoDuplicateObjectKeys(jsonText)
	const value = JSON.parse(jsonText)
	assert.ok(Array.isArray(value) || (value && typeof value === 'object'), 'top-level JSON value must be an object or array')
	return serializeCanonical(value)
}

function assertNoDuplicateObjectKeys(jsonText) {
	let index = 0

	function skipWhitespace() {
		while (index < jsonText.length && /\s/.test(jsonText[index])) {
			index += 1
		}
	}

	function parseStringToken() {
		const start = index
		assert.equal(jsonText[index], '"', 'expected string')
		index += 1
		while (index < jsonText.length) {
			const ch = jsonText[index]
			if (ch === '\\') {
				index += 2
				continue
			}
			if (ch === '"') {
				index += 1
				return jsonText.slice(start, index)
			}
			index += 1
		}
		throw new Error('unterminated string literal')
	}

	function parseLiteral(lit) {
		assert.equal(jsonText.slice(index, index + lit.length), lit, `expected literal ${lit}`)
		index += lit.length
	}

	function parseNumber() {
		const match = /^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/.exec(jsonText.slice(index))
		assert.ok(match, 'invalid number')
		index += match[0].length
	}

	function parseArray() {
		assert.equal(jsonText[index], '[', 'expected array')
		index += 1
		skipWhitespace()
		if (jsonText[index] === ']') {
			index += 1
			return
		}
		while (true) {
			parseValue()
			skipWhitespace()
			if (jsonText[index] === ']') {
				index += 1
				return
			}
			assert.equal(jsonText[index], ',', 'expected comma in array')
			index += 1
			skipWhitespace()
		}
	}

	function parseObject() {
		assert.equal(jsonText[index], '{', 'expected object')
		index += 1
		skipWhitespace()
		if (jsonText[index] === '}') {
			index += 1
			return
		}
		const keys = new Set()
		while (true) {
			const token = parseStringToken()
			const key = JSON.parse(token)
			if (keys.has(key)) {
				throw new Error(`duplicate object key ${JSON.stringify(key)}`)
			}
			keys.add(key)
			skipWhitespace()
			assert.equal(jsonText[index], ':', 'expected colon in object')
			index += 1
			skipWhitespace()
			parseValue()
			skipWhitespace()
			if (jsonText[index] === '}') {
				index += 1
				return
			}
			assert.equal(jsonText[index], ',', 'expected comma in object')
			index += 1
			skipWhitespace()
		}
	}

	function parseValue() {
		skipWhitespace()
		const ch = jsonText[index]
		if (ch === '{') {
			parseObject()
			return
		}
		if (ch === '[') {
			parseArray()
			return
		}
		if (ch === '"') {
			parseStringToken()
			return
		}
		if (ch === 't') {
			parseLiteral('true')
			return
		}
		if (ch === 'f') {
			parseLiteral('false')
			return
		}
		if (ch === 'n') {
			parseLiteral('null')
			return
		}
		parseNumber()
	}

	parseValue()
	skipWhitespace()
	assert.equal(index, jsonText.length, 'unexpected trailing JSON data')
}

function serializeCanonical(value) {
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
		assert.ok(Number.isFinite(value), `invalid JSON number ${value}`)
		if (Object.is(value, -0)) {
			return '0'
		}
		return value.toString()
	}
	if (Array.isArray(value)) {
		return `[${value.map(serializeCanonical).join(',')}]`
	}
	if (value && typeof value === 'object') {
		const keys = Object.keys(value).sort(compareUtf16)
		return `{${keys.map((key) => `${JSON.stringify(key)}:${serializeCanonical(value[key])}`).join(',')}}`
	}
	throw new Error(`unsupported JSON type ${typeof value}`)
}

function compareUtf16(left, right) {
	const limit = Math.min(left.length, right.length)
	for (let index = 0; index < limit; index += 1) {
		const diff = left.charCodeAt(index) - right.charCodeAt(index)
		if (diff !== 0) {
			return diff
		}
	}
	return left.length - right.length
}

function sha256Hex(text) {
	return crypto.createHash('sha256').update(text, 'utf8').digest('hex')
}

function digestIdentity(digest) {
  return `${digest.hash_alg}:${digest.hash}`
}

function validateRuntimeInvariant(rule, fixture, bundle) {
  switch (rule) {
    case 'llm_request_unique_artifact_digests':
      return requireUniqueArtifactDigests(fixture.input_artifacts)
    case 'llm_request_unique_tool_identities':
      return requireUniqueToolIdentities(fixture.tool_allowlist)
    case 'llm_response_unique_output_artifact_digests':
      return requireUniqueArtifactDigests(fixture.output_artifacts)
    case 'llm_response_unique_tool_call_ids':
      return requireUniqueToolCallIds(fixture.proposed_tool_calls)
    case 'signed_envelope_payload_schema_match':
      return requireSignedEnvelopePayloadSchemaMatch(fixture, bundle)
    default:
      throw new Error(`unknown runtime invariant rule ${rule}`)
  }
}

function requireSignedEnvelopePayloadSchemaMatch(envelope, bundle) {
  if (envelope.payload_schema_id !== envelope?.payload?.schema_id) {
    return new Error(`payload_schema_id ${JSON.stringify(envelope.payload_schema_id)} does not match payload.schema_id ${JSON.stringify(envelope?.payload?.schema_id)}`)
  }
  if (envelope.payload_schema_version !== envelope?.payload?.schema_version) {
    return new Error(`payload_schema_version ${JSON.stringify(envelope.payload_schema_version)} does not match payload.schema_version ${JSON.stringify(envelope?.payload?.schema_version)}`)
  }

  const payloadSchemaPath = bundle.runtimeSchemaKeyToPath.get(runtimeSchemaKey(envelope.payload_schema_id, envelope.payload_schema_version))
  if (!payloadSchemaPath) {
    return new Error(`payload schema ${envelope.payload_schema_id}@${envelope.payload_schema_version} not found in schema manifest`)
  }

  const validatePayload = validatorForPath(bundle, payloadSchemaPath)
  const validPayload = validatePayload(envelope.payload)
  if (!validPayload) {
    return new Error(`payload failed ${envelope.payload_schema_id}@${envelope.payload_schema_version} schema validation: ${JSON.stringify(validatePayload.errors)}`)
  }

  return null
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
      const runtimeError = validateRuntimeInvariant(entry.rule, fixture, bundle)
      assert.equal(runtimeError === null, entry.expect_valid, `${entry.id} runtime mismatch: ${runtimeError}`)
    })
  }
})

test('canonicalization fixtures match golden bytes and hashes', async (t) => {
  const manifest = loadJson(path.join(fixtureRoot, 'manifest.json'))

	for (const entry of manifest.canonicalization_fixtures) {
		await t.test(entry.id, () => {
			const raw = fs.readFileSync(path.join(fixtureRoot, entry.payload_path), 'utf8')
			if (!entry.expect_valid) {
				assert.throws(() => canonicalizeFromText(raw))
				return
			}

			const golden = fs.readFileSync(path.join(fixtureRoot, entry.canonical_json_path), 'utf8').replace(/\n$/, '')
			const canonical = canonicalizeFromText(raw)

			assert.equal(canonical, golden)
			assert.equal(sha256Hex(canonical), entry.sha256)
    })
	}
})

test('canonicalization rejects top-level scalar roots', () => {
	for (const payload of ['1', '"text"', 'true', 'null']) {
		assert.throws(() => canonicalizeFromText(payload), /top-level JSON value must be an object or array/)
	}
})

test('canonicalization rejects duplicate object keys', () => {
	assert.throws(() => canonicalizeFromText('{"a":1,"a":2}'), /duplicate object key/)
})
