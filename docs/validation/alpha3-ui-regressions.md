# Alpha 3 UI regression qualification

These fixes follow the published `v0.2.0-alpha.3` image. They do not replace its tag, image or historical qualification results. A subsequent release or deployment is required to deliver them.

## Reproduced defects and repairs

| Defect | Evidence before repair | Repair and regression |
| --- | --- | --- |
| White attachment shares | Exact published image digest reproduced a visible closed native dialog, covering phone screens and part of desktop | Dialog layout applies only to `[open]`; all three share modes and six widths check visibility, geometry, image preview/download, Escape and focus return |
| Shared row overlap | Manual 320px screenshot review exposed intersecting amount/type/attachment controls | Bounded grid tracks, wrapping titles and a separate action row; geometric no-overlap assertions |
| Binding seems unresponsive | The selected member changed but the invitation form stayed off screen | Render the target, immediately scroll the form into view, then focus email without a second scroll; cancellation sends no invitation |
| Mobile category selection | Pointer clicks intercepted by the native modal because the teleported menu was outside its inert subtree | Keep menus inside the current modal or parent menu; nested Escape and reduced visual viewport checks |
| Squeezed overview cards | Large real expense and unequal card geometry failed the new test | Responsive equal tracks, shrinkable card contents and wrapped long values |
| Squeezed overview month picker | Actual desktop screenshot and a failing measurement showed a 25px-wide picker | Separate navigation/action rows; regression requires at least 140px width and 44px height at all six widths |
| Clipped profile/admin forms and list controls | Mobile screenshot review showed profile controls ending at 428px on a 390px viewport; an old page-wide clipping rule hid overflow from document-width checks | Remove that rule, constrain form/workspace grid tracks and dropdown minima, wrap list toolbars and measure actual form-control bounds on the route matrix |
| Focus zoom | Visible text controls computed to 14px | Minimum 16px on narrow/coarse-pointer contexts; viewport scaling and ordinary touch behavior remain enabled |
| Production PDF fallback | Frontend harness passed, but a production container rejected blob fetches under connect-src and displayed download-only fallback | Add only blob: to connect-src; real production PDF canvas regression and backend CSP restriction test |
| PDF API compatibility | Removing Promise.try from window and worker failed with modern bundles | Matching PDF.js legacy main/worker bundles; nonblank rendering with that capability removed |

The original image was `ghcr.io/kenny1203520/subflow@sha256:0e561c88ed409b1441b48ab44b3ee851db8590adaea87440dc41c8056b48b017`. The isolated candidate container passed fresh setup, registration, invalid login, logout/audit, permission, direct reload and actual shared PDF checks. Candidate images are local qualification builds, not published release artifacts.

## Verification and boundaries

- Frontend: 158 Vitest tests across 39 files; production build; ESLint with no errors (three existing/generated warnings).
- Backend: 427 Go tests across 14 packages and `go vet ./...`.
- Browser configuration: 130 cases, 26 scenarios across Chromium, Firefox and WebKit desktop plus Chromium/WebKit touch emulation. Final execution is reported by the PR checks and local qualification report.
- Financial form interaction covers personal/group expense, income and subscription drawers, plus profile currency/timezone controls. Cancelling these forms must send no financial write.
- The sitewide route matrix remains six widths: 320, 375, 390, 768, 1024 and 1440 CSS pixels. Theme/language combinations are representative, not exhaustive per route.
- No physical iOS device was available. The report of iOS 18 or newer informs the target; actual focus zoom, pinch/double-tap and keyboard behavior remain physical-device checks. Browser touch emulation is not called real-device validation.
- No financial calculation, schema or mutation endpoint changed. New fixture writes use existing audited backend APIs. The CSP change retains same-origin workers, object-src none and the existing script restrictions.
- Full per-role CRUD, backend failure/retry, Office preview flows, arbitrary PDF files and approved visual baselines remain incomplete. See the explicit [coverage matrix](../../frontend/e2e/COVERAGE.md).

## Modern Web Guidance applied

I used the layout guidance, “Do NOT use overflow: hidden to mask layout issues,” to fix grid tracks and wrap content. I did not add page-level clipping.

I followed the motion guidance, “Default to static views,” for binding navigation: immediate scrolling keeps the form and focus together and adds no animation for reduced-motion users.

I used the scroll-completion guidance to sequence scrollIntoView and focus({preventScroll:true}). Awaiting scrollIntoView also works when older browser implementations return undefined.

I consulted the top-layer guidance for native dialog state handling. The repair scopes layout to the open state; no transition or animation was added.

PDF compatibility decisions use the official [PDF.js FAQ](https://github.com/mozilla/pdf.js/wiki/Frequently-Asked-Questions) and [Promise.try compatibility reference](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/try). Capability injection models a missing API; the confirmed production root cause is the separate CSP blob rejection.
