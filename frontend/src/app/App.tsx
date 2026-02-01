//import srcBg from "@/assets/bg4.jpg"
import MainMenu from "@/app/mainMenu/MainMenu"
import docsSo from "@/stores/docs"
import authSo from "@/stores/auth"
import { ProtobufSchemaProvider } from "@/contexts/ProtobufSchemaContext"
import { useStore } from "@priolo/jon"
import { FunctionComponent, useEffect } from "react"
import cls from "./App.module.css"
import DeckGroup from "./DeckGroup"
import DrawerGroup from "./DrawerGroup"
import ZenCard from "./ZenCard"
import LoginPage from "./LoginPage"
import { TooltipCmp, DragCmp } from "@priolo/jack"
import layoutSo from "@/stores/layout"



const App: FunctionComponent = () => {

	// STORES
	const docsSa = useStore(docsSo)
	const authSa = useStore(authSo)
	useStore(layoutSo)

	// HOOKS
	useEffect(() => {
		authSo.checkAuth()
	}, [])

	// HANDLERS

	// RENDER
	// Show loading while checking auth
	if (!authSa.initialized) {
		return (
			<div className={`${cls.root} ${cls[layoutSo.state.theme]}`}>
				<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
					<span style={{ color: 'var(--text)', opacity: 0.6 }}>Loading...</span>
				</div>
			</div>
		)
	}

	// Show login page if not authenticated
	if (!authSo.isAuthenticated()) {
		return <LoginPage />
	}

	const clsContent = `${cls.content} ${cls[docsSa.drawerPosition]}`

	return (
		<ProtobufSchemaProvider>
			<div className={`${cls.root} ${cls[layoutSo.state.theme]}`}>

				<ZenCard />

				<MainMenu />

				<div className={clsContent}>
					<DeckGroup />
					<DrawerGroup />
				</div>

				<DragCmp />
				<TooltipCmp />
			</div>
		</ProtobufSchemaProvider>
	)
}

export default App
