import Alpine from 'alpinejs'
import { gql } from '../lib/gql.js'
import { router } from '../router.js'

const LOGIN_MUTATION = /* graphql */`
  mutation Login($email: String!, $password: String!) {
    login(email: $email, password: $password) {
      accessToken
      user { id email }
    }
  }
`

export function Login(container) {
  Alpine.data('loginForm', () => ({
    email: '',
    password: '',
    error: '',
    loading: false,

    async submit() {
      this.error = ''
      this.loading = true
      try {
        const data = await gql(LOGIN_MUTATION, { email: this.email, password: this.password })
        Alpine.store('auth').setToken(data.login.accessToken)
        Alpine.store('auth').user = data.login.user
        router.navigate('/')
      } catch (e) {
        this.error = 'Invalid email or password'
      } finally {
        this.loading = false
      }
    },
  }))

  container.innerHTML = /* html */`
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
        </form>
      </div>
    </div>
  </div>
  `
  Alpine.initTree(container)
}
