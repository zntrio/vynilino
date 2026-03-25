## 1. Project Setup & Build Pipeline

- [x] 1.1 Create `ui/` directory with `package.json` (Alpine.js, HTMX, Tailwind CSS 4.x as devDeps), `vite.config.js`, and `.gitignore` for `ui/dist/`
- [x] 1.2 Configure Vite to output to `ui/dist/` with hashed asset filenames and a flat `index.html` entry point
- [x] 1.3 Add Tailwind CSS configuration (`tailwind.config.js`) scanning `ui/src/**/*.{html,js}`
- [x] 1.4 Add `Makefile` targets: `ui-install` (npm ci), `ui-build` (vite build), `ui-dev` (vite dev with proxy to Go backend); update `make build` to depend on `ui-build`
- [x] 1.5 Update Docker multi-stage build: add a `node:22-alpine` stage that runs `ui-build` before the Go build stage

## 2. Go Backend — Static Serving & New Endpoints

- [x] 2.1 Create `internal/adapter/ui/` package with `//go:embed ui/dist` directive and an `http.FileServer` handler over `embed.FS`
- [x] 2.2 Implement SPA fallback middleware: for GET requests not matching `/graphql`, `/api/`, `/auth/`, or a known static asset, serve `ui/dist/index.html`
- [x] 2.3 Register the UI adapter in the chi router after existing adapters (graphql, auth) so API routes take precedence
- [x] 2.4 Implement `GET /api/me` handler that reads the session cookie and returns `{"id","email","name"}` or `401`
- [x] 2.5 Implement `POST /api/upload` handler that accepts multipart form with an image file (≤ 5 MB), delegates to the existing filestore adapter, and returns `{"url": "<stored-url>"}`
- [x] 2.6 Add unit tests for the SPA fallback middleware (API paths not intercepted, unknown paths return index.html)
- [x] 2.7 Add integration test: `GET /` returns 200 with HTML content-type after `ui-build`

## 3. UI Shell & Authentication

- [x] 3.1 Create `ui/src/index.html` as the single HTML entry point with Alpine.js and HTMX loaded from `node_modules` (bundled by Vite), Tailwind CSS stylesheet link
- [x] 3.2 Implement boot script (`ui/src/main.js`): on load call `GET /api/me`; if 401 redirect to `/auth/login`; otherwise store user in Alpine global store
- [x] 3.3 Implement client-side router (`ui/src/router.js`): listen to `popstate`, map `/` → CollectionList, `/records/new` → AddRecord, `/records/:id` → EditRecord
- [x] 3.4 Build app shell layout component (`ui/src/components/AppShell.html`): desktop left sidebar (nav links + user name + logout link) and mobile bottom nav bar (CSS media query, `max-width: 767px`)
- [x] 3.5 Add logout link pointing to `GET /auth/logout` in the sidebar and mobile nav

## 4. Collection List View

- [x] 4.1 Create `CollectionList` component (`ui/src/views/CollectionList.js`): Alpine data function that executes `records(first: 20)` GraphQL query on init
- [x] 4.2 Render collection grid/list: card per record showing cover art thumbnail (or placeholder), title, artist, year, format badge, condition badge
- [x] 4.3 Implement "Load more" button: fetches next page with `endCursor` and appends to list; hide button when `hasNextPage` is false
- [x] 4.4 Implement empty-state view: illustration + "Add your first record" CTA button
- [x] 4.5 Open WebSocket subscription (`recordChanged`) on component mount; update list in real time on received events; close subscription on component unmount

## 5. Search & Filter Panel

- [x] 5.1 Add search input with 300 ms debounce (Alpine `x-model` + `x-effect`) that re-executes `records(query: ...)` and replaces list
- [x] 5.2 Add format multi-select filter (LP, EP, Single, 7", 10", 12") that passes selected values as `filter.formats` to the query
- [x] 5.3 Add "Clear filters" button that resets all filter state and reloads the unfiltered list
- [x] 5.4 Show active filter count badge on the filter toggle button (mobile)

## 6. Add Record Form

- [x] 6.1 Create `AddRecord` view (`ui/src/views/AddRecord.js`): Alpine component with form state for all collection fields (title, artist, year, label, format, condition, genre, notes, coverArtUrl)
- [x] 6.2 Implement client-side validation: required fields (title, artist) show inline error messages; form not submitted until valid
- [x] 6.3 Implement cover art file input: on file selection validate size ≤ 5 MB, then POST to `/api/upload`, store returned URL in form state; show thumbnail preview
- [x] 6.4 On form submit: execute `createRecord` GraphQL mutation; on success navigate back to `/` and show success toast; on duplicate warning show inline banner while still adding record
- [x] 6.5 On mobile: render form in a bottom sheet drawer (full-width, slides up from bottom); on desktop: render as a centered modal or full-page form

## 7. Edit Record Form

- [x] 7.1 Create `EditRecord` view (`ui/src/views/EditRecord.js`): pre-populate form with values from `record(id)` query; reuse form field components from AddRecord
- [x] 7.2 Implement optimistic update: immediately update list item with new values before mutation resolves; revert on error with error toast
- [x] 7.3 On successful `updateRecord` mutation: navigate back to `/` and show success toast

## 8. Delete Record

- [x] 8.1 Add "Delete" button on each record card and in the edit form
- [x] 8.2 Show confirmation dialog (Alpine `x-show` modal) with record title; "Cancel" dismisses; "Delete" executes `deleteRecord` mutation
- [x] 8.3 On successful deletion: remove record from list immediately and show success toast

## 9. Polish & Accessibility

- [x] 9.1 Add loading skeleton placeholders while collection list is fetching
- [x] 9.2 Add error state UI for failed GraphQL requests (retry button)
- [x] 9.3 Ensure all interactive elements have visible focus rings (Tailwind `focus-visible:ring`)
- [x] 9.4 Add `aria-label` attributes to icon-only buttons (delete, edit, upload)
- [x] 9.5 Verify keyboard navigation through list, forms, and modals (Tab, Escape to close modal)
- [x] 9.6 Measure gzipped JS bundle size; must be ≤ 50 kB (run `vite build --report`)

## 10. Documentation

- [x] 10.1 Update `README.md` with UI development workflow (`make ui-dev`, `make ui-build`)
- [x] 10.2 Document `/api/me` and `/api/upload` endpoints in the project's API reference
- [x] 10.3 Add architecture note to `docs/` (or inline in README) explaining the embed.FS approach and the SPA fallback routing rule
