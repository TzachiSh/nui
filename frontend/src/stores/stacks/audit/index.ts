import auditApi, { AuditLog, AuditListFilter } from "@/api/audit"
import viewSetup, { ViewStore } from "@/stores/stacks/viewBase"
import { StoreCore, mixStores } from "@priolo/jon"
import { ViewState } from "../viewBase"


const setup = {

	state: {
		//#region VIEWBASE
		pinnable: false,
		width: 500,
		//#endregion
		logs: [] as AuditLog[],
		filter: {
			sort_by: "timestamp",
			sort_desc: true,
		} as AuditListFilter,
		loading: false,
		autoRefresh: false,
		refreshInterval: null as ReturnType<typeof setInterval> | null,
	},

	getters: {
		//#region VIEWBASE
		getTitle: (_: void, store?: ViewStore) => "AUDIT",
		getSubTitle: (_: void, store?: ViewStore) => "Activity logs",
		getSerialization: (_: void, store?: ViewStore) => {
			const state = store.state as AuditState
			return {
				...viewSetup.getters.getSerialization(null, store),
				filter: state.filter,
			}
		},
		//#endregion
	},

	actions: {
		//#region VIEWBASE
		setSerialization: (data: any, store?: AuditStore) => {
			viewSetup.actions.setSerialization(data, store)
			if (data.filter) store.setFilter(data.filter)
		},
		//#endregion

		async fetch(_: void, store?: AuditStore) {
			store.setLoading(true)
			try {
				const logs = await auditApi.list(store.state.filter)
				store.setLogs(logs || [])
			} catch (e) {
				console.error("Failed to fetch audit logs", e)
				store.setLogs([])
			} finally {
				store.setLoading(false)
			}
		},

		toggleAutoRefresh(_: void, store?: AuditStore) {
			if (store.state.autoRefresh) {
				// Stop auto-refresh
				if (store.state.refreshInterval) {
					clearInterval(store.state.refreshInterval)
					store.setRefreshInterval(null)
				}
				store.setAutoRefresh(false)
			} else {
				// Start auto-refresh
				store.fetch()
				const interval = setInterval(() => store.fetch(), 10000)
				store.setRefreshInterval(interval)
				store.setAutoRefresh(true)
			}
		},

		cleanup(_: void, store?: AuditStore) {
			if (store.state.refreshInterval) {
				clearInterval(store.state.refreshInterval)
				store.setRefreshInterval(null)
			}
		},

		setSort(sortBy: string, store?: AuditStore) {
			const currentSortBy = store.state.filter.sort_by
			const currentDesc = store.state.filter.sort_desc

			// If clicking same column, toggle direction
			// If clicking different column, set to descending
			const newDesc = sortBy === currentSortBy ? !currentDesc : true

			store.setFilter({
				...store.state.filter,
				sort_by: sortBy,
				sort_desc: newDesc,
			})
			store.fetch()
		},
	},

	mutators: {
		setLogs: (logs: AuditLog[]) => ({ logs }),
		setFilter: (filter: AuditListFilter) => ({ filter }),
		setLoading: (loading: boolean) => ({ loading }),
		setAutoRefresh: (autoRefresh: boolean) => ({ autoRefresh }),
		setRefreshInterval: (refreshInterval: ReturnType<typeof setInterval> | null) => ({ refreshInterval }),
	},
}

export type AuditState = typeof setup.state & ViewState
export type AuditGetters = typeof setup.getters
export type AuditActions = typeof setup.actions
export type AuditMutators = typeof setup.mutators
export interface AuditStore extends ViewStore, AuditGetters, AuditActions, AuditMutators {
	state: AuditState
}
const auditSetup = mixStores(viewSetup, setup)
export default auditSetup
