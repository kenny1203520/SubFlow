# Playwright coverage matrix

The sitewide suite checks the 28 route targets listed in `sitewide-ui.spec.ts` at 320, 375, 390, 768, 1024 and 1440 CSS pixels for horizontal overflow. It asserts the requested URL remains active and the app does not render its runtime-error fallback, then saves desktop and mobile screenshots. English/Traditional Chinese and light/dark combinations are checked on the personal ledger, group audit and admin audit pages. These are browser viewport checks, not physical-device tests. Setup and authentication coverage limits are listed below.

| Route or feature | Actor | Success path covered | Rejection or failure covered | Playwright coverage |
| --- | --- | --- | --- | --- |
| Sign in | Owner | UI sign-in with seeded isolated backend | Invalid credentials are not covered | `sitewide-ui.spec.ts`; share and notification setup helpers |
| Registration and first setup | New user / first administrator | Registration API is used to seed share and notification tests; isolated backend is initialized by the test server | UI registration and a separate uninitialized setup instance are not covered | `share-flow.spec.ts`, `share-attachments.spec.ts`, `notifications.spec.ts`; setup server in `start.mjs` |
| Personal ledger | Owner | Expense create, edit, cancel, validation and confirmed delete; previous day, date picker and query preservation | Network failure and authorization denial are not covered | `sitewide-ui.spec.ts` |
| Personal income | Owner | Create and confirmed delete; empty API result renders safely; ledger refresh uses the viewer's date | Edit, cancel and backend failure are not covered | `sitewide-ui.spec.ts`; `workspace.test.ts` null-list regression |
| Personal subscriptions | Owner | Create and confirmed delete | Edit, stop/cancel-stop, period history and backend failure are not covered | `sitewide-ui.spec.ts` |
| Personal shares and public share | Owner / public visitor | Create, edit, disable, open public share; fixed-date validation; attachment sharing opt-in persists | Invalid public token is route-smoke only; request failure is not injected | `share-flow.spec.ts`, `share-attachments.spec.ts`, `sitewide-ui.spec.ts` |
| Groups, ledger and settings | Group owner | Create/cancel/required-field validation, update group name, route rendering, ledger date navigation | Member and restricted-role permissions; financial CRUD; request failure are not covered | `sitewide-ui.spec.ts` |
| Group members, roles, audit, shares and settings | Group owner | Audit search/apply/clear; pages render; mobile navigation displays allowed owner links | Member/role mutations, denied access and audit-write failures are not covered | `sitewide-ui.spec.ts` |
| Notifications | Owner | Save preferences, open notification dialog, close with Escape | Backend failure and unauthorized actor are not covered | `notifications.spec.ts` |
| Admin sections | Administrator | All admin routes render at all six widths | Admin mutations, role denial and backend failures are not covered | `sitewide-ui.spec.ts` route and layout matrix |
| Admin notification emails | Administrator / regular account | Real backend list returns 200; named navigation and direct reload; URL pagination and page size; pending/retry-success UI in Vitest | Simulated list 404 recovers through the real backend; real API denies a regular account with 403; real retry of a missing job returns 404 and writes a failure audit; row remains on rejection. Successful retry mutation against a real failed job is not covered | `admin-notification-emails.spec.ts`, `AdminView.test.ts` |
| Profile, categories, contacts, attachments, about | Owner | Routes render; attachment sharing preference is covered separately | Page-specific create/edit/delete operations are not covered | `sitewide-ui.spec.ts`, `share-attachments.spec.ts` |
| Confirmation dialog | Owner | Focus enters dialog; cancel and confirm; Escape behavior is covered by notification dialog | Focus restoration is implemented but not asserted in E2E | `sitewide-ui.spec.ts`, `notifications.spec.ts` |

## Validation status

- Full E2E suite: 27 tests across Chromium, Firefox and WebKit, all passed in the latest complete run.
- Frontend unit tests: 139 passed across 34 files.
- Frontend production build and changed-file ESLint checks passed.
- Screenshots are retained under Playwright `test-results` locally and uploaded by the Verify workflow as a 14-day artifact.

Rows explicitly marked as not covered are follow-up work; a route rendering check is not counted as functional verification.

## Admin visual review and diagnosis

- Reviewed 320px and 1440px failed-email screenshots plus mobile user-list and empty-email screens. Long error codes now wrap completely; the active admin tab remains visible when the navigation scrolls. Desktop rows place the action beside the text only when their container has enough width.
- Long failed-job records are explicitly simulated UI fixtures. List recovery, permission rejection and the missing-job retry/audit assertions use the isolated real backend.
- The production backend launched from this checkout returns 200 for `/api/subflow/v1/system/notification-emails/failed?page=1&perPage=25&sort=-created`. The supplied server-side 404 was not reproduced there. An existing server still returning a default 404 should be checked for a frontend/backend version mismatch or different API target.
- The reproduced frontend defect was a missing `notification-emails` alternative in the admin route pattern: a menu link worked but refreshing redirected to `/personal`. The focused regression test failed before the route fix and now passes. Section-specific loading also prevents unrelated settings/users/roles/audit requests when visiting this page.
