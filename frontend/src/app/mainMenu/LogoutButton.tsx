import LogoutIcon from "@/icons/LogoutIcon"
import authSo from "@/stores/auth"
import { FunctionComponent } from "react"
import MenuButton from "./MenuButton"



interface Props {
}

const LogoutButton: FunctionComponent<Props> = ({
}) => {

	// STORE

	// HOOKs

	// HANDLER
	const handleLogout = () => {
		authSo.logout()
	}

	// RENDER
	return (
		<MenuButton
			title="Logout"
			subtitle="LOGOUT"
			onClick={handleLogout}
		>
			<LogoutIcon style={{ width: 20 }} className="color-fg" />
		</MenuButton>
	)
}

export default LogoutButton
