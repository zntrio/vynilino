import Alpine from '@alpinejs/csp'

const LOGIN_MUTATION = /* graphql */`
  mutation Login($email: String!, $password: String!) {
    login(email: $email, password: $password) {
      accessToken
      user { id email }
    }
  }
`

async function gqlLogin(email, password) {
  const res = await fetch('/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query: LOGIN_MUTATION, variables: { email, password } }),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const json = await res.json()
  if (json.errors?.length) throw new Error(json.errors[0].message)
  return json.data
}

Alpine.store('auth', {
  setToken(token) {
    if (token) {
      localStorage.setItem('vynilino_token', token)
    } else {
      localStorage.removeItem('vynilino_token')
    }
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

const _oidcErrors = {
  auth_failed: 'Authentication failed. Please try again.',
  oidc_not_configured: 'Single sign-on is not configured.',
  oidc_unavailable: 'SSO provider is unavailable.',
  access_denied: 'Access was denied by the identity provider.',
}

Alpine.data('loginForm', () => {
  const _raw = new URLSearchParams(window.location.search).get('error') ?? ''
  return {
  email: '',
  password: '',
  error: _oidcErrors[_raw] ?? (_raw ? `Login error: ${_raw}` : ''),
  loading: false,

  async submit() {
    this.error = ''
    this.loading = true
    try {
      const data = await gqlLogin(this.email, this.password)
      Alpine.store('auth').setToken(data.login.accessToken)
      window.location.href = '/'
    } catch (e) {
      this.error = 'Invalid email or password'
    } finally {
      this.loading = false
    }
  },
}})

document.getElementById('login-app').innerHTML = /* html */`
<div class="min-h-screen flex items-center justify-center p-4 bg-zinc-950">
  <div class="w-full max-w-sm" x-data="loginForm">
    <div class="flex items-center justify-center gap-2 mb-8">
      <svg class="w-8 h-8 text-violet-400" fill="currentColor" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
        <circle cx="12" cy="12" r="3"/>
      </svg>
      <span class="text-2xl font-bold tracking-tight text-zinc-100">Vynilino</span>
    </div>

    <div class="bg-zinc-900 border border-zinc-800 rounded-2xl p-6 shadow-xl">
      <h1 class="text-lg font-semibold mb-5 text-center">Sign in to your collection</h1>

      <form @submit.prevent="submit()" novalidate class="space-y-4">
        <div>
          <label class="block text-xs font-medium text-zinc-400 mb-1" for="login-email">Email</label>
          <input
            id="login-email"
            type="email"
            x-model="email"
            autocomplete="email"
            required
            class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 placeholder-zinc-500"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-zinc-400 mb-1" for="login-password">Password</label>
          <input
            id="login-password"
            type="password"
            x-model="password"
            autocomplete="current-password"
            required
            class="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
          />
        </div>

        <p x-show="error" x-text="error" class="text-sm text-red-400 text-center"></p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-violet-600 hover:bg-violet-500 text-white font-medium py-2.5 rounded-lg transition-colors disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-violet-400"
        >
          <span x-show="!loading">Sign in</span>
          <span x-show="loading">Signing in…</span>
        </button>

        <div class="text-center mt-2">
          <a href="/oidc/authorize" class="text-sm text-violet-400 hover:text-violet-300 underline">
            Sign in with SSO
          </a>
        </div>
      </form>
    </div>
  </div>
</div>
`

Alpine.start()
