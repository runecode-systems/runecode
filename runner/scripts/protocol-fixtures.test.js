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

function requireDigestObject(value, location) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return new Error(`${location} must be an object`)
  }
  if (typeof value.hash_alg !== 'string' || value.hash_alg.length === 0) {
    return new Error(`${location}.hash_alg must be a non-empty string`)
  }
  if (typeof value.hash !== 'string' || value.hash.length === 0) {
    return new Error(`${location}.hash must be a non-empty string`)
  }
  return value
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
    case 'audit_receipt_import_restore_byte_identity':
      return requireImportRestoreReceiptByteIdentity(fixture)
    case 'session_send_message_ack_alignment':
      return requireSessionSendMessageAckAlignment(fixture)
    case 'external_anchor_evidence_conformance':
      return requireExternalAnchorEvidenceConformance(fixture)
    default:
      throw new Error(`unknown runtime invariant rule ${rule}`)
  }
}

function requireExternalAnchorEvidenceConformance(envelope) {
  const payloadSchemaId = 'runecode.protocol.v0.ExternalAnchorEvidence'
  const payloadSchemaVersion = '0.1.0'
  if (envelope.payload_schema_id !== payloadSchemaId || envelope.payload_schema_version !== payloadSchemaVersion) {
    return new Error(`payload must be ${payloadSchemaId}@${payloadSchemaVersion}`)
  }

  const payload = requireObjectField(envelope, 'payload')
  if (payload instanceof Error) {
    return payload
  }

  const canonicalTargetDigest = requireDigestObject(payload.canonical_target_digest, 'payload.canonical_target_digest')
  if (canonicalTargetDigest instanceof Error) {
    return canonicalTargetDigest
  }
  const canonicalTargetIdentity = requireNonEmptyString(payload, 'canonical_target_identity', 'payload')
  if (canonicalTargetIdentity instanceof Error) {
    return canonicalTargetIdentity
  }
  if (digestIdentity(canonicalTargetDigest) !== canonicalTargetIdentity) {
    return new Error('target identity mismatch: canonical_target_identity does not match canonical_target_digest')
  }

  const outcome = requireNonEmptyString(payload, 'outcome', 'payload')
  if (outcome instanceof Error) {
    return outcome
  }
  if (!new Set(['completed', 'deferred', 'unavailable', 'invalid', 'failed']).has(outcome)) {
    return new Error(`unsupported external anchor outcome ${JSON.stringify(outcome)}`)
  }

  if (!Array.isArray(payload.sidecar_refs)) {
    return new Error('payload.sidecar_refs must be an array')
  }
  const byKind = new Map()
  for (let index = 0; index < payload.sidecar_refs.length; index += 1) {
    const ref = payload.sidecar_refs[index]
    if (!ref || typeof ref !== 'object' || Array.isArray(ref)) {
      return new Error(`payload.sidecar_refs[${index}] must be an object`)
    }
    const kind = requireNonEmptyString(ref, 'evidence_kind', `payload.sidecar_refs[${index}]`)
    if (kind instanceof Error) {
      return kind
    }
    const digest = requireDigestObject(ref.digest, `payload.sidecar_refs[${index}].digest`)
    if (digest instanceof Error) {
      return digest
    }
    if (byKind.has(kind)) {
      return new Error(`payload.sidecar_refs includes duplicate evidence_kind ${JSON.stringify(kind)}`)
    }
    byKind.set(kind, digestIdentity(digest))
  }
  if (!byKind.has('proof_bytes')) {
    return new Error('sidecar_refs must include proof_bytes')
  }

  const receiptTargetDescriptorDigest = optionalDigestIdentity(payload, 'receipt_target_descriptor_digest', 'payload')
  if (receiptTargetDescriptorDigest instanceof Error) {
    return receiptTargetDescriptorDigest
  }
  if (receiptTargetDescriptorDigest !== null && receiptTargetDescriptorDigest !== canonicalTargetIdentity) {
    return new Error('target identity mismatch: receipt_target_descriptor_digest does not match canonical_target_digest')
  }

  const receiptProofDigest = optionalDigestIdentity(payload, 'receipt_proof_digest', 'payload')
  if (receiptProofDigest instanceof Error) {
    return receiptProofDigest
  }
  if (receiptProofDigest !== null && receiptProofDigest !== byKind.get('proof_bytes')) {
    return new Error('invalid target proof: receipt_proof_digest does not match sidecar proof_bytes digest')
  }

  for (const [actualKey, expectedKey] of [
    ['typed_request_hash', 'expected_typed_request_hash'],
    ['action_request_hash', 'expected_action_request_hash'],
    ['approval_request_hash', 'expected_approval_request_hash'],
    ['approval_decision_hash', 'expected_approval_decision_hash'],
  ]) {
    const expected = optionalDigestIdentity(payload, expectedKey, 'payload')
    if (expected instanceof Error) {
      return expected
    }
    if (expected === null) {
      continue
    }
    const actual = optionalDigestIdentity(payload, actualKey, 'payload')
    if (actual instanceof Error) {
      return actual
    }
    if (actual === null || actual !== expected) {
      return new Error(`exact-action binding mismatch: ${actualKey} does not match ${expectedKey}`)
    }
  }

  return null
}

