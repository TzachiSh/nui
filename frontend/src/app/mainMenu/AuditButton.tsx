import AuditIcon from "@/icons/AuditIcon"
import { deckCardsSo } from "@/stores/docs/cards"
import { buildStore } from "@/stores/docs/utils/factory"
import { DOC_TYPE } from "@/types"
import { focusSo } from "@priolo/jack"
import { FunctionComponent } from "react"
import MenuButton from "./MenuButton"



interface Props {
}

const AuditButton: FunctionComponent<Props> = ({
}) => {

	// STORE

	// HOOKs

	// HANDLER
	const handleClick = async () => {
		const store = buildStore({ type: DOC_TYPE.AUDIT })
		await deckCardsSo.add({ view: store, anim: true })
		focusSo.focus(store)
	}

	// RENDER
	return (
		<MenuButton
			title="Audit Logs"
			subtitle="AUDIT"
			onClick={handleClick}
		>
			<AuditIcon style={{ width: 20 }} className="color-fg" />
		</MenuButton>
	)
}

export default AuditButton
