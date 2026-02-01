import FrameworkCard from "@/components/cards/FrameworkCard"
import AuditIcon from "@/icons/AuditIcon"
import { AuditLog } from "@/api/audit"
import { AuditStore } from "@/stores/stacks/audit"
import { Button } from "@priolo/jack"
import { useStore } from "@priolo/jon"
import React, { FunctionComponent, useEffect } from "react"
import clsCard from "../CardWhiteDef.module.css"
import cls from "./View.module.css"



interface Props {
	store?: AuditStore
	style?: React.CSSProperties
}

const AuditView: FunctionComponent<Props> = ({
	store: auditSo,
	style,
}) => {

	// STORE
	const auditSa = useStore(auditSo)

	// HOOKs
	useEffect(() => {
		auditSo.fetch()
		return () => auditSo.cleanup()
	}, [])

	// HANDLER
	const handleRefresh = () => auditSo.fetch()
	const handleToggleAutoRefresh = () => auditSo.toggleAutoRefresh()
	const handleSort = (field: string) => auditSo.setSort(field)

	// RENDER
	const logs = auditSa.logs || []
	const loading = auditSa.loading
	const sortBy = auditSa.filter?.sort_by || "timestamp"
	const sortDesc = auditSa.filter?.sort_desc !== false

	const getSortIndicator = (field: string) => {
		if (sortBy !== field) return ""
		return sortDesc ? " ▼" : " ▲"
	}

	return <FrameworkCard
		icon={<AuditIcon />}
		className={clsCard.root}
		store={auditSo}
		actionsRender={<>
			<Button
				children={auditSa.autoRefresh ? "STOP" : "AUTO"}
				onClick={handleToggleAutoRefresh}
				select={auditSa.autoRefresh}
			/>
			<Button
				children="REFRESH"
				onClick={handleRefresh}
			/>
		</>}
	>
		<div className={cls.container}>
			{loading && <div className={cls.loading}>Loading...</div>}

			{!loading && logs.length === 0 && (
				<div className={cls.empty}>No audit logs found</div>
			)}

			{!loading && logs.length > 0 && (
				<table className={cls.table}>
					<thead>
						<tr>
							<th className={cls.sortable} onClick={() => handleSort("timestamp")}>
								Time{getSortIndicator("timestamp")}
							</th>
							<th className={cls.sortable} onClick={() => handleSort("user_name")}>
								User{getSortIndicator("user_name")}
							</th>
							<th className={cls.sortable} onClick={() => handleSort("action")}>
								Action{getSortIndicator("action")}
							</th>
							<th>Path</th>
							<th>Status</th>
						</tr>
					</thead>
					<tbody>
						{logs.map((log) => (
							<AuditRow key={log.id} log={log} />
						))}
					</tbody>
				</table>
			)}
		</div>
	</FrameworkCard>
}

interface AuditRowProps {
	log: AuditLog
}

const AuditRow: FunctionComponent<AuditRowProps> = ({ log }) => {
	const [expanded, setExpanded] = React.useState(false)

	const formatTime = (timestamp: string) => {
		const date = new Date(timestamp)
		return date.toLocaleString()
	}

	const getStatusColor = (code: number) => {
		if (code >= 200 && code < 300) return "var(--color-green)"
		if (code >= 400 && code < 500) return "var(--color-yellow)"
		if (code >= 500) return "var(--color-red)"
		return "var(--color-fg)"
	}

	return (
		<>
			<tr onClick={() => setExpanded(!expanded)} style={{ cursor: "pointer" }}>
				<td>{formatTime(log.timestamp)}</td>
				<td>{log.userName || log.userEmail || log.userId || "-"}</td>
				<td>{log.action}</td>
				<td className={cls.pathCell}>{log.path}</td>
				<td style={{ color: getStatusColor(log.statusCode) }}>{log.statusCode}</td>
			</tr>
			{expanded && (
				<tr className={cls.expandedRow}>
					<td colSpan={5}>
						<div className={cls.details}>
							<div><strong>ID:</strong> {log.id}</div>
							{log.userName && <div><strong>User:</strong> {log.userName}</div>}
							{log.userEmail && <div><strong>Email:</strong> {log.userEmail}</div>}
							<div><strong>Method:</strong> {log.method}</div>
							{log.resourceType && <div><strong>Resource:</strong> {log.resourceType}</div>}
							{log.resourceId && <div><strong>Resource ID:</strong> {log.resourceId}</div>}
							{log.topic && <div><strong>Topic:</strong> {log.topic}</div>}
							{log.details && (
								<div><strong>Details:</strong> <pre>{JSON.stringify(log.details, null, 2)}</pre></div>
							)}
						</div>
					</td>
				</tr>
			)}
		</>
	)
}

export default AuditView
