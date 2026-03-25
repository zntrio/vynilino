import Alpine from 'alpinejs'
import { gql, subscribe } from '../lib/gql.js'
import { router } from '../router.js'
import { renderShell } from '../components/AppShell.js'

const RECORDS_QUERY = /* graphql */`
  query ListRecords($first: Int, $after: String, $filter: RecordFilterInput) {
    records(first: $first, after: $after, filter: $filter) {
      edges { node { id title artist year format condition coverArtUrl favorite } }
      pageInfo { hasNextPage endCursor }
    }
  }
`

const TOGGLE_FAVORITE_MUTATION = /* graphql */`
  mutation ToggleFavorite($id: ID!, $input: UpdateRecordInput!) {
    updateRecord(id: $id, input: $input) { id favorite }
  }
`

const RECORD_CHANGED_SUB = /* graphql */`
  subscription {
    recordChanged { type record { id title artist year format condition coverArtUrl favorite } }
  }
`

export function CollectionList(container) {
  renderShell(container, 'collection', () => collectionHTML())

  // Bootstrap Alpine component on the view container.
  const view = document.getElementById('view')
  if (!view) return

  // Set up Alpine component data.
  Alpine.data('collectionList', () => ({
    records: [],
    cursor: null,
    hasMore: false,
    loading: false,
    error: null,
    searchQuery: '',
    selectedFormats: [],
    favoritesOnly: false,
    filterOpen: false,
    activeFilterCount: 0,
    showDeleteModal: false,
    deleteTarget: null,
    searchTimer: null,
    _unsubscribe: null,

    async init() {
      await this.load(true)
      this._unsubscribe = subscribe(
        RECORD_CHANGED_SUB,
        {},
        (data) => this._handleSubscriptionEvent(data.recordChanged),
        (err) => console.error('subscription error', err),
      )
    },

    destroy() {
      this._unsubscribe?.()
    },

    async load(reset = false) {
      this.loading = true
      this.error = null
      try {
        const filter = this._buildFilter()
        const data = await gql(RECORDS_QUERY, {
          first: 20,
          after: reset ? null : this.cursor,
          filter: Object.keys(filter).length ? filter : null,
        })
        const { edges, pageInfo } = data.records
        const newRecords = edges.map(e => e.node)
        this.records = reset ? newRecords : [...this.records, ...newRecords]
        this.cursor = pageInfo.endCursor
        this.hasMore = pageInfo.hasNextPage
      } catch (e) {
        this.error = e.message
      } finally {
        this.loading = false
      }
    },

    async loadMore() {
      await this.load(false)
    },

    onSearchInput() {
      clearTimeout(this.searchTimer)
      this.searchTimer = setTimeout(() => this.load(true), 300)
    },

    toggleFormat(fmt) {
      if (this.selectedFormats.includes(fmt)) {
        this.selectedFormats = this.selectedFormats.filter(f => f !== fmt)
      } else {
        this.selectedFormats.push(fmt)
      }
      this._updateActiveFilterCount()
      this.load(true)
    },

    toggleFavoritesFilter() {
      this.favoritesOnly = !this.favoritesOnly
      this._updateActiveFilterCount()
      this.load(true)
    },

    _updateActiveFilterCount() {
      this.activeFilterCount = this.selectedFormats.length + (this.favoritesOnly ? 1 : 0)
    },

    clearFilters() {
      this.searchQuery = ''
      this.selectedFormats = []
      this.favoritesOnly = false
      this.activeFilterCount = 0
      this.load(true)
    },

    _buildFilter() {
      const f = {}
      if (this.searchQuery.trim()) f.search = this.searchQuery.trim()
      if (this.selectedFormats.length === 1) f.format = this.selectedFormats[0]
      if (this.favoritesOnly) f.favoritesOnly = true
      return f
    },

    async toggleFavorite(record) {
      const newFav = !record.favorite
      // Optimistic update
      this.records = this.records.map(r => r.id === record.id ? { ...r, favorite: newFav } : r)
      try {
        await gql(TOGGLE_FAVORITE_MUTATION, { id: record.id, input: { favorite: newFav } })
      } catch (e) {
        // Revert on error
        this.records = this.records.map(r => r.id === record.id ? { ...r, favorite: !newFav } : r)
        Alpine.store('toast').show('Failed to update favorite: ' + e.message, 'error')
      }
    },

    addRecord() {
      router.navigate('/records/new')
    },

    editRecord(id) {
      router.navigate(`/records/${id}`)
    },

    confirmDelete(record) {
      this.deleteTarget = record
      this.showDeleteModal = true
    },

    cancelDelete() {
      this.showDeleteModal = false
      this.deleteTarget = null
    },

    async doDelete() {
      if (!this.deleteTarget) return
      const id = this.deleteTarget.id
      this.showDeleteModal = false
      this.deleteTarget = null
      try {
        await gql(`mutation DeleteRecord($id: ID!) { deleteRecord(id: $id) }`, { id })
        this.records = this.records.filter(r => r.id !== id)
        Alpine.store('toast').show('Record deleted')
      } catch (e) {
        Alpine.store('toast').show('Delete failed: ' + e.message, 'error')
      }
    },

    _handleSubscriptionEvent(event) {
      if (!event) return
      const { type, record } = event
      if (type === 'DELETED') {
        this.records = this.records.filter(r => r.id !== record.id)
      } else if (type === 'CREATED') {
        if (!this.records.find(r => r.id === record.id)) {
          this.records.unshift(record)
        }
      } else if (type === 'UPDATED') {
        this.records = this.records.map(r => r.id === record.id ? record : r)
      }
    },
  }))

  view.innerHTML = viewHTML()
  Alpine.initTree(view)
}