function optionalDigestIdentity(value, key, prefix = '') {
  if (!Object.prototype.hasOwnProperty.call(value, key)) {
    return null
  }
  const digest = requireDigestObject(value[key], formatFieldPath(prefix, key))
  if (digest instanceof Error) {
    return digest
  }
  return digestIdentity(digest)
}

function requireSessionSendMessageAckAlignment(fixture) {
	const sessionId = requireNonEmptyString(fixture, 'session_id')
	if (sessionId instanceof Error) {
		return sessionId
	}
	if (fixture.event_type !== 'session_message_ack') {
		return new Error('event_type must be session_message_ack')
	}
	if (fixture.stream_id !== `session-${sessionId}`) {
		return new Error(`stream_id ${JSON.stringify(fixture.stream_id)} must equal session-${sessionId}`)
	}
	const message = requireObjectField(fixture, 'message')
	if (message instanceof Error) {
		return message
	}
	const turn = requireObjectField(fixture, 'turn')
	if (turn instanceof Error) {
		return turn
	}
	const messageSessionId = requireNonEmptyString(message, 'session_id', 'message')
	if (messageSessionId instanceof Error) {
		return messageSessionId
	}
	if (messageSessionId !== sessionId) {
		return new Error(`message.session_id ${JSON.stringify(messageSessionId)} must match session_id ${JSON.stringify(sessionId)}`)
	}
	const turnSessionId = requireNonEmptyString(turn, 'session_id', 'turn')
	if (turnSessionId instanceof Error) {
		return turnSessionId
	}
	if (turnSessionId !== sessionId) {
		return new Error(`turn.session_id ${JSON.stringify(turnSessionId)} must match session_id ${JSON.stringify(sessionId)}`)
	}
	const turnId = requireNonEmptyString(turn, 'turn_id', 'turn')
	if (turnId instanceof Error) {
		return turnId
	}
	const messageTurnId = requireNonEmptyString(message, 'turn_id', 'message')
	if (messageTurnId instanceof Error) {
		return messageTurnId
	}
	if (messageTurnId !== turnId) {
		return new Error(`message.turn_id ${JSON.stringify(messageTurnId)} must match turn.turn_id ${JSON.stringify(turnId)}`)
	}
	if (!Number.isInteger(fixture.seq) || fixture.seq < 1) {
		return new Error('seq must be >= 1')
	}
	return null
}

function requireObjectField(value, key) {
	const object = value?.[key]
	if (!object || typeof object !== 'object' || Array.isArray(object)) {
		return new Error(`${key} must be an object`)
	}
	return object
}

function requireNonEmptyString(value, key, prefix = '') {
	const fieldValue = value?.[key]
	if (typeof fieldValue !== 'string' || fieldValue.trim().length === 0) {
		return new Error(`${formatFieldPath(prefix, key)} must be a non-empty string`)
	}
	return fieldValue
}

function formatFieldPath(prefix, key) {
	return prefix ? `${prefix}.${key}` : key
}

