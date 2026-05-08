const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const test = require("node:test");

function expectedAttemptID(prefix, planID, scopeID, attemptIndex, token) {
  const digest = createHash("sha256")
    .update(planID)
    .update("\n")
    .update(scopeID)
    .digest("hex");
  return `${prefix}_${token}_${digest}_${attemptIndex}`;
}

test("boundedAttemptID trims repeated separators around normalized content", async () => {
  const { boundedAttemptID } = await import("../src/runner-identifiers.ts");
  const scopeID = `${"_-".repeat(2048)}Scope_ID${"-_".repeat(2048)}`;

  assert.equal(
    boundedAttemptID("step_attempt", "plan_alpha", scopeID, 1),
    expectedAttemptID("step_attempt", "plan_alpha", scopeID, 1, "scope_id"),
  );
});

test("boundedAttemptID preserves s_ prefix when normalized token starts with a digit", async () => {
  const { boundedAttemptID } = await import("../src/runner-identifiers.ts");
  const scopeID = "___9-lives___";

  assert.equal(
    boundedAttemptID("step_attempt", "plan_alpha", scopeID, 1),
    expectedAttemptID("step_attempt", "plan_alpha", scopeID, 1, "s_9-lives"),
  );
});

test("boundedAttemptID falls back to scope when normalization removes all content", async () => {
  const { boundedAttemptID } = await import("../src/runner-identifiers.ts");
  const scopeID = `${"_-".repeat(2048)}!!!${"-_".repeat(2048)}`;

  assert.equal(
    boundedAttemptID("step_attempt", "plan_alpha", scopeID, 1),
    expectedAttemptID("step_attempt", "plan_alpha", scopeID, 1, "scope"),
  );
});

test("boundedAttemptID always stays within the 128 character schema limit", async () => {
  const { boundedAttemptID } = await import("../src/runner-identifiers.ts");
  const scopeID = `${"scope-".repeat(4096)}tail`;
  const attemptID = boundedAttemptID("step_attempt", "plan_alpha", scopeID, 1234567890);

  assert.ok(attemptID.length <= 128, `attempt id length ${attemptID.length} exceeded 128`);
  assert.match(attemptID, /^step_attempt_[a-z0-9_-]+_[0-9a-f]{64}_1234567890$/);
});

test("boundedAttemptID falls back to scope when prefix and suffix leave no token budget", async () => {
  const { boundedAttemptID } = await import("../src/runner-identifiers.ts");
  const prefix = `prefix_${"x".repeat(120)}`;
  const scopeID = "scope_id";

  assert.equal(
    boundedAttemptID(prefix, "plan_alpha", scopeID, 1),
    expectedAttemptID(prefix, "plan_alpha", scopeID, 1, "scope"),
  );
});