const FORMATS = [
  { value: 'LP', label: 'LP' },
  { value: 'EP', label: 'EP' },
  { value: 'SINGLE', label: 'Single' },
  { value: 'SEVEN_INCH', label: '7"' },
  { value: 'TEN_INCH', label: '10"' },
  { value: 'TWELVE_INCH', label: '12"' },
]

function collectionHTML() {
  return `<div x-data="collectionList" x-init="init()" @destroy.window="destroy()"></div>`
}

function viewHTML() {
  return /* html */`
  <div x-data="collectionList" x-init="init()" class="pb-20 md:pb-0">
    <!-- ── Header ─────────────────────────────────────────────────── -->
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold tracking-tight">My Collection</h1>
      <button
        @click="addRecord()"
        class="inline-flex items-center gap-2 bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-violet-400"
        aria-label="Add record"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/>
        </svg>
        Add Record
      </button>
    </div>

    <!-- ── Search & Filter ──────────────────────────────────────────── -->
    <div class="flex gap-2 mb-4">
      <input
        type="search"
        placeholder="Search title or artist…"
        x-model="searchQuery"
        @input="onSearchInput()"
        class="flex-1 bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-violet-500"
      />
      <button
        @click="filterOpen = !filterOpen"
        class="relative inline-flex items-center gap-1.5 bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-700 transition-colors focus-visible:ring-2 focus-visible:ring-violet-500"
        aria-label="Toggle filters"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 4h18M7 8h10M10 12h4"/>
        </svg>
        Filters
        <span x-show="activeFilterCount > 0" x-text="activeFilterCount"
          class="absolute -top-1.5 -right-1.5 bg-violet-600 text-white text-xs rounded-full w-4 h-4 flex items-center justify-center font-bold"></span>
      </button>
    </div>

    <!-- ── Filter panel ──────────────────────────────────────────────── -->
    <div x-show="filterOpen" x-transition class="bg-zinc-800 border border-zinc-700 rounded-lg p-4 mb-4">
      <div class="flex flex-wrap gap-2 mb-3">
        <button
          @click="toggleFavoritesFilter()"
          :class="favoritesOnly ? 'bg-pink-600 text-white border-pink-500' : 'bg-zinc-700 text-zinc-300 border-zinc-600 hover:bg-zinc-600'"
          class="px-3 py-1 rounded-full text-xs border font-medium transition-colors focus-visible:ring-2 focus-visible:ring-pink-500 inline-flex items-center gap-1"
        >
          <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24"><path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/></svg>
          Favorites only
        </button>
        ${FORMATS.map(({ value, label }) => /* html */`
        <button
          data-format="${value}"
          @click="toggleFormat($el.dataset.format)"
          :class="selectedFormats.includes($el.dataset.format) ? 'bg-violet-600 text-white border-violet-500' : 'bg-zinc-700 text-zinc-300 border-zinc-600 hover:bg-zinc-600'"
          class="px-3 py-1 rounded-full text-xs border font-medium transition-colors focus-visible:ring-2 focus-visible:ring-violet-500"
        >${label}</button>
        `).join('')}
      </div>
      <button @click="clearFilters()" class="text-xs text-zinc-400 hover:text-zinc-100 transition-colors focus-visible:ring-2 focus-visible:ring-violet-500 rounded">
        Clear all filters
      </button>
    </div>

    <!-- ── Loading skeletons ─────────────────────────────────────────── -->
    <template x-if="loading && records.length === 0">
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        ${Array(8).fill(/* html */`
        <div class="bg-zinc-800 rounded-xl overflow-hidden animate-pulse">
          <div class="aspect-square bg-zinc-700"></div>
          <div class="p-3 space-y-2">
            <div class="h-3 bg-zinc-700 rounded w-3/4"></div>
            <div class="h-3 bg-zinc-700 rounded w-1/2"></div>
          </div>
        </div>
        `).join('')}
      </div>
    </template>

    <!-- ── Error state ───────────────────────────────────────────────── -->
    <template x-if="error && records.length === 0">
      <div class="flex flex-col items-center justify-center py-24 text-center gap-4">
        <p class="text-zinc-400" x-text="'Error: ' + error"></p>
        <button @click="load(true)" class="bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-violet-400">
          Retry
        </button>
      </div>
    </template>

    <!-- ── Empty state ───────────────────────────────────────────────── -->
    <template x-if="!loading && !error && records.length === 0">
      <div class="flex flex-col items-center justify-center py-24 text-center gap-6">
        <template x-if="favoritesOnly">
          <div class="flex flex-col items-center gap-6">
            <svg class="w-20 h-20 text-zinc-700" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
            </svg>
            <div>
              <p class="text-zinc-300 font-medium text-lg">No favorites yet</p>
              <p class="text-zinc-500 text-sm mt-1">Click the heart on any record to add it here</p>
            </div>
            <button @click="toggleFavoritesFilter()" class="bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 text-sm font-medium px-4 py-2 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-violet-500">
              Show all records
            </button>
          </div>
        </template>
        <template x-if="!favoritesOnly">
          <div class="flex flex-col items-center gap-6">
            <svg class="w-20 h-20 text-zinc-700" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <circle cx="12" cy="12" r="10"/>
              <circle cx="12" cy="12" r="4"/>
              <line x1="12" y1="2" x2="12" y2="8"/>
            </svg>
            <div>
              <p class="text-zinc-300 font-medium text-lg">Your collection is empty</p>
              <p class="text-zinc-500 text-sm mt-1">Start by adding your first vinyl record</p>
            </div>
            <button
              @click="addRecord()"
              class="bg-violet-600 hover:bg-violet-500 text-white font-medium px-6 py-2.5 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-violet-400"
            >Add your first record</button>
          </div>
        </template>
      </div>
    </template>

    <!-- ── Record grid ───────────────────────────────────────────────── -->
    <template x-if="records.length > 0">
      <div>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <template x-for="record in records" :key="record.id">
            <div class="group bg-zinc-800 rounded-xl overflow-hidden hover:ring-2 hover:ring-violet-500/50 transition-all">
              <!-- Cover art -->
              <div class="aspect-square bg-zinc-700 relative overflow-hidden">
                <template x-if="record.coverArtUrl">
                  <img
                    :src="record.coverArtUrl"
                    :alt="record.title + ' cover art'"
                    class="w-full h-full object-cover"
                    loading="lazy"
                  />
                </template>
                <template x-if="!record.coverArtUrl">
                  <div class="w-full h-full flex items-center justify-center">
                    <svg class="w-12 h-12 text-zinc-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                      <circle cx="12" cy="12" r="10"/>
                      <circle cx="12" cy="12" r="3"/>
                    </svg>
                  </div>
                </template>

                <!-- Action overlay -->
                <div class="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-3">
                  <button
                    @click.stop="editRecord(record.id)"
                    class="p-2 bg-zinc-800 rounded-full hover:bg-zinc-700 transition-colors focus-visible:ring-2 focus-visible:ring-violet-500"
                    aria-label="Edit record"
                  >
                    <svg class="w-4 h-4 text-zinc-100" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                    </svg>
                  </button>
                  <button
                    @click.stop="confirmDelete(record)"
                    class="p-2 bg-zinc-800 rounded-full hover:bg-red-700 transition-colors focus-visible:ring-2 focus-visible:ring-red-500"
                    aria-label="Delete record"
                  >
                    <svg class="w-4 h-4 text-zinc-100" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                    </svg>
                  </button>
                </div>
              </div>

              <!-- Info -->
              <div class="p-3">
                <div class="flex items-start justify-between gap-1">
                  <div class="min-w-0">
                    <p class="font-medium text-sm truncate text-zinc-100" x-text="record.title"></p>
                    <p class="text-xs text-zinc-400 truncate" x-text="record.artist + (record.year ? ' · ' + record.year : '')"></p>
                  </div>
                  <button
                    @click.stop="toggleFavorite(record)"
                    :class="record.favorite ? 'text-pink-400' : 'text-zinc-600 hover:text-pink-400'"
                    class="flex-shrink-0 p-0.5 transition-colors focus-visible:ring-2 focus-visible:ring-pink-500 rounded"
                    :aria-label="record.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  >
                    <svg class="w-4 h-4" :fill="record.favorite ? 'currentColor' : 'none'" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
                    </svg>
                  </button>
                </div>
                <div class="flex gap-1.5 mt-2 flex-wrap">
                  <template x-if="record.format">
                    <span class="px-1.5 py-0.5 bg-zinc-700 text-zinc-300 text-xs rounded" x-text="record.format"></span>
                  </template>
                  <template x-if="record.condition">
                    <span class="px-1.5 py-0.5 bg-violet-900/50 text-violet-300 text-xs rounded" x-text="record.condition"></span>
                  </template>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Load more -->
        <div class="mt-8 flex justify-center" x-show="hasMore">
          <button
            @click="loadMore()"
            :disabled="loading"
            class="bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 text-sm font-medium px-6 py-2 rounded-lg transition-colors disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-violet-500"
          >
            <span x-show="!loading">Load more</span>
            <span x-show="loading">Loading…</span>
          </button>
        </div>
      </div>
    </template>

    <!-- ── Delete confirmation modal ─────────────────────────────────── -->
    <div
      x-show="showDeleteModal"
      x-transition.opacity
      class="fixed inset-0 bg-black/70 z-50 flex items-end md:items-center justify-center p-4"
      @keydown.escape.window="cancelDelete()"
      role="dialog"
      aria-modal="true"
      :aria-label="'Delete ' + (deleteTarget?.title ?? 'record')"
    >
      <div class="bg-zinc-900 border border-zinc-700 rounded-2xl p-6 w-full max-w-sm shadow-2xl">
        <h2 class="text-lg font-semibold mb-2">Delete record?</h2>
        <p class="text-sm text-zinc-400 mb-6">
          "<span x-text="deleteTarget?.title"></span>" will be permanently deleted.
        </p>
        <div class="flex gap-3 justify-end">
          <button
            @click="cancelDelete()"
            class="px-4 py-2 text-sm bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-zinc-500"
          >Cancel</button>
          <button
            @click="doDelete()"
            class="px-4 py-2 text-sm bg-red-700 hover:bg-red-600 text-white rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-red-500"
          >Delete</button>
        </div>
      </div>
    </div>
  </div>
  `
}
