/**
 * Bounded deterministic runner-local identifiers.
 *
 * These helpers keep runner-generated attempt and request identifiers within
 * protocol schema limits while preserving stable plan-scoped derivation.
 */

import { createHash } from "node:crypto";
import type { RunnerPlanEntry } from "./run-plan.ts";

const MAX_IDENTIFIER_LENGTH = 128;

export function boundedAttemptID(prefix: string, planID: string, scopeID: string, attemptIndex: number): string {
  const digest = createHash("sha256")
    .update(planID)
    .update("\n")
    .update(scopeID)
    .digest("hex");
  const token = idToken(scopeID);
  const suffix = `${digest}_${attemptIndex}`;
  const maxTokenLength = MAX_IDENTIFIER_LENGTH - prefix.length - suffix.length - 2;
  const boundedToken = maxTokenLength > 0 && token.length > maxTokenLength ? token.slice(0, maxTokenLength) : token;
  return `${prefix}_${boundedToken}_${suffix}`;
}

export function boundedReportRequestID(kind: "checkpoint" | "result", runID: string, entry: RunnerPlanEntry, index: number): string {
  const digest = createHash("sha256")
    .update(kind)
    .update("\n")
    .update(runID)
    .update("\n")
    .update(entry.entry_id)
    .update("\n")
    .update(String(index))
    .digest("hex");
  return `runner-${kind}:${digest}`;
}

function idToken(value: string): string {
  const trimmed = value.trim().toLowerCase();
  if (!trimmed) {
    return "scope";
  }
  const normalized = trimmed.replace(/[^a-z0-9_-]/g, "_").replace(/^[_-]+|[_-]+$/g, "");
  if (!normalized) {
    return "scope";
  }
  return /^[a-z]/.test(normalized) ? normalized : `s_${normalized}`;
}
