package main

const brokerHelpText = `Usage: runecode-broker [--state-root path] [--audit-ledger-root path] [--runtime-dir dir] [--socket-name broker.sock] <command> [flags]

Global options:
  --state-root path         broker state root (artifact store and broker-owned local state)
  --audit-ledger-root path  audit ledger root
  --runtime-dir dir         explicit runtime directory for live IPC commands
  --socket-name name        explicit socket filename for live IPC commands

Commands:
  serve-local [--runtime-dir dir] [--socket-name broker.sock] [--once]
  run-list [--limit N]
  run-get --run-id id
  run-watch [--stream-id id] [--run-id id] [--workspace-id id] [--lifecycle-state state] [--follow] [--include-snapshot]
  backend-posture-get
  backend-posture-change --target-backend-kind microvm|container [--target-instance-id id] [--selection-mode explicit_selection|automatic_fallback_attempt] [--change-kind select_backend] [--assurance-change-kind reduce_assurance|maintain_assurance] [--opt-in-kind exact_action_approval|none] [--reduced-assurance-acknowledged] [--reason text]
  session-list [--limit N]
  session-get --session-id id
  session-send-message --session-id id --content text [--role user|assistant|system|tool] [--idempotency-key key]
  session-execution-trigger --session-id id [--turn-id id] [--trigger-source interactive_user|autonomous_background|resume_follow_up] [--requested-operation start|continue] [--workflow-family runecontext] [--workflow-operation change_draft|spec_draft|draft_promote_apply|approved_change_implementation] [--user-message text] [--idempotency-key key]
  session-watch [--stream-id id] [--session-id id] [--workspace-id id] [--status active|completed|archived] [--last-activity-kind kind] [--follow] [--include-snapshot]
  approval-list [--run-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--limit N]
  approval-get --approval-id sha256:...
  approval-resolve --approval-request approval-request.json --approval-envelope approval.json [--approval-id sha256:...]
  approval-watch [--stream-id id] [--approval-id sha256:...] [--run-id id] [--workspace-id id] [--status pending|approved|denied|expired|cancelled|superseded|consumed] [--follow] [--include-snapshot]
  list-artifacts
  head-artifact --digest sha256:...
  get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path
  put-artifact --file path --content-type type --data-class class --provenance-hash sha256:... [--runtime-dir dir] [--socket-name broker.sock]
  check-flow --producer role --consumer role --data-class class --digest sha256:... [--egress] [--manifest-opt-in]
  promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible
  revoke-approved-excerpt --digest sha256:... --actor user
  set-run-status --run-id id --status active|retained|closed
  gc
  export-backup --path backup.json (artifact/broker state backup; trusted-context audit import links are not made portable by this command)
  restore-backup --path backup.json (restores artifact/broker state only; re-import trusted contracts/evidence in the target environment as needed)
  show-audit
  show-policy
  set-reserved-classes --enabled=true|false
	  import-trusted-contract --kind <kind> --file payload.json --evidence import-evidence.json
  seed-dev-manual-scenario --dev-only [--profile tui-rich-v1] (requires dev-seed build tag; seeds trusted context using the same import/audit semantics as other trusted policy artifacts)
	  audit-readiness
	  audit-verification [--limit N]
	  audit-finalize-verify
	  audit-record-get --record-digest sha256:...
	  audit-record-inclusion-get --record-digest sha256:...
	  audit-evidence-snapshot-get
	  audit-evidence-retention-review --request-file path
	  audit-evidence-bundle-manifest-get --request-file path [--external-sharing]
	  audit-evidence-bundle-export --request-file path --out path (streaming tar export with start/chunk/terminal events)
	  audit-evidence-bundle-offline-verify --bundle path [--archive-format tar]
	  audit-anchor-segment --seal-digest sha256:... [--approval-decision-digest sha256:...] [--approval-assurance-level level] [--export-receipt-copy]
	  git-setup-get [--provider github]
	  git-setup-auth-bootstrap [--provider github] --mode browser|device_code
	  git-setup-identity-upsert [--provider github] --profile-id id --display-name name --author-name name --author-email mail --committer-name name --committer-email mail --signoff-name name --signoff-email mail [--default-profile]
	  provider-setup-direct --provider-family family --canonical-host host [--canonical-path-prefix /v1] [--display-label label] [--adapter-kind kind] [--allowlisted-model-ids id1,id2]
	  provider-credential-lease-issue --provider-profile-id id --run-id run-1 [--ttl-seconds 900]
	  provider-profile-list
	  provider-profile-get --provider-profile-id id
	  dependency-cache-ensure --request-file path
	  dependency-fetch-registry --request-file path
	  dependency-cache-handoff --request-file path
	  project-substrate-get
	  project-substrate-posture-get
	  project-substrate-adopt
	  project-substrate-init-preview
	  project-substrate-init-apply [--expected-preview-token sha256:...]
	  project-substrate-upgrade-preview
	  project-substrate-upgrade-apply --expected-preview-digest sha256:...
	  git-remote-mutation-prepare --request-file path
	  git-remote-mutation-get --request-file path
	  git-remote-mutation-issue-execute-lease --request-file path
	  git-remote-mutation-execute --request-file path
	  external-anchor-mutation-prepare --request-file path
	  external-anchor-mutation-get --request-file path
	  external-anchor-mutation-issue-execute-lease --request-file path
	  external-anchor-mutation-execute --request-file path
  version-info
  stream-logs [--stream-id id] [--run-id id] [--role-instance-id id] [--start-cursor cursor] [--follow] [--include-backlog]
  llm-invoke --run-id id --request-file path [--request-digest sha256:...]
  llm-stream --run-id id --request-file path [--request-digest sha256:...] [--stream-id id] [--follow]`
