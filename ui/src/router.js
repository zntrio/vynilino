/**
 * Minimal client-side History API router.
 * Maps URL pathnames to view render functions.
 */

const routes = {}

export const router = {
  /** Register a route: router.on('/path', renderFn) */
  on(pattern, renderFn) {
    routes[pattern] = renderFn
    return this
  },

  /** Navigate to a path (pushes history state). */
  navigate(path) {
    history.pushState(null, '', path)
    this._render(path)
  },

  /** Replace current history entry and render. */
  replace(path) {
    history.replaceState(null, '', path)
    this._render(path)
  },

  /** Start listening to popstate and render current path. */
  init() {
    window.addEventListener('popstate', () => {
      this._render(location.pathname)
    })
    this._render(location.pathname)
  },

  _render(path) {
    const app = document.getElementById('app')
    if (!app) return

    // Match exact path first, then check for dynamic segments.
    if (routes[path]) {
      routes[path](app)
      return
    }

    // Dynamic route: /records/:id
    for (const [pattern, fn] of Object.entries(routes)) {
      const regex = patternToRegex(pattern)
      const match = path.match(regex)
      if (match) {
        fn(app, match.groups ?? {})
        return
      }
    }

    // 404 → fall back to collection list.
    if (routes['/']) {
      routes['/'](app)
    }
  },
}

function patternToRegex(pattern) {
  const escaped = pattern
    .replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    .replace(/:\w+/g, (m) => `(?<${m.slice(1)}>[^/]+)`)
  return new RegExp(`^${escaped}$`)
}
