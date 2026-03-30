# Changelog

## [Unreleased]

### Changed

- Upgraded `github.com/lib/pq` to v1.12.1. **PostgreSQL 14 or later is now required** for consumers that register the `lib/pq` driver for `database/sql` in their test suites. This does not affect japi-core's primary database interface (pgx/v5).
