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
  const boundedToken = boundedIdentifierToken(token, maxTokenLength);
  return `${prefix}_${boundedToken}_${suffix}`;
}

function boundedIdentifierToken(token: string, maxTokenLength: number): string {
  if (maxTokenLength <= 0) {
    return "scope";
  }
  if (token.length <= maxTokenLength) {
    return token;
  }
  return token.slice(0, maxTokenLength);
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
  let normalized = "";
  for (const character of trimmed) {
    normalized += isIdentifierTokenCharacter(character) ? character : "_";
  }
  normalized = trimIdentifierTokenSeparators(normalized);
  if (!normalized) {
    return "scope";
  }
  return startsWithASCIILowercase(normalized) ? normalized : `s_${normalized}`;
}

function isIdentifierTokenCharacter(character: string): boolean {
  return startsWithASCIILowercase(character) || isASCIIDigit(character) || character === "_" || character === "-";
}

function trimIdentifierTokenSeparators(value: string): string {
  let start = 0;
  let end = value.length;

  while (start < end && isIdentifierSeparator(value.charAt(start))) {
    start += 1;
  }
  while (end > start && isIdentifierSeparator(value.charAt(end - 1))) {
    end -= 1;
  }

  return start === 0 && end === value.length ? value : value.slice(start, end);
}

function startsWithASCIILowercase(value: string): boolean {
  if (value.length === 0) {
    return false;
  }
  const code = value.charCodeAt(0);
  return code >= 97 && code <= 122;
}

function isASCIIDigit(value: string): boolean {
  if (value.length === 0) {
    return false;
  }
  const code = value.charCodeAt(0);
  return code >= 48 && code <= 57;
}

function isIdentifierSeparator(character: string): boolean {
  return character === "_" || character === "-";
}