function requireImportRestoreReceiptByteIdentity(receipt) {
	if (receipt.audit_receipt_kind !== 'import' && receipt.audit_receipt_kind !== 'restore') {
		return null
	}

	if (receipt.receipt_payload_schema_id !== 'runecode.protocol.audit.receipt.import_restore_provenance.v0') {
		return new Error('import/restore receipts must use import_restore provenance payload schema')
	}

	if (!receipt.receipt_payload || typeof receipt.receipt_payload !== 'object' || Array.isArray(receipt.receipt_payload)) {
		return new Error('receipt_payload must be an object')
	}

	if (receipt.receipt_payload.provenance_action !== receipt.audit_receipt_kind) {
		return new Error(`provenance_action ${JSON.stringify(receipt.receipt_payload.provenance_action)} must match audit_receipt_kind ${JSON.stringify(receipt.audit_receipt_kind)}`)
	}

	if (!Array.isArray(receipt.receipt_payload.imported_segments)) {
		return new Error('receipt_payload.imported_segments must be an array')
	}

	for (let index = 0; index < receipt.receipt_payload.imported_segments.length; index += 1) {
		const segment = receipt.receipt_payload.imported_segments[index]
		if (!segment || typeof segment !== 'object' || Array.isArray(segment)) {
			return new Error(`receipt_payload.imported_segments[${index}] must be an object`)
		}
		if (segment.byte_identity_verified !== true) {
			return new Error(`receipt_payload.imported_segments[${index}].byte_identity_verified must be true`)
		}
		const sourceDigest = requireDigestObject(segment.source_segment_file_hash, `receipt_payload.imported_segments[${index}].source_segment_file_hash`)
		if (sourceDigest instanceof Error) {
			return sourceDigest
		}
		const localDigest = requireDigestObject(segment.local_segment_file_hash, `receipt_payload.imported_segments[${index}].local_segment_file_hash`)
		if (localDigest instanceof Error) {
			return localDigest
		}
		const source = digestIdentity(sourceDigest)
		const local = digestIdentity(localDigest)
		if (source !== local) {
			return new Error(`receipt_payload.imported_segments[${index}] source/local segment hashes differ`)
		}
	}

	return null
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
  if (!isTerminalEventType(events[events.length - 1].event_type)) {
    return new Error('stream must contain exactly one terminal event')
  }

  const streamId = first.stream_id
  const requestIdentity = streamRequestIdentity(first)
  if (requestIdentity instanceof Error) {
    return requestIdentity
  }
  let lastSeq = 0

  for (let index = 0; index < events.length; index += 1) {
    const event = events[index]
    if (event.stream_id !== streamId) {
      return new Error('stream_id must remain constant across a stream')
    }
    const eventIdentity = streamRequestIdentity(event)
    if (eventIdentity instanceof Error) {
      return eventIdentity
    }
    if (eventIdentity !== requestIdentity) {
      return new Error('request identity must remain constant across a stream')
    }
    if (event.seq <= lastSeq) {
      return new Error('stream sequence numbers must be strictly monotonic')
    }
    if (index < events.length - 1 && isTerminalEventType(event.event_type)) {
      return new Error('terminal event must be the final event in the stream')
    }
    lastSeq = event.seq
  }
  return null
}

function isTerminalEventType(eventType) {
  return eventType === 'response_terminal' || (typeof eventType === 'string' && eventType.endsWith('_terminal'))
}

function streamRequestIdentity(event) {
  if (Object.prototype.hasOwnProperty.call(event, 'request_hash')) {
    const digest = requireDigestObject(event.request_hash, 'request_hash')
    if (digest instanceof Error) {
      return digest
    }
    return digestIdentity(digest)
  }
  if (typeof event.request_id === 'string' && event.request_id.length > 0) {
    return `request_id:${event.request_id}`
  }
  return new Error('stream event must include request_hash or request_id')
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

test('import/restore runtime invariant returns structured error for malformed digest objects', () => {
	const runtimeError = requireImportRestoreReceiptByteIdentity({
		audit_receipt_kind: 'import',
		receipt_payload_schema_id: 'runecode.protocol.audit.receipt.import_restore_provenance.v0',
		receipt_payload: {
			provenance_action: 'import',
			imported_segments: [
				{
					byte_identity_verified: true,
					source_segment_file_hash: 'not-an-object',
					local_segment_file_hash: { hash_alg: 'sha256', hash: 'abcd' },
				},
			],
		},
	})
	assert.ok(runtimeError instanceof Error)
	assert.match(runtimeError.message, /source_segment_file_hash must be an object/)
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
