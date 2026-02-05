import { socketPool } from "@/plugins/SocketService/pool"
import { MSG_TYPE } from "@/plugins/SocketService/types"
import metricsSo from "@/stores/connections/metrics"
import { ConsumerStore } from "@/stores/stacks/consumer/detail"
import { ConnzConnection } from "@/types/Metrics"
import { compactByte, compactNumber } from "@/utils/conversion"
import { getDeltaTime } from "@/utils/timeUtils"
import { TooltipWrapCmp } from "@priolo/jack"
import { useStore } from "@priolo/jon"
import dayjs from "dayjs"
import { FunctionComponent, useEffect, useState } from "react"


interface Props {
	store: ConsumerStore
}

const PAGE_SIZE = 10

const ClientInfoSection: FunctionComponent<Props> = ({ store }) => {
	const state = useStore(store)
	useStore(metricsSo)

	const [clients, setClients] = useState<ConnzConnection[]>([])
	const [error, setError] = useState<string | null>(null)
	const [loading, setLoading] = useState(true)
	const [page, setPage] = useState(0)

	const connectionId = state.connectionId
	const consumer = state.consumer

	useEffect(() => {
		if (!connectionId || !consumer) return

		// Enable metrics to cache connection data
		metricsSo.enable(connectionId)

		// Set up listener for consumer clients response
		const socketId = `global::${connectionId}`
		const ss = socketPool.getById(socketId)
		if (!ss) {
			setError("Socket not available")
			setLoading(false)
			return
		}

		const onConsumerClientsResp = (message: any) => {
			const payload = message.payload
			if (payload.stream_name !== consumer.streamName || payload.consumer_name !== consumer.name) {
				return
			}
			if (payload.error) {
				setError(payload.error)
				setClients([])
			} else {
				setError(null)
				setClients(payload.clients || [])
			}
			setLoading(false)
		}

		ss.emitter.on(MSG_TYPE.CONSUMER_CLIENTS_RESP, onConsumerClientsResp)

		// Request consumer clients
		const requestClients = () => {
			const deliverSubject = consumer.config?.deliverSubject || ""
			const filterSubject = consumer.config?.filterSubject || consumer.config?.filterSubjects?.[0] || ""

			ss.send(JSON.stringify({
				type: MSG_TYPE.CONSUMER_CLIENTS_REQ,
				payload: {
					stream_name: consumer.streamName,
					consumer_name: consumer.name,
					deliver_subject: deliverSubject,
					filter_subject: filterSubject,
				}
			}))
		}

		// Initial request after a short delay to allow metrics to populate
		const initialTimer = setTimeout(requestClients, 1500)

		// Periodic refresh
		const intervalId = setInterval(requestClients, 5000)

		return () => {
			clearTimeout(initialTimer)
			clearInterval(intervalId)
			ss.emitter.off(MSG_TYPE.CONSUMER_CLIENTS_RESP, onConsumerClientsResp)
		}
	}, [connectionId, consumer?.streamName, consumer?.name])

	// Pagination
	const totalPages = Math.ceil(clients.length / PAGE_SIZE)
	const paginatedClients = clients.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)

	if (loading) {
		return (
			<div style={{ padding: "10px", color: "#888" }}>
				Loading client information...
			</div>
		)
	}

	if (error) {
		return (
			<div style={{ padding: "10px", color: "#f59e0b" }}>
				{error}
			</div>
		)
	}

	if (clients.length === 0) {
		return (
			<div style={{ padding: "10px" }}>
				<div style={{ color: "#888", marginBottom: "8px" }}>
					No active client connections found for this consumer.
				</div>
				<div style={{ fontSize: "11px", color: "#666" }}>
					Clients are matched by subscription to the consumer's delivery or filter subject.
					Make sure clients are connected and subscribed to receive messages from this consumer.
				</div>
			</div>
		)
	}

	return (
		<div>
			{/* Summary */}
			<div style={{ marginBottom: "10px", fontSize: "12px", color: "#888" }}>
				{clients.length} client{clients.length !== 1 ? 's' : ''} connected
			</div>

			{/* Client list */}
			{paginatedClients.map((client) => (
				<ClientCard key={client.cid} client={client} />
			))}

			{/* Pagination */}
			{totalPages > 1 && (
				<div style={{ display: "flex", justifyContent: "center", gap: "10px", marginTop: "10px" }}>
					<button
						onClick={() => setPage(p => Math.max(0, p - 1))}
						disabled={page === 0}
						style={{
							padding: "4px 8px",
							backgroundColor: page === 0 ? "#333" : "#555",
							border: "none",
							borderRadius: "3px",
							color: page === 0 ? "#666" : "#fff",
							cursor: page === 0 ? "default" : "pointer",
						}}
					>
						Prev
					</button>
					<span style={{ color: "#888", fontSize: "12px", alignSelf: "center" }}>
						{page + 1} / {totalPages}
					</span>
					<button
						onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))}
						disabled={page >= totalPages - 1}
						style={{
							padding: "4px 8px",
							backgroundColor: page >= totalPages - 1 ? "#333" : "#555",
							border: "none",
							borderRadius: "3px",
							color: page >= totalPages - 1 ? "#666" : "#fff",
							cursor: page >= totalPages - 1 ? "default" : "pointer",
						}}
					>
						Next
					</button>
				</div>
			)}
		</div>
	)
}

