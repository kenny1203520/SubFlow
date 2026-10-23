# Playwright coverage matrix

The sitewide suite checks the 28 route targets listed in `sitewide-ui.spec.ts` at 320, 375, 390, 768, 1024 and 1440 CSS pixels for horizontal overflow. It asserts the requested URL remains active and the app does not render its runtime-error fallback, then saves desktop and mobile screenshots. English/Traditional Chinese and light/dark combinations are checked on the personal ledger, group audit and admin audit pages. The suite runs Chromium, Firefox and WebKit desktop projects plus Chromium and WebKit touch emulation. These are browser contexts, not physical-device tests; the language/theme matrix is representative rather than every combination on every route.

| Route or feature | Actor | Success path covered | Rejection or failure covered | Playwright coverage |
| --- | --- | --- | --- | --- |
| Sign in | Owner / regular user | UI sign-in with isolated backend; production-container login | Invalid credentials covered by production-container browser check | `sitewide-ui.spec.ts`; `scripts/release/container-smoke.mjs` |
| Registration and first setup | New user / first administrator | Separate uninitialized production container uses setup and registration UI; main Playwright server initializes separate test data via API | Empty setup form and repeat initialization rejected; full registration validation remains incomplete | `scripts/release/container-smoke.mjs`; API fixture server in `start.mjs` |
| Explicit sign out | Regular user | UI sign out followed by unauthenticated screen; backend success audit | Real anonymous rejection/failure audit; audit failure returns 503 in Go; frontend retains session on network/audit failure in Vitest | `container-smoke.mjs`, `logout_api_test.go`, `auth.test.ts` |
| Personal ledger | Owner / isolated regular user | Expense create, edit, cancel, validation and confirmed delete; previous day, date picker and query preservation; save remains usable during a delayed real SSE refresh | Network outage and authorization denial are not covered; duplicate submit and rejected write are checked in Vitest | `sitewide-ui.spec.ts`, `ledger-save.spec.ts`, `LedgerQuickAddDrawer.test.ts` |
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
| Attachment drafts and previews | Isolated regular user / component harness | Draft files selected by desktop drop or touch file input; no early upload; cancel clears selection; a real one-page PDF renders nonblank pixels | Dropzone disabled and PDF load failure covered in Vitest; image upload-to-preview and download are covered by the share regression; Office browser flows remain incomplete | `attachment-dropzone.spec.ts`, `pdf-preview.spec.ts`, `AttachmentDraftField.test.ts`, `AttachmentDropzone.test.ts`, `AttachmentPdfPreview.test.ts`, `attachmentPreview.test.ts` |

## Validation status

- Full E2E configuration: 95 tests (19 scenarios across five projects). Complete execution results are recorded by CI and the release evidence; no project is skipped for publication.
- Frontend unit tests: 158 passed across 39 files.
- Frontend production build and changed-file ESLint checks passed.
- Screenshots are retained under Playwright `test-results` locally and uploaded by the Verify workflow as a 14-day artifact.

Rows explicitly marked as not covered are follow-up work; a route rendering check is not counted as functional verification.

## Save-race follow-up and CI isolation

- Post-merge CI exposed Firefox losing a save click while SSE background reads briefly toggled the global loading state. A deterministic regression holds a real SSE-triggered expense read, then requires a valid edit to send its real PATCH successfully. Before repair, the save button remained disabled; the repaired behavior passes all five browser projects.
- The drawer now tracks its own pending submission independently and rejects duplicate submits. A rejected write keeps the form values and error feedback available for retry. Existing backend mutations and audit behavior are unchanged.
- Sitewide mutation names include a unique identifier for each test attempt. A failed attempt's records cannot match another browser's expense, income, subscription or group selectors.
- CI runs each browser project in a separate job with its own temporary PocketBase instance. All five jobs are required and fail-fast is disabled. Browser binaries and OS fonts come from the official Playwright 1.64.1 Noble image pinned by digest; an explicit SDK-version check guards mismatches. This replaces repeated Ubuntu browser-dependency installation in the full browser suite. The production-container gate remains separate.

