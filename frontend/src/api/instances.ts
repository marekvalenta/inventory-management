import { apiFetch } from './client'

export interface InstanceFieldValue {
  field_id: string
  field_name: string
  field_type: string
  enum_values: string[] | null
  value: string | null
}

export interface BreadcrumbEntry {
  id: string
  name: string
  kind: 'location' | 'instance'
}

export interface InstanceDetail {
  id: string
  definition_id: string
  definition_name: string
  parent_def_id: string | null
  parent_def_name: string | null
  unit: string | null
  quantity: number
  location_id: string | null
  location_name: string | null
  parent_instance_id: string | null
  parent_instance_name: string | null
  field_values: InstanceFieldValue[]
  child_instance_count: number
  breadcrumb: BreadcrumbEntry[]
  created_at: string
  updated_at: string
}

export interface InstanceSummary {
  id: string
  definition_id: string
  definition_name: string
  quantity: number
  location_id: string | null
  location_name: string | null
  parent_instance_id: string | null
  parent_instance_name: string | null
  updated_at: string
}

export interface InstanceListResult {
  instances: InstanceSummary[]
  total_count: number
  truncated?: boolean
}

export interface MoveResult {
  source: InstanceDetail | null
  target: InstanceDetail
}

export interface CreateInstanceRequest {
  definition_id: string
  quantity: number
  location_id?: string | null
  parent_instance_id?: string | null
  field_values?: { field_id: string; value: string | null }[]
}

export interface UpdateInstanceRequest {
  quantity?: number
  field_values?: { field_id: string; value: string | null }[]
}

export interface MoveInstanceRequest {
  quantity: number
  target_location_id?: string | null
  target_parent_instance_id?: string | null
}

export interface InstanceContentsResponse {
  instances: InstanceSummary[]
}

export function fetchInstances(params?: {
  location_id?: string
  definition_id?: string
  parent_instance_id?: string
}): Promise<InstanceListResult> {
  const searchParams = new URLSearchParams()
  if (params?.location_id) searchParams.set('location_id', params.location_id)
  if (params?.definition_id) searchParams.set('definition_id', params.definition_id)
  if (params?.parent_instance_id) searchParams.set('parent_instance_id', params.parent_instance_id)
  const qs = searchParams.toString()
  return apiFetch<InstanceListResult>(`/api/v1/instances${qs ? `?${qs}` : ''}`)
}

export function fetchInstance(id: string): Promise<InstanceDetail> {
  return apiFetch<InstanceDetail>(`/api/v1/instances/${encodeURIComponent(id)}`)
}

export function createInstance(data: CreateInstanceRequest): Promise<InstanceDetail> {
  return apiFetch<InstanceDetail>('/api/v1/instances', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateInstance(id: string, data: UpdateInstanceRequest): Promise<InstanceDetail> {
  return apiFetch<InstanceDetail>(`/api/v1/instances/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteInstance(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/instances/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function moveInstance(id: string, data: MoveInstanceRequest): Promise<MoveResult> {
  return apiFetch<MoveResult>(`/api/v1/instances/${encodeURIComponent(id)}/move`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function fetchInstanceContents(id: string): Promise<InstanceContentsResponse> {
  return apiFetch<InstanceContentsResponse>(`/api/v1/instances/${encodeURIComponent(id)}/contents`)
}

export function fetchInstanceBreadcrumb(id: string): Promise<BreadcrumbEntry[]> {
  return apiFetch<BreadcrumbEntry[]>(`/api/v1/instances/${encodeURIComponent(id)}/breadcrumb`)
}