export default ClientInfoSection


interface ClientCardProps {
	client: ConnzConnection
}

const ClientCard: FunctionComponent<ClientCardProps> = ({ client }) => {
	const startActivity = dayjs(client.start).format("YYYY-MM-DD HH:mm:ss")
	const lastActivity = dayjs(client.last_activity).format("YYYY-MM-DD HH:mm:ss")
	const [lastActivityDelta, isRecentActivity] = getDeltaTime(client.last_activity)
	const rtt = parseInt(client.rtt) + client.rtt?.slice(-2)
	const pending = compactByte(client?.pending_bytes)

	const lang = `${client.lang?.toLowerCase() ?? "--"} v${client.version ?? "--"}`

	const msgsIn = compactNumber(client?.in_msgs)
	const msgsInRate = compactNumber(client?.nui_in_msgs_sec)
	const msgsOut = compactNumber(client?.out_msgs)
	const msgsOutRate = compactNumber(client?.nui_out_msgs_sec)

	const bytesOut = compactByte(client?.out_bytes)
	const bytesOutRate = compactByte(client?.nui_out_bytes_sec)
	const bytesIn = compactByte(client?.in_bytes)
	const bytesInRate = compactByte(client?.nui_in_bytes_sec)

	return (
		<div style={{
			padding: "8px",
			marginBottom: "8px",
			border: "1px solid #333",
			borderRadius: "3px",
			backgroundColor: "#1a1a1a",
			fontSize: 12,
			fontWeight: 400,
		}}>
			{/* Identifier */}
			<div style={{ display: "flex", gap: 3, alignItems: "center" }}>
				<div style={{
					width: "8px",
					height: "8px",
					borderRadius: "50%",
					backgroundColor: isRecentActivity ? "var(--color-mint)" : "#575757ff",
					marginRight: "4px",
				}} />
				<div style={{ flex: 1, ...ellipsisStyle }}>
					<span style={{ fontWeight: 700 }}>{client.cid}</span> / {client.ip}:{client.port}
				</div>
				<div style={{
					backgroundColor: "var(--cmp-select-bg)",
					padding: "2px 4px",
					borderRadius: "2px",
					fontSize: 10,
					color: "#000"
				}}>
					{lang}
				</div>
			</div>

			{/* Name */}
			{client.name && (
				<div style={{ color: "var(--cmp-select-bg)", ...ellipsisStyle }}>{client.name}</div>
			)}

			{/* Properties */}
			<div style={{ display: "flex", flexDirection: "column", gap: "0px" }}>
				<div style={{ display: "flex", gap: "10px", flexWrap: "wrap" }}>
					<TooltipWrapCmp content={lastActivity}>
						<ValueCmp
							title="LAST ACT."
							value={lastActivityDelta}
							style={{ color: isRecentActivity ? "var(--cmp-select-bg)" : undefined }}
						/>
					</TooltipWrapCmp>
					<ValueCmp title="START" value={startActivity} />
					<ValueCmp title="UPTIME" value={client.uptime} />
					<ValueCmp title="RTT" value={rtt} />
					<ValueCmp title="SUBS." value={client.subscriptions} />
					<ValueCmp title="PENDING" value={pending.value + pending.unit} />
				</div>

				<div style={{ display: "flex", flexWrap: "wrap" }}>
					<Value2Cmp title="MESS. IN" value={msgsIn} rate={msgsInRate} />
					<Value2Cmp title="MESS. OUT" value={msgsOut} rate={msgsOutRate} />
					<Value2Cmp title="DATA IN" value={bytesIn} rate={bytesInRate} />
					<Value2Cmp title="DATA OUT" value={bytesOut} rate={bytesOutRate} />
				</div>
			</div>
		</div>
	)
}

const ellipsisStyle: React.CSSProperties = {
	overflow: "hidden",
	textOverflow: "ellipsis",
	whiteSpace: "nowrap"
}

interface ValueCmpProps {
	title: string
	value: string | number
	style?: React.CSSProperties
}

const ValueCmp: FunctionComponent<ValueCmpProps> = ({ title, value, style }) => {
	return (
		<div style={{ display: "flex", flexDirection: "column", ...style, ...ellipsisStyle }}>
			<div style={{ color: "#888", fontSize: 10 }}>{title}</div>
			<div>{value}</div>
		</div>
	)
}

interface Value2CmpProps {
	title: string
	value: { value: number; unit: string }
	rate?: { value: number; unit: string }
	style?: React.CSSProperties
}

const Value2Cmp: FunctionComponent<Value2CmpProps> = ({ title, value, rate, style }) => {
	return (
		<div style={{ display: "flex", flexDirection: "column", flex: 1, ...style }}>
			<div style={{ color: "#888", fontSize: 10 }}>{title}</div>
			<div>
				<span>{value.value?.toFixed(1) ?? "--"}</span><span>{value.unit}</span>
				<span> / </span>
				<span>{rate?.value?.toFixed(1) ?? "--"}</span><span>{rate?.unit}</span>
			</div>
		</div>
	)
}
