# Reflection - 2026-02-20

## Scope
Consolidated diary learnings from `.agents/diary/` and hardened project operating rules.

## Repeated Patterns
- Ambiguous requirements occasionally led to edits before direction was explicitly confirmed.
- Patch application sometimes used indirect shell wrapping instead of native patch tooling.
- Behavior changelog notes were sometimes implicit and not explicit about newly added metadata fields.
- `make vet` was not always executed early enough after API/signature changes.
- Documentation updates occasionally drifted to locations not explicitly requested by the user.

## Hardening Applied
Added session rules in `AGENTS.md` to enforce:
- direction confirmation before edits under ambiguity
- direct native `apply_patch` usage for patch edits
- explicit enumeration of user-visible metadata in changelog behavior notes
- early `make vet` after structural/API edits
- strict adherence to user-specified documentation locations

## Outcome
Project memory now contains stricter, concrete process controls focused on reducing rework and improving auditability.
