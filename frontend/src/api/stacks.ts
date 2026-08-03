import { apiFetch } from './client'

export interface BrowseStack {
  definition_id: string
  definition_name: string
  unit: string | null
  total_quantity: number
  instance_count: number
  is_container: boolean
  child_count: number
  single_instance_id?: string | null
}

export interface StackListResult {
  stacks: BrowseStack[]
  total_count: number
  truncated?: boolean
}

export interface StackInstanceFieldValue {
  field_id: string
  field_name: string
  field_type: string
  enum_values: string[] | null
  value: string | null
}

export interface InstanceInStack {
  id: string
  definition_id: string
  definition_name: string
  quantity: number
  field_values: StackInstanceFieldValue[]
  location_id: string | null
  location_name: string | null
  parent_instance_id: string | null
  parent_instance_name: string | null
  created_at: string
  updated_at: string
}

export interface PaginationInfo {
  page: number
  per_page: number
  total_pages: number
  total_instances: number
}

export interface BreadcrumbEntry {
  id: string
  name: string
  kind: 'location' | 'instance'
}

export interface StackDetail {
  definition_id: string
  definition_name: string
  unit: string | null
  is_container: boolean
  parent_def_id: string | null
  parent_def_name: string | null
  location_id: string | null
  location_name: string | null
  parent_instance_id: string | null
  parent_instance_name: string | null
  total_quantity: number
  instance_count: number
  child_count: number
  breadcrumb: BreadcrumbEntry[]
  instances: InstanceInStack[]
  pagination: PaginationInfo
}

export interface MoveStackRequest {
  definition_id: string
  source_location_id?: string | null
  source_parent_instance_id?: string | null
  quantity: number
  target_location_id?: string | null
  target_parent_instance_id?: string | null
}

export interface MoveStackResult {
  moved_quantity: number
  source: StackDetail | null
  target: StackDetail
}

export function fetchStacks(params?: {
  location_id?: string
  parent_instance_id?: string
}): Promise<StackListResult> {
  const searchParams = new URLSearchParams()
  if (params?.location_id) searchParams.set('location_id', params.location_id)
  if (params?.parent_instance_id) searchParams.set('parent_instance_id', params.parent_instance_id)
  const qs = searchParams.toString()
  return apiFetch<StackListResult>(`/api/v1/stacks${qs ? `?${qs}` : ''}`)
}

export function fetchStackDetail(params: {
  definition_id: string
  location_id?: string
  parent_instance_id?: string
  page?: number
  per_page?: number
}): Promise<StackDetail> {
  const searchParams = new URLSearchParams()
  searchParams.set('definition_id', params.definition_id)
  if (params.location_id) searchParams.set('location_id', params.location_id)
  if (params.parent_instance_id) searchParams.set('parent_instance_id', params.parent_instance_id)
  if (params.page) searchParams.set('page', String(params.page))
  if (params.per_page) searchParams.set('per_page', String(params.per_page))
  return apiFetch<StackDetail>(`/api/v1/stacks/detail?${searchParams.toString()}`)
}

export function moveStack(data: MoveStackRequest): Promise<MoveStackResult> {
  return apiFetch<MoveStackResult>('/api/v1/stacks/move', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function deleteStack(params: {
  definition_id: string
  location_id?: string
  parent_instance_id?: string
}): Promise<void> {
  const searchParams = new URLSearchParams()
  searchParams.set('definition_id', params.definition_id)
  if (params.location_id) searchParams.set('location_id', params.location_id)
  if (params.parent_instance_id) searchParams.set('parent_instance_id', params.parent_instance_id)
  return apiFetch<void>(`/api/v1/stacks?${searchParams.toString()}`, {
    method: 'DELETE',
  })
}
