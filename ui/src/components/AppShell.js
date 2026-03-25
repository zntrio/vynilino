import { router } from '../router.js'

/**
 * Renders the persistent app shell (sidebar on desktop, bottom nav on mobile)
 * and a <main> slot where view content is injected.
 *
 * @param {HTMLElement} container - the #app element
 * @param {string} activeRoute - current route key for nav highlight
 * @param {() => string} contentFn - function that returns the inner HTML for <main>
 */
export function renderShell(container, activeRoute, contentFn) {
  const user = Alpine.store('auth').user
  const email = user?.email ?? ''

  container.innerHTML = /* html */`
    <div class="flex h-full" x-data>
      <!-- ── Desktop sidebar ─────────────────────────────────────────── -->
      <aside class="hidden md:flex md:flex-col md:w-56 bg-zinc-900 border-r border-zinc-800 shrink-0">
        <div class="px-5 py-6 flex items-center gap-2">
          <svg class="w-6 h-6 text-violet-400" fill="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span class="font-semibold tracking-tight text-zinc-100">Vynilino</span>
        </div>

        <nav class="flex-1 px-3 space-y-1">
          ${navLink('/', 'collection', activeRoute, /* html */`
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            Collection
          `)}
        </nav>

        <div class="px-4 py-4 border-t border-zinc-800">
          <p class="text-xs text-zinc-500 truncate mb-2">${escHtml(email)}</p>
          <button
            @click="$store.auth.logout()"
            class="text-xs text-zinc-400 hover:text-zinc-100 transition-colors focus-visible:outline focus-visible:ring-2 focus-visible:ring-violet-500 rounded"
            aria-label="Log out"
          >Log out</button>
        </div>
      </aside>

      <!-- ── Main content ───────────────────────────────────────────────── -->
      <div class="flex-1 flex flex-col min-h-0 overflow-hidden">
        <!-- Toast container -->
        <div
          class="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none"
          x-data
        >
          <template x-for="msg in $store.toast.messages" :key="msg.id">
            <div
              class="px-4 py-3 rounded-lg shadow-lg text-sm font-medium pointer-events-auto transition-all"
              :class="msg.type === 'error' ? 'bg-red-600 text-white' : 'bg-violet-600 text-white'"
              x-text="msg.text"
            ></div>
          </template>
        </div>

        <main id="view" class="flex-1 overflow-y-auto px-4 md:px-8 py-6">
          ${contentFn()}
        </main>
      </div>

      <!-- ── Mobile bottom nav ──────────────────────────────────────────── -->
      <nav class="md:hidden fixed bottom-0 inset-x-0 bg-zinc-900 border-t border-zinc-800 flex justify-around py-2 z-40">
        <button
          @click="window._router.navigate('/')"
          class="flex flex-col items-center gap-0.5 px-4 py-1 text-xs ${activeRoute === 'collection' ? 'text-violet-400' : 'text-zinc-400'} focus-visible:ring-2 focus-visible:ring-violet-500 rounded"
          aria-label="Collection"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          Collection
        </button>
        <button
          @click="$store.auth.logout()"
          class="flex flex-col items-center gap-0.5 px-4 py-1 text-xs text-zinc-400 focus-visible:ring-2 focus-visible:ring-violet-500 rounded"
          aria-label="Log out"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h6a2 2 0 012 2v1" />
          </svg>
          Log out
        </button>
      </nav>
    </div>
  `
}

function navLink(path, key, activeRoute, inner) {
  const active = activeRoute === key
  return /* html */`
    <button
      @click="window._router.navigate('${path}')"
      class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:ring-violet-500
        ${active ? 'bg-violet-600/20 text-violet-300' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100'}"
    >${inner}</button>
  `
}

function escHtml(str) {
  return str.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))
}
