Integration tests for the queryengine package should be used to exercise the paths that connect to and query the underlying data sources.

Integration tests should:

1. The tests should be runnable without requiring out-of-band setup of additional external infrastructure dependencies (other than Docker)
2. The tests should use testcontainers for isolation and reproducability across different environments such as local dev and CI

Scope:

- For now, focus on adding integration tests for Postgres and MySQL, where there currently is a gap in test coverage. Leave S3 out of scope for now - but we may layer this in later.
- Later on, it might be valuable to consider using the same patterns to add end to end tests for the cli.
