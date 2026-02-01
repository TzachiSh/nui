import ajax, { CallOptions } from "@/plugins/AjaxService"

export interface AuditLog {
	id: string
	timestamp: string
	userId: string
	userEmail: string
	userName: string
	action: string
	method: string
	path: string
	statusCode: number
	resourceType?: string
	resourceId?: string
	topic?: string
	details?: Record<string, unknown>
}

export interface AuditListFilter {
	user_id?: string
	action?: string
	start_time?: string
	end_time?: string
	limit?: number
	offset?: number
	sort_by?: string
	sort_desc?: boolean
}

function list(filter?: AuditListFilter, opt?: CallOptions): Promise<AuditLog[]> {
	const params = new URLSearchParams()
	if (filter?.user_id) params.append("user_id", filter.user_id)
	if (filter?.action) params.append("action", filter.action)
	if (filter?.start_time) params.append("start_time", filter.start_time)
	if (filter?.end_time) params.append("end_time", filter.end_time)
	if (filter?.limit) params.append("limit", filter.limit.toString())
	if (filter?.offset) params.append("offset", filter.offset.toString())
	if (filter?.sort_by) params.append("sort_by", filter.sort_by)
	if (filter?.sort_desc !== undefined) params.append("sort_desc", filter.sort_desc.toString())

	const queryString = params.toString()
	const url = queryString ? `audit?${queryString}` : "audit"
	return ajax.get(url, null, opt)
}

const auditApi = {
	list,
}

export default auditApi
