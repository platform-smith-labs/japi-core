---
type: interface
title: "Schema: controller"
tags: [schema, postgres, controller]
timestamp: 2026-07-07T01:02:42Z
description: "Final-state reference for controller and controller_instance tables"
repo: db-migration
commit_sha: 455ca0a
evidence:
  - migrations/0003_controllers.sql
  - migrations/0001_enums.sql
provides_interfaces:
  - {name: "controller tables", kind: postgres-schema, intent: "controller registration and per-process instance/connection tracking"}
---

# Schema: controller domain

### controller

Logical controller definitions (one per controller name).

| column | type | null | default |
|---|---|---|---|
| controller_id | SERIAL (PK) | NOT NULL | auto |
| controller_uuid | UUID | NOT NULL | gen_random_uuid() |
| company_id | INTEGER | NOT NULL | — |
| name | TEXT | NOT NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |

**Constraints:**
- PK: `controller_id`
- UNIQUE: `controller_uuid`
- FK: `company_id` → `company(company_id)`
- UNIQUE: `(company_id, name)` — controller names unique per tenant
- UNIQUE: `(company_id, controller_id)` — composite-FK target for child tables

**Indexes:**
- `idx_controller_company_id` on `(company_id)`

### controller_instance

Each instance of a controller process; supports reconnection tracking via `instance_uuid` (a UUID the controller process generates on startup, distinguishing reconnections from restarts).

| column | type | null | default |
|---|---|---|---|
| controller_instance_id | SERIAL (PK) | NOT NULL | auto |
| controller_instance_uuid | UUID | NOT NULL | gen_random_uuid() |
| instance_uuid | UUID | NOT NULL | — |
| company_id | INTEGER | NOT NULL | — |
| controller_id | INTEGER | NOT NULL | — |
| connected | BOOLEAN | NOT NULL | TRUE |
| first_connected_at | TIMESTAMPTZ | NOT NULL | NOW() |
| last_seen | TIMESTAMPTZ | NOT NULL | NOW() |
| disconnected_at | TIMESTAMPTZ | NULL | — |
| remote_addr | TEXT | NULL | — |
| created_at | TIMESTAMPTZ | NOT NULL | NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL | NOW() |
| workspace_token_id | INTEGER | NULL | — |
| workspace_id | INTEGER | NULL | — |
| version | TEXT | NULL | — |

Column notes: `controller_instance_uuid` is the DB-generated UUID for API responses (distinct from the process-generated `instance_uuid`). `workspace_token_id` records which workspace token authenticated this controller connection; `workspace_id` records the workspace the instance belongs to. `version` is the controller software version string (e.g. "0.4.3") reported on WebSocket connect — NULL for instances that have not re-registered.

**Constraints:**
- PK: `controller_instance_id`
- UNIQUE: `controller_instance_uuid`
- FK: `company_id` → `company(company_id)`
- FK (composite): `(company_id, controller_id)` → `controller(company_id, controller_id)`
- FK (composite): `(company_id, workspace_token_id)` → `workspace_token(company_id, workspace_token_id)`
- FK (composite): `(company_id, workspace_id)` → `workspace(company_id, workspace_id)`
- UNIQUE: `(company_id, instance_uuid)`
- UNIQUE: `(company_id, controller_instance_id)` — composite-FK target for child tables

**Indexes:**
- `idx_controller_instance_company_id` on `(company_id)`
- `idx_controller_instance_company_controller` on `(company_id, controller_id)`
- `idx_controller_instance_connected` on `(company_id, connected)` WHERE `connected = TRUE` (partial)
- `idx_controller_instance_workspace_id` on `(company_id, workspace_id)` WHERE `workspace_id IS NOT NULL` (partial)

## ENUM types

None — neither table uses any PostgreSQL ENUM type (`connected` is a plain BOOLEAN).
