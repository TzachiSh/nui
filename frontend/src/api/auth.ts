import ajax, { CallOptions } from "@/plugins/AjaxService"

export interface AuthUser {
	id: string
	email: string
	name: string
}

export interface AuthStatus {
	authenticated: boolean
	authEnabled?: boolean
	user?: AuthUser
}

/** Get current auth status */
function me(opt?: CallOptions): Promise<AuthStatus> {
	return ajax.get(`auth/me`, null, { ...opt, noError: true })
}

/** Redirect to login */
function login(): void {
	window.location.href = "/auth/login"
}

/** Logout */
function logout(): void {
	window.location.href = "/auth/logout"
}

const authApi = {
	me,
	login,
	logout,
}

export default authApi
