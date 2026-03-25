/**
 * Shared record form fields HTML (used by both AddRecord and EditRecord).
 * Returns an HTML string to be injected inside a form element.
 */
export function recordFormFields() {
  return /* html */`
    <!-- Title (required) -->
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-title">Title <span class="text-red-400">*</span></label>
      <input
        id="f-title"
        type="text"
        x-model="form.title"
        @blur="touched.title = true"
        placeholder="e.g. Kind of Blue"
        class="w-full bg-zinc-800 border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
        :class="touched.title && !form.title.trim() ? 'border-red-500' : 'border-zinc-700'"
      />
      <p x-show="touched.title && !form.title.trim()" class="text-xs text-red-400 mt-1">Title is required</p>
    </div>

    <!-- Artist (required) -->
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-artist">Artist <span class="text-red-400">*</span></label>
      <input
        id="f-artist"
        type="text"
        x-model="form.artist"
        @blur="touched.artist = true"
        placeholder="e.g. Miles Davis"
        class="w-full bg-zinc-800 border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
        :class="touched.artist && !form.artist.trim() ? 'border-red-500' : 'border-zinc-700'"
      />
      <p x-show="touched.artist && !form.artist.trim()" class="text-xs text-red-400 mt-1">Artist is required</p>
    </div>

    <!-- Year & Label row -->
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-year">Year</label>
        <input
          id="f-year"
          type="number"
          x-model.number="form.year"
          min="1900" max="2100"
          placeholder="1959"
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
        />
      </div>
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-label">Label</label>
        <input
          id="f-label"
          type="text"
          x-model="form.label"
          placeholder="Columbia"
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
        />
      </div>
    </div>

    <!-- Format & Condition row -->
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-format">Format</label>
        <select
          id="f-format"
          x-model="form.format"
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
        >
          <option value="">— Select —</option>
          <option value="LP">LP</option>
          <option value="EP">EP</option>
          <option value="SINGLE">Single</option>
          <option value="SEVEN_INCH">7"</option>
          <option value="TEN_INCH">10"</option>
          <option value="TWELVE_INCH">12"</option>
        </select>
      </div>
      <div>
        <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-condition">Condition</label>
        <select
          id="f-condition"
          x-model="form.condition"
          class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
        >
          <option value="">— Select —</option>
          <option value="MINT">Mint</option>
          <option value="NEAR_MINT">Near Mint</option>
          <option value="VERY_GOOD_PLUS">Very Good Plus</option>
          <option value="VERY_GOOD">Very Good</option>
          <option value="GOOD_PLUS">Good Plus</option>
          <option value="GOOD">Good</option>
          <option value="FAIR">Fair</option>
          <option value="POOR">Poor</option>
        </select>
      </div>
    </div>

    <!-- Genre -->
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-genre">Genre <span class="text-zinc-600 font-normal">(comma-separated)</span></label>
      <input
        id="f-genre"
        type="text"
        x-model="form.genreRaw"
        placeholder="Jazz, Bebop"
        class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
      />
    </div>

    <!-- Notes -->
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1" for="f-notes">Notes</label>
      <textarea
        id="f-notes"
        x-model="form.notes"
        rows="2"
        placeholder="Optional notes…"
        class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500 resize-none"
      ></textarea>
    </div>

    <!-- Cover art upload -->
    <div>
      <label class="block text-xs font-medium text-zinc-400 mb-1">Cover Art</label>
      <div class="flex items-start gap-4">
        <div class="w-20 h-20 bg-zinc-700 rounded-lg overflow-hidden shrink-0 flex items-center justify-center">
          <template x-if="form.coverArtUrl">
            <img :src="form.coverArtUrl" alt="Cover preview" class="w-full h-full object-cover" />
          </template>
          <template x-if="!form.coverArtUrl">
            <svg class="w-8 h-8 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <circle cx="8.5" cy="8.5" r="1.5"/>
              <polyline points="21 15 16 10 5 21"/>
            </svg>
          </template>
        </div>
        <div class="flex-1">
          <label
            class="inline-flex items-center gap-2 cursor-pointer bg-zinc-800 border border-zinc-700 hover:bg-zinc-700 text-zinc-300 text-sm px-3 py-2 rounded-lg transition-colors focus-within:ring-2 focus-within:ring-violet-500"
            aria-label="Upload cover art"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
            </svg>
            <span x-text="uploadState === 'uploading' ? 'Uploading…' : 'Choose image'"></span>
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              class="sr-only"
              @change="onFileChange($event)"
              :disabled="uploadState === 'uploading'"
            />
          </label>
          <p x-show="uploadError" x-text="uploadError" class="text-xs text-red-400 mt-1"></p>
          <p class="text-xs text-zinc-500 mt-1">JPEG, PNG or WebP · max 5 MB</p>
        </div>
      </div>
    </div>
  `
}

/** Returns the base form state object. */
export function emptyForm() {
  return {
    title: '',
    artist: '',
    year: null,
    label: '',
    format: '',
    condition: '',
    genreRaw: '',
    notes: '',
    coverArtUrl: '',
    favorite: false,
    personalNote: '',
  }
}

/** Parses genres from the comma-separated raw string. */
export function parseGenres(raw) {
  return raw.split(',').map(g => g.trim()).filter(Boolean)
}
