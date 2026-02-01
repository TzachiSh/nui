import authSo from "@/stores/auth"
import { FunctionComponent } from "react"
import cls from "./LoginPage.module.css"
import layoutSo from "@/stores/layout"
import { useStore } from "@priolo/jon"

const LoginPage: FunctionComponent = () => {
	useStore(layoutSo)

	const handleLogin = () => {
		authSo.login()
	}

	return (
		<div className={`${cls.root} ${cls[layoutSo.state.theme]}`}>
			<div className={cls.container}>
				<div className={cls.logo}>
					<svg width="80" height="80" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
						<circle cx="50" cy="50" r="45" stroke="currentColor" strokeWidth="4" />
						<text x="50" y="60" textAnchor="middle" fill="currentColor" fontSize="32" fontWeight="bold">N</text>
					</svg>
				</div>
				<h1 className={cls.title}>NUI</h1>
				<p className={cls.subtitle}>NATS Management Interface</p>

				<button className={cls.loginButton} onClick={handleLogin}>
					<svg className={cls.jumpcloudIcon} viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
						<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
					</svg>
					Sign in with JumpCloud
				</button>

				<p className={cls.footer}>
					Secure single sign-on powered by JumpCloud
				</p>
			</div>
		</div>
	)
}

export default LoginPage
