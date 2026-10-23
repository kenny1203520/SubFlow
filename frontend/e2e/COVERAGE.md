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

- Full E2E configuration: 130 tests (26 scenarios across five projects). Complete execution results are recorded by CI and qualification evidence; no project is skipped for publication.
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

## Mobile forms, overview and production preview regressions

The group route fixture now includes a long unbroken name. Firefox with the pinned Linux fonts exposed a card widening a 320px page to 370px after the old clipping mask was removed; an explicit unbroken fixture also reproduced the issue locally. Group cards and headings now wrap these values rather than relying on browser-specific UUID line breaks.

Manual mobile screenshot review exposed profile/admin forms clipped by an old page-wide overflow-x:hidden rule. The rule is removed. Form and workspace grids now use shrinkable tracks, dropdown roots have no intrinsic minimum, and list headings wrap. Sitewide checks also measure form inputs, selects, textareas, buttons and primary heading actions against viewport bounds; document width alone cannot approve clipped controls. Overflow diagnostics include only path, element tag/class and geometry, never field values or share queries.

| Feature | Actor and real-backend path | Assertions | Limits |
| --- | --- | --- | --- |
| Participant binding | Isolated group owner with a placeholder member | Invite & bind reveals/focuses the invitation form at 320/390/768/1440px with both motion preferences; cancellation sends no invitation | Invitation acceptance and restricted-member mutation paths remain outside this case |
| Modal category and nested icon menus | Isolated user opens the personal ledger drawer | Options accept real pointer clicks; Escape closes only the active menu, restores focus and leaves the drawer open; simulated reduced visual viewport keeps options reachable | The keyboard viewport is injected, not a physical on-screen keyboard |
| Financial and profile forms | Isolated user and group owner | Personal/group expense, income and subscription forms at 320/768/1440px: category, available currency, billing cycle, input focus, cancellation without writes and no page overflow; profile currency/timezone selection | Does not claim all form validation, successful mutations, role denial or backend failure paths |
| Overview metrics | Isolated user with a large real expense | Six widths; equal card tracks, values and children not clipped, cards stay within their parent; month picker at least 140px wide and 44px high; mobile/desktop screenshots | This case uses personal overview; route checks cover the group overview separately |
| Touch text controls and zoom | Isolated user opens ledger form | Visible text inputs/selects/textarea are at least 16px on narrow/coarse-pointer contexts; viewport metadata permits zoom and body touch-action stays auto | Focus autozoom, pinch and double-tap gestures still need physical iOS qualification |
| PDF compatibility | Real PDF component harness | Nonblank canvas pixels with native APIs and with Promise.try removed from both window and worker; all five projects | Capability injection is explicitly simulated; it does not identify a user's exact Safari minor version |
| Uploaded PDF | Isolated user uploads a valid PDF to a real expense and creates an attachment-enabled public share | Personal and public dialogs render nonblank canvas pixels, show no preview error, and close correctly; screenshots wait for actual content | Generated one-page PDF; complex, malformed, encrypted and large files remain outside this case |
| Production CSP | Fresh production container, anonymous shared PDF | Closed dialog stays hidden; bundled worker renders nonblank canvas at 390/1440px; backend test requires blob connect permission while retaining script/worker/object restrictions | Chromium touch emulation plus desktop, not physical Safari |

The confirmed production PDF failure was CSP rejecting fetch(blob:), which development Vite does not reproduce. The required container check now uses a real uploaded PDF rather than an image-only attachment. No new write endpoint, audit event, schema or financial rule is introduced by these presentation/security-header fixes. Existing fixture writes use the existing backend APIs.

See [Alpha 3 UI regression qualification](../../docs/validation/alpha3-ui-regressions.md). Screenshots are review evidence, not approved visual baselines; the full site/role/failure acceptance plan remains incomplete.
