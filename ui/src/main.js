import Alpine from 'alpinejs'
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
  token: localStorage.getItem('vynilino_token'),
  ready: false,

  setToken(token) {
    this.token = token
    if (token) {
      localStorage.setItem('vynilino_token', token)
    } else {
      localStorage.removeItem('vynilino_token')
    }
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
  const token = Alpine.store('auth').token

  if (!token) {
    window.location.href = '/login'
    return
  }

  try {
    const data = await gql(`query { me { id email } }`)
    Alpine.store('auth').user = data.me
  } catch {
    Alpine.store('auth').setToken(null)
    window.location.href = '/login'
    return
  }

  Alpine.start()
  router.init()
}

boot()
