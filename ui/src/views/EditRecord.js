import Alpine from '@alpinejs/csp'
import { gql } from '../lib/gql.js'
import { router } from '../router.js'
import { renderShell } from '../components/AppShell.js'
import { recordFormFields, emptyForm, parseGenres } from '../components/RecordForm.js'

const GET_RECORD = /* graphql */`
  query GetRecord($id: ID!) {
    record(id: $id) {
      id title artist year label format condition genres notes coverArtUrl favorite personalNote
    }
  }
`

const UPDATE_MUTATION = /* graphql */`
  mutation UpdateRecord($id: ID!, $input: UpdateRecordInput!) {
    updateRecord(id: $id, input: $input) {
      id title artist year format condition coverArtUrl favorite personalNote
    }
  }
`

const DELETE_MUTATION = /* graphql */`
  mutation DeleteRecord($id: ID!) { deleteRecord(id: $id) }
`

const MAX_UPLOAD_BYTES = 5 * 1024 * 1024

export function EditRecord(container, { id }) {
  renderShell(container, 'collection', () => '')

  const view = document.getElementById('view')
  if (!view) return

  Alpine.data('editRecord', () => ({
    recordId: id,
    form: emptyForm(),
    originalForm: null,
    touched: { title: false, artist: false },
    loading: true,
    submitting: false,
    uploadState: 'idle',
    uploadError: '',
    showDeleteModal: false,

    async init() {
      try {
        const data = await gql(GET_RECORD, { id })
        const r = data.record
        if (!r) { router.navigate('/'); return }
        this.form = {
          title: r.title ?? '',
          artist: r.artist ?? '',
          year: r.year ?? null,
          label: r.label ?? '',
          format: r.format ?? '',
          condition: r.condition ?? '',
          genreRaw: (r.genres ?? []).join(', '),
          notes: r.notes ?? '',
          coverArtUrl: r.coverArtUrl ?? '',
          favorite: r.favorite ?? false,
          personalNote: r.personalNote ?? '',
        }
        this.originalForm = { ...this.form }
      } catch (e) {
        Alpine.store('toast').show('Failed to load record: ' + e.message, 'error')
        router.navigate('/')
      } finally {
        this.loading = false
      }
    },

    isValid() {
      return this.form.title.trim() !== '' && this.form.artist.trim() !== ''
    },

    async onFileChange(event) {
      const file = event.target.files?.[0]
      if (!file) return
      if (file.size > MAX_UPLOAD_BYTES) {
        this.uploadError = 'File is too large (max 5 MB)'
        return
      }
      this.uploadError = ''
      this.uploadState = 'uploading'
      try {
        const fd = new FormData()
        fd.append('file', file)
        fd.append('recordId', this.recordId)
        const res = await fetch('/api/upload', {
          method: 'POST',
          headers: { Authorization: `Bearer ${Alpine.store('auth').token}` },
          body: fd,
        })
        if (!res.ok) throw new Error(`Upload failed: ${res.status}`)
        const json = await res.json()
        this.form.coverArtUrl = json.url
        this.uploadState = 'done'
      } catch (e) {
        this.uploadError = e.message
        this.uploadState = 'error'
      }
    },

    async submit() {
      this.touched = { title: true, artist: true }
      if (!this.isValid()) return

      // Optimistic update: cache current form and navigate away immediately.
      const optimisticValues = { ...this.form }
      this.submitting = true
      try {
        const input = buildInput(this.form)
        await gql(UPDATE_MUTATION, { id: this.recordId, input })
        Alpine.store('toast').show('Record updated!')
        router.navigate('/')
      } catch (e) {
        // Revert on error.
        this.form = { ...optimisticValues }
        Alpine.store('toast').show('Update failed: ' + e.message, 'error')
      } finally {
        this.submitting = false
      }
    },

    confirmDelete() {
      this.showDeleteModal = true
    },

    cancelDelete() {
      this.showDeleteModal = false
    },

    async doDelete() {
      this.showDeleteModal = false
      try {
        await gql(DELETE_MUTATION, { id: this.recordId })
        Alpine.store('toast').show('Record deleted')
        router.navigate('/')
      } catch (e) {
        Alpine.store('toast').show('Delete failed: ' + e.message, 'error')
      }
    },

    cancel() {
      router.navigate('/')
    },
  }))

  view.innerHTML = formHTML()
  Alpine.initTree(view)
}