## Additional release qualification

- `scripts/release/container-smoke.mjs` starts a production image with fresh temporary data, drives setup/registration/auth/admin UI in Chromium and checks backend permission and logout audit results. It is required on trusted CI branches and version publication.
- `scripts/release/upgrade-smoke.mjs` uses a pinned Alpha 1 image and generated data, backs up the stopped database, upgrades/restarts the candidate and restores a separate backup under Alpha 1. IDs/amounts, groups, attachment bytes and legacy notification backfill are checked. This does not qualify every historical data shape.
- `notification_smtp_test.go` uses the real mailer over loopback TCP, checking delivery, rejection, outbox state, duplicate prevention and backend audit outcomes. External provider TLS and inbox deliverability are not tested. Existing unit tests check bounded retries; successful administrator retry of a real failed job remains outside the browser suite.
- Release contract tests verify metadata and final publication outputs, including alpha preservation of stable latest. Deliberate dispatch against a branch is a negative test and must fail before publication.
- E2E startup withholds raw backend logs because setup tokens, credentials or SQL values could otherwise reach uploaded diagnostics.

## Share editor and mobile controls

- Touch emulation exposed a hidden checkbox inheriting full input width and expanding a 1024px page to 1705px. The checkbox now has explicit one-pixel dimensions inside its 44px label, with visible keyboard focus on the custom switch.
- Date cases independently check Taipei/UTC/New York at 375/768/1024/1440px before saving and after clipboard-denied feedback. Measurements compare document width to the configured width; expanded mobile innerWidth cannot produce a false pass. Diagnostics omit text that might contain a secret share URL.
- Inspected mobile header and light/dark share editor screenshots. Language controls retain an accessible name when their visible text is hidden. Content option cards use a readable minimum width rather than squeezing words into thin columns.
- Screenshots provide review evidence, not automatically approved visual baselines. Complete per-route visual approval and physical-device checks remain outstanding.

## Admin visual review and diagnosis

- Reviewed 320px and 1440px failed-email screenshots plus mobile user-list and empty-email screens. Long error codes now wrap completely; the active admin tab remains visible when the navigation scrolls. Desktop rows place the action beside the text only when their container has enough width.
- Long failed-job records are explicitly simulated UI fixtures. List recovery, permission rejection and the missing-job retry/audit assertions use the isolated real backend.
- The production backend launched from this checkout returns 200 for `/api/subflow/v1/system/notification-emails/failed?page=1&perPage=25&sort=-created`. The supplied server-side 404 was not reproduced there. An existing server still returning a default 404 should be checked for a frontend/backend version mismatch or different API target.
- The reproduced frontend defect was a missing `notification-emails` alternative in the admin route pattern: a menu link worked but refreshing redirected to `/personal`. The focused regression test failed before the route fix and now passes. Section-specific loading also prevents unrelated settings/users/roles/audit requests when visiting this page.

## Alpha 3 shared-attachment white-overlay regression

- The published Alpha 3 image was reproduced by its immutable digest. A populated share mounted an attachment dialog with no open attribute, but PDF global CSS forced display:flex; it covered the viewport at phone widths and part of desktop screens. Empty-share smoke tests had not exercised this state.
- Dialog layout now belongs to ShareAttachments and only applies to [open]. An explicit trigger reference also restores focus in WebKit, where pointer activation does not necessarily focus a button.
- share-view.spec.ts adds a minimal simulated projection plus real-backend link/password/accounts shares with uploaded PNGs. Every project checks 320/375/390/768/1024/1440px in light/dark, closed/open state, geometry, Escape, focus return, rejected-read retry, image decoding, download and unauthenticated/wrong-password rejection. These are five browser projects, not physical devices.
- Production-container qualification now creates a real anonymous attachment share, verifies the closed dialog stays hidden, and opens/previews/closes it at mobile and desktop widths. This checks bundled CSS before future publication.

- Manual review also exposed overlapping type/amount/attachment controls on 320px populated shares. Record rows now use bounded grid tracks with stacked mobile content; tests assert control rectangles do not overlap and titles are not clipped.
