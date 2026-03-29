# Dashboard Card IP Visibility Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an admin-configurable dashboard setting that controls whether homepage node cards display IP addresses while keeping guest mode, node tables, and node details unchanged.

**Architecture:** Add a small dashboard settings model persisted in `app_settings`, expose it through a new `/api/settings/dashboard` read/write pair, and wire both the Settings page and Dashboard page to the same typed frontend API. Reuse the existing `NodeCard.showIP` prop so the behavioral change stays localized to settings loading and prop plumbing rather than card rendering logic.

**Tech Stack:** Go, SQLite, React, TypeScript, Vitest, Testing Library.

---

### Task 1: Capture the current baseline for dashboard IP visibility

**Files:**
- Read: `frontend/src/components/NodeCard.tsx`
- Read: `frontend/src/pages/Dashboard.tsx`
- Read: `frontend/src/pages/dashboard-states.test.tsx`
- Read: `frontend/src/pages/Settings.tsx`
- Read: `frontend/src/pages/settings-states.test.tsx`
- Read: `internal/api/api.go`
- Read: `internal/api/api_test.go`
- Read: `internal/store/store.go`
- Read: `internal/store/store_test.go`

**Step 1: Run the current backend settings tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/store ./internal/api -run 'TestGetMetricsRetentionDefaultsToSevenDays|TestNotificationSettingsRoundTripEndpoints|TestStoreNotificationSettingsRoundTripAndCooldown'`

Expected: PASS on the current settings baseline.

**Step 2: Run the current frontend dashboard/settings tests**

Run:

`cd frontend && npm test -- node-card-redesign.test.tsx dashboard-states.test.tsx settings-states.test.tsx`

Expected: PASS on the current card/settings baseline.

**Step 3: Record the current behavior**

Confirm from code that:

- `NodeCard` already supports `showIP`
- `Dashboard` passes `showIP={accessMode !== "guest"}`
- no persisted dashboard display setting exists yet

### Task 2: Add a persisted dashboard settings model in the backend

**Files:**
- Create: `internal/models/dashboard_settings.go`
- Modify: `internal/store/store.go`
- Modify: `internal/store/store_test.go`

**Step 1: Write the failing store tests**

Add tests covering:

- default dashboard settings return `show_dashboard_card_ip = true`
- updated dashboard settings round-trip through store persistence

**Step 2: Run the focused store tests to verify failure**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/store -run 'TestStoreDashboardSettings'`

Expected: FAIL because dashboard settings storage does not exist yet.

**Step 3: Write the minimal implementation**

Add:

- a `DashboardSettings` model
- a new `app_settings` key for dashboard settings
- store helpers for defaulting, reading, normalizing, and upserting dashboard settings

**Step 4: Re-run the focused store tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/store -run 'TestStoreDashboardSettings'`

Expected: PASS.

### Task 3: Expose dashboard settings through the API

**Files:**
- Modify: `internal/api/api.go`
- Modify: `internal/api/api_test.go`

**Step 1: Write the failing API tests**

Add tests covering:

- `GET /api/settings/dashboard` returns the default `show_dashboard_card_ip: true`
- `PUT /api/settings/dashboard` persists `show_dashboard_card_ip: false`

**Step 2: Run the focused API tests to verify failure**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'TestDashboardSettings'`

Expected: FAIL because the routes and handlers do not exist yet.

**Step 3: Write the minimal implementation**

Add:

- viewer `GET /api/settings/dashboard`
- admin `PUT /api/settings/dashboard`
- request decoding and response writing aligned with other settings handlers

**Step 4: Re-run the focused API tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'TestDashboardSettings'`

Expected: PASS.

### Task 4: Add the typed frontend API and settings card

**Files:**
- Modify: `frontend/src/lib/api.ts`
- Create: `frontend/src/components/settings/DashboardVisibilityCard.tsx`
- Modify: `frontend/src/pages/Settings.tsx`
- Modify: `frontend/src/pages/settings-states.test.tsx`
- Modify: `frontend/src/i18n/messages.en.ts`
- Modify: `frontend/src/i18n/messages.zh-CN.ts`

**Step 1: Write the failing frontend settings test**

Add a test expecting:

- the new card to load dashboard settings
- the checkbox to reflect the backend value
- saving a changed value to call the new update API

**Step 2: Run the focused frontend settings test to verify failure**

Run:

`cd frontend && npm test -- settings-states.test.tsx`

Expected: FAIL because the new API methods and card do not exist yet.

**Step 3: Write the minimal implementation**

Add:

- typed dashboard settings methods to `frontend/src/lib/api.ts`
- a small settings card matching the existing settings shell
- i18n copy for title, description, toggle label, save CTA, and feedback
- render the card in `Settings`

**Step 4: Re-run the focused frontend settings test**

Run:

`cd frontend && npm test -- settings-states.test.tsx`

Expected: PASS.

### Task 5: Wire the dashboard page to the new setting

**Files:**
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/pages/dashboard-states.test.tsx`

**Step 1: Write the failing dashboard tests**

Add tests covering:

- admin dashboard hides IP when `show_dashboard_card_ip` is false
- admin dashboard shows IP when `show_dashboard_card_ip` is true
- guest dashboard still hides IP even if the setting is true

**Step 2: Run the focused dashboard test to verify failure**

Run:

`cd frontend && npm test -- dashboard-states.test.tsx`

Expected: FAIL because `Dashboard` does not load or apply the new setting yet.

**Step 3: Write the minimal implementation**

Update `Dashboard` to:

- request dashboard settings once on mount
- keep IP hidden until settings load
- pass `showIP={accessMode !== "guest" && showDashboardCardIP}`

**Step 4: Re-run the focused dashboard test**

Run:

`cd frontend && npm test -- dashboard-states.test.tsx`

Expected: PASS.

### Task 6: Run the full targeted verification and restart validation

**Files:**
- Verify only

**Step 1: Run the targeted backend tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/store ./internal/api`

Expected: PASS.

**Step 2: Run the targeted frontend tests**

Run:

`cd frontend && npm test -- node-card-redesign.test.tsx dashboard-states.test.tsx settings-states.test.tsx`

Expected: PASS.

**Step 3: Run the required restart validation**

Run:

`make dev-restart TOKEN=thism2026 PORT=12026`

Expected: frontend assets rebuild, Go server rebuild, and runtime restart complete successfully on port `12026`.
