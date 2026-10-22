# Playwright coverage matrix

The sitewide suite checks every registered page at 320, 375, 390, 768, 1024 and 1440 CSS pixels for horizontal overflow. It saves desktop and mobile screenshots for each route. English/Traditional Chinese and light/dark combinations are checked on the personal ledger, group audit and admin audit pages. These are browser viewport checks, not physical-device tests.

| Route or feature | Actor | Success path covered | Rejection or failure covered | Playwright coverage |
| --- | --- | --- | --- | --- |
| Sign in | Owner | UI sign-in with seeded isolated backend | Invalid credentials are not covered | `sitewide-ui.spec.ts`; share and notification setup helpers |
| Registration and first setup | New user / first administrator | Registration API is used to seed share and notification tests; isolated backend is initialized by the test server | UI registration and a separate uninitialized setup instance are not covered | `share-flow.spec.ts`, `share-attachments.spec.ts`, `notifications.spec.ts`; setup server in `start.mjs` |
| Personal ledger | Owner | Expense create, edit, cancel, validation and confirmed delete; previous day, date picker and query preservation | Network failure and authorization denial are not covered | `sitewide-ui.spec.ts` |
| Personal income and subscriptions | Owner | Routes render | Create, edit, cancel and delete flows are not covered | `sitewide-ui.spec.ts` route smoke |
| Personal shares and public share | Owner / public visitor | Create, edit, disable, open public share; fixed-date validation; attachment sharing opt-in persists | Invalid public token is route-smoke only; request failure is not injected | `share-flow.spec.ts`, `share-attachments.spec.ts`, `sitewide-ui.spec.ts` |
| Groups, ledger and settings | Group owner | Create/cancel/required-field validation, update group name, route rendering, ledger date navigation | Member and restricted-role permissions; financial CRUD; request failure are not covered | `sitewide-ui.spec.ts` |
| Group members, roles, audit, shares and settings | Group owner | Audit search/apply/clear; pages render; mobile navigation displays allowed owner links | Member/role mutations, denied access and audit-write failures are not covered | `sitewide-ui.spec.ts` |
| Notifications | Owner | Save preferences, open notification dialog, close with Escape | Backend failure and unauthorized actor are not covered | `notifications.spec.ts` |
| Admin sections | Administrator | All admin routes render at all six widths | Admin mutations, role denial and backend failures are not covered | `sitewide-ui.spec.ts` route and layout matrix |
| Profile, categories, contacts, attachments, about | Owner | Routes render; attachment sharing preference is covered separately | Page-specific create/edit/delete operations are not covered | `sitewide-ui.spec.ts`, `share-attachments.spec.ts` |
| Confirmation dialog | Owner | Focus enters dialog; cancel and confirm; Escape behavior is covered by notification dialog | Focus restoration is implemented but not asserted in E2E | `sitewide-ui.spec.ts`, `notifications.spec.ts` |

## Validation status

- Full E2E suite: 15 tests across Chromium, Firefox and WebKit. The latest all-route six-width matrix passed in all three browsers; the other four E2E scenarios passed in the preceding full-suite run.
- Frontend unit tests: 135 passed across 33 files.
- Frontend production build and changed-file ESLint checks passed.
- Screenshots are retained under Playwright `test-results` locally and uploaded by the Verify workflow as a 14-day artifact.

Rows explicitly marked as not covered are follow-up work; a route rendering check is not counted as functional verification.
