import Alpine from '@alpinejs/csp'
import { router } from './router.js'

window.Alpine = Alpine
import { gql } from './lib/gql.js'
import { CollectionList } from './views/CollectionList.js'
import { AddRecord } from './views/AddRecord.js'
import { EditRecord } from './views/EditRecord.js'
import { Login } from './views/Login.js'

// Expose router globally for inline onclick handlers in templates.
window._router = router

// ── Alpine global store ───────────────────────────────────────────────────────
Alpine.store('auth', {
  user: null,
  // Token is no longer stored in localStorage. The server sets an HttpOnly
  // cookie (vynilino_access) that the browser sends automatically with every
  // request. This field is kept for backward compatibility with templates that
  // check auth.token, but it holds a transient in-memory copy only.
  token: null,
  ready: false,

  setToken(token) {
    this.token = token
    // Do not persist to localStorage — the HttpOnly cookie is the authoritative
    // credential store (THREAT-006 mitigation).
  },

  logout() {
    this.setToken(null)
    this.user = null
    router.navigate('/login')
  },
})

Alpine.store('toast', {
  messages: [],
  show(text, type = 'success') {
    const id = Date.now()
    this.messages.push({ id, text, type })
    setTimeout(() => {
      this.messages = this.messages.filter(m => m.id !== id)
    }, 3500)
  },
})

// ── Routes ───────────────────────────────────────────────────────────────────
router
  .on('/login', (el) => Login(el))
  .on('/', (el) => CollectionList(el))
  .on('/records/new', (el) => AddRecord(el))
  .on('/records/:id', (el, params) => EditRecord(el, params))

// ── Boot sequence ─────────────────────────────────────────────────────────────
async function boot() {
  // With HttpOnly cookie auth we cannot read the token from JS. Just probe the
  // server — if the cookie is valid the me query succeeds; otherwise redirect.
  try {
    const data = await gql(`query { me { id email } }`)
    Alpine.store('auth').user = data.me
  } catch {
    window.location.href = '/login'
    return
  }

  Alpine.start()
  router.init()
}

boot()