function buildInput(form) {
  return {
    title: form.title.trim(),
    artist: form.artist.trim(),
    ...(form.year ? { year: form.year } : {}),
    ...(form.label ? { label: form.label.trim() } : {}),
    ...(form.format ? { format: form.format } : {}),
    ...(form.condition ? { condition: form.condition } : {}),
    ...(form.genreRaw ? { genres: parseGenres(form.genreRaw) } : {}),
    ...(form.notes ? { notes: form.notes.trim() } : {}),
    ...(form.coverArtUrl ? { coverArtUrl: form.coverArtUrl } : {}),
    favorite: form.favorite,
    personalNote: form.personalNote.trim() || null,
  }
}

function formHTML() {
  return /* html */`
  <div x-data="editRecord" x-init="init()" class="max-w-lg mx-auto pb-20 md:pb-0">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <button
          @click="cancel()"
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors focus-visible:ring-2 focus-visible:ring-violet-500"
          aria-label="Back to collection"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
          </svg>
        </button>
        <h1 class="text-xl font-bold tracking-tight">Edit Record</h1>
      </div>
      <button
        @click="confirmDelete()"
        class="p-2 rounded-lg text-zinc-400 hover:text-red-400 hover:bg-zinc-800 transition-colors focus-visible:ring-2 focus-visible:ring-red-500"
        aria-label="Delete record"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
        </svg>
      </button>
    </div>

    <!-- Loading skeleton -->
    <template x-if="loading">
      <div class="space-y-4 animate-pulse">
        <div class="h-10 bg-zinc-800 rounded-lg"></div>
        <div class="h-10 bg-zinc-800 rounded-lg"></div>
        <div class="grid grid-cols-2 gap-3">
          <div class="h-10 bg-zinc-800 rounded-lg"></div>
          <div class="h-10 bg-zinc-800 rounded-lg"></div>
        </div>
      </div>
    </template>

    <!-- Form -->
    <template x-if="!loading">
      <form @submit.prevent="submit()" class="space-y-4" novalidate>
        ${recordFormFields()}

        <!-- Favorite toggle -->
        <label class="flex items-center gap-3 cursor-pointer select-none">
          <input
            type="checkbox"
            x-model="form.favorite"
            class="w-4 h-4 rounded border-zinc-600 bg-zinc-800 text-pink-500 focus:ring-pink-500 focus:ring-offset-zinc-900 cursor-pointer"
          />
          <span class="text-sm text-zinc-300 inline-flex items-center gap-1.5">
            <svg class="w-4 h-4 text-pink-400" :fill="form.favorite ? 'currentColor' : 'none'" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"/>
            </svg>
            Mark as favorite
          </span>
        </label>

        <!-- Personal note -->
        <div>
          <label class="block text-sm font-medium text-zinc-300 mb-1">Personal note</label>
          <textarea
            x-model="form.personalNote"
            rows="3"
            placeholder="Private thoughts about this record…"
            class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-violet-500 resize-none"
          ></textarea>
          <p class="text-xs text-zinc-500 mt-1">Only visible to you. Not included in exports.</p>
        </div>

        <div class="flex gap-3 pt-2">
          <button
            type="button"
            @click="cancel()"
            class="flex-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 text-sm font-medium py-2.5 rounded-lg transition-colors focus-visible:ring-2 focus-visible:ring-zinc-500"
          >Cancel</button>
          <button
            type="submit"
            :disabled="submitting"
            class="flex-1 bg-violet-600 hover:bg-violet-500 text-white text-sm font-medium py-2.5 rounded-lg transition-colors disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-violet-400"
          >
            <span x-show="!submitting">Save Changes</span>
            <span x-show="submitting">Saving…</span>
          </button>
        </div>
      </form>
    </template>

    <!-- Delete confirmation modal -->
    <div
      x-show="showDeleteModal"
      x-transition.opacity
      class="fixed inset-0 bg-black/70 z-50 flex items-end md:items-center justify-center p-4"
      @keydown.escape.window="cancelDelete()"
      role="dialog"
      aria-modal="true"
      aria-label="Delete record"
    >
      <div class="bg-zinc-900 border border-zinc-700 rounded-2xl p-6 w-full max-w-sm shadow-2xl">
        <h2 class="text-lg font-semibold mb-2">Delete record?</h2>
        <p class="text-sm text-zinc-400 mb-6">
          "<span x-text="form.title"></span>" will be permanently deleted.
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
