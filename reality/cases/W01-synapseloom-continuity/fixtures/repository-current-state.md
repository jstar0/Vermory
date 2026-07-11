# Repository-grounded current state

At repository revision 793f4c72a20f6bdb18a0f29d1a18794445ebde62:

- The frontend development port is 5173.
- The API Gateway default port is 8081 and business routes are under `/v1`.
- `make dev-up` starts only the Go API Gateway and Go LLM worker.
- `make dev-up` does not start PostgreSQL, Redis, the document worker, or the frontend.
- Phase plans are not long-term truth. The current overview and runbooks are the authority for a new coding session.
- The quota ledger has already moved beyond direct balance mutation and has a unified ledger table.

This excerpt is minimized from the repository overview documents. No environment file or private runtime value is included.
