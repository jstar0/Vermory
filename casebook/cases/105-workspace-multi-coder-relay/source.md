# Workspace Multi Coder Relay

`invoice-ops` is a workspace shared by Codex, Claude Code, and Gemini CLI.

Codex finished the ingestion schema. Claude Code is responsible for the reconciliation UI. Gemini CLI is only being used for SQL explanation, not for editing source files.

The next relay should go to Claude Code for `frontend/src/pages/Reconciliation.tsx`.

The backend contract is frozen for this sprint; changing API names would break the ingestion tests.

