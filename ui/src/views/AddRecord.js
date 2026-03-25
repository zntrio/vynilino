import Alpine from 'alpinejs'
import { gql } from '../lib/gql.js'
import { router } from '../router.js'
import { renderShell } from '../components/AppShell.js'
import { recordFormFields, emptyForm, parseGenres } from '../components/RecordForm.js'
import { discogsSearchData, discogsSearchHTML } from '../components/DiscogsSearch.js'

const CREATE_MUTATION = /* graphql */`
  mutation CreateRecord($input: CreateRecordInput!) {
    createRecord(input: $input) {
      record { id title }
      duplicateWarning
    }
  }
`

const MAX_UPLOAD_BYTES = 5 * 1024 * 1024

export function AddRecord(container) {
  renderShell(container, 'collection', () => '')

  const view = document.getElementById('view')
  if (!view) return

  Alpine.data('discogsSearch', discogsSearchData)
  Alpine.data('addRecord', () => ({
    form: { ...emptyForm(), discogsId: null },
    touched: { title: false, artist: false },
    submitting: false,
    uploadState: 'idle', // 'idle' | 'uploading' | 'done' | 'error'
    uploadError: '',
    duplicateWarning: '',

    isValid() {
      return this.form.title.trim() !== '' && this.form.artist.trim() !== ''
    },

    onDiscogsSelect(event) {
      const r = event.detail
      this.form.title = r.title || ''
      this.form.artist = r.artist || ''
      this.form.year = r.year || null
      this.form.label = r.label || ''
      this.form.coverArtUrl = r.thumbUrl || ''
      this.form.discogsId = r.discogsId || null
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

      this.submitting = true
      this.duplicateWarning = ''
      try {
        const input = buildInput(this.form)
        const data = await gql(CREATE_MUTATION, { input })
        const { duplicateWarning } = data.createRecord
        if (duplicateWarning) {
          this.duplicateWarning = duplicateWarning
          Alpine.store('toast').show('Record added (duplicate warning — see below)', 'success')
          // Stay on form briefly so warning is visible, then navigate.
          setTimeout(() => router.navigate('/'), 2500)
        } else {
          Alpine.store('toast').show('Record added!')
          router.navigate('/')
        }
      } catch (e) {
        Alpine.store('toast').show('Error: ' + e.message, 'error')
      } finally {
        this.submitting = false
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
    ...(form.discogsId ? { discogsId: form.discogsId } : {}),
  }
}

function formHTML() {
  return /* html */`
  <div x-data="addRecord" @discogs-select="onDiscogsSelect($event)" class="max-w-lg mx-auto pb-20 md:pb-0">
    <!-- Header -->
    <div class="flex items-center gap-3 mb-6">
      <button
        @click="cancel()"
        class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors focus-visible:ring-2 focus-visible:ring-violet-500"
        aria-label="Back to collection"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
        </svg>
      </button>
      <h1 class="text-xl font-bold tracking-tight">Add Record</h1>
    </div>

    <!-- Discogs search panel -->
    ${discogsSearchHTML()}

    <!-- Duplicate warning banner -->
    <div
      x-show="duplicateWarning"
      x-transition
      class="mb-4 p-3 bg-amber-900/40 border border-amber-700 rounded-lg text-sm text-amber-300"
      x-text="duplicateWarning"
    ></div>

    <!-- Form -->
    <form @submit.prevent="submit()" class="space-y-4" novalidate>
      ${recordFormFields()}

      <!-- Actions -->
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
          <span x-show="!submitting">Add Record</span>
          <span x-show="submitting">Saving…</span>
        </button>
      </div>
    </form>
  </div>
  `
}
