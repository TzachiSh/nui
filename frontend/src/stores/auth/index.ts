import authApi, { AuthUser } from "@/api/auth"
import { StoreCore, createStore } from "@priolo/jon"


const setup = {

	state: {
		initialized: false,
		authenticated: false,
		authEnabled: true,
		user: null as AuthUser | null,
	},

	getters: {
		isAuthenticated(_: void, store?: AuthStore): boolean {
			return store.state.authenticated || !store.state.authEnabled
		},
	},

	actions: {
		async checkAuth(_: void, store?: AuthStore) {
			try {
				const status = await authApi.me()
				store.setAuthenticated(status.authenticated)
				store.setAuthEnabled(status.authEnabled !== false)
				store.setUser(status.user || null)
			} catch (e) {
				store.setAuthenticated(false)
				store.setUser(null)
			} finally {
				store.setInitialized(true)
			}
		},

		login(_: void, _store?: AuthStore) {
			authApi.login()
		},

		logout(_: void, _store?: AuthStore) {
			authApi.logout()
		},
	},

	mutators: {
		setInitialized: (initialized: boolean) => ({ initialized }),
		setAuthenticated: (authenticated: boolean) => ({ authenticated }),
		setAuthEnabled: (authEnabled: boolean) => ({ authEnabled }),
		setUser: (user: AuthUser | null) => ({ user }),
	},
}

export type AuthState = typeof setup.state
export type AuthGetters = typeof setup.getters
export type AuthActions = typeof setup.actions
export type AuthMutators = typeof setup.mutators
export interface AuthStore extends StoreCore<AuthState>, AuthGetters, AuthActions, AuthMutators {
	state: AuthState
}
const authSo = createStore(setup) as AuthStore
export default authSo
