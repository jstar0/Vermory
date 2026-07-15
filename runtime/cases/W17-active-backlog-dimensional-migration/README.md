# W17 Active-Backlog Dimensional Migration

W17 qualifies coexistence of the active 1024-dimensional retrieval projection
and a candidate 512-dimensional projection while PostgreSQL authority events
continue arriving.

The case creates 20,000 initial active governed facts across four tenants,
starts the candidate snapshot, commits 2,000 revisions, 500 deletions, and 500
new facts during the snapshot, and drains both profile-specific tails. It
injects a PostgreSQL immediate restart during candidate embedding, then proves
same-pool recovery, deletion safety, candidate reset/rebuild isolation, and
zero final lag for both physical projection classes.

Deterministic 1024- and 512-dimensional embeddings qualify storage, worker,
cursor, lifecycle, restart, and scope mechanics. A separate small tenant uses
the direct SiliconFlow OpenAI-compatible embeddings endpoint with
`BAAI/bge-small-zh-v1.5` and must return exactly 512 dimensions before the
formal profile is accepted.

This case does not rank embedding models, promote a semantic profile, claim
long-duration retention, or qualify cross-host HA.
