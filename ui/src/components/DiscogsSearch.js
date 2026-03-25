import { searchDiscogs } from '../lib/gql.js'

/**
 * Alpine.js component for searching Discogs and selecting a result
 * to pre-populate the Add Record form.
 *
 * Usage: register with Alpine.data('discogsSearch', discogsSearchData)
 *
 * Emits a custom 'discogs-select' event on the element with the selected
 * result as `event.detail`.
 */
export function discogsSearchData() {
  return {
    query: '',
    loading: false,
    error: '',
    results: [],

    async search() {
      const q = this.query.trim()
      if (!q) return
      this.loading = true
      this.error = ''
      this.results = []
      try {
        this.results = await searchDiscogs(q, 'RELEASE')
      } catch (e) {
        this.error = 'Discogs search is temporarily unavailable. You can still add records manually.'
      } finally {
        this.loading = false
      }
    },

    select(result) {
      this.$el.dispatchEvent(new CustomEvent('discogs-select', { bubbles: true, detail: result }))
    },

    get hasResults() {
      return this.results.length > 0
    },

    get noResults() {
      return !this.loading && !this.error && this.query.trim() && this.results.length === 0
    },
  }
}

export function discogsSearchHTML() {
  return /* html */`
  <div
    x-data="discogsSearch"
    class="mb-6 rounded-xl border border-zinc-700 bg-zinc-900 p-4"
  >
    <h2 class="text-sm font-semibold text-zinc-300 mb-3">Search Discogs</h2>

    <!-- Search input -->
    <div class="flex gap-2">
      <input
        type="text"
        x-model="query"
        @keydown.enter.prevent="search()"
        placeholder="Artist, album title, or barcode…"
        class="flex-1 bg-zinc-800 border border-zinc-700 text-zinc-100 text-sm rounded-lg px-3 py-2 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-violet-500"
      />
      <button
        type="button"
        @click="search()"
        :disabled="loading || !query.trim()"
        class="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors disabled:opacity-50"
      >
        <span x-show="!loading">Search</span>
        <span x-show="loading">…</span>
      </button>
    </div>

    <!-- Error state -->
    <p x-show="error" x-text="error" class="mt-2 text-sm text-amber-400"></p>

    <!-- No results state -->
    <p x-show="noResults" class="mt-2 text-sm text-zinc-500">No results found. Try a different query.</p>

    <!-- Results list -->
    <ul x-show="hasResults" class="mt-3 space-y-2 max-h-72 overflow-y-auto pr-1">
      <template x-for="r in results" :key="r.discogsId">
        <li>
          <button
            type="button"
            @click="$dispatch('discogs-select', r)"
            class="w-full flex items-center gap-3 rounded-lg p-2 hover:bg-zinc-800 transition-colors text-left"
          >
            <img
              x-show="r.thumbUrl"
              :src="r.thumbUrl"
              class="w-10 h-10 rounded object-cover flex-shrink-0 bg-zinc-800"
              loading="lazy"
              alt=""
            />
            <div x-show="!r.thumbUrl" class="w-10 h-10 rounded flex-shrink-0 bg-zinc-800 flex items-center justify-center text-zinc-600">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <circle cx="12" cy="12" r="10" stroke-width="1.5"/>
                <circle cx="12" cy="12" r="3" stroke-width="1.5"/>
              </svg>
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-zinc-100 truncate" x-text="r.artist ? r.artist + ' — ' + r.title : r.title"></p>
              <p class="text-xs text-zinc-500 truncate">
                <span x-show="r.year" x-text="r.year"></span>
                <span x-show="r.year && (r.label || r.format)"> · </span>
                <span x-show="r.label" x-text="r.label"></span>
                <span x-show="r.label && r.format"> · </span>
                <span x-show="r.format" x-text="r.format"></span>
              </p>
            </div>
          </button>
        </li>
      </template>
    </ul>
  </div>
  `
}
