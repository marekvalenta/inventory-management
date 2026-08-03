import { apiFetch } from './client'
import type { BrowseStack } from './stacks'

export interface Location {
  id: string
  name: string
  description: string | null
  parent_id: string | null
  created_at: string
  updated_at: string
}

export interface TreeNode {
  id: string
  name: string
  description: string | null
  children: TreeNode[]
}

export interface BreadcrumbNode {
  id: string
  name: string
}

export interface InstanceSummary {
  id: string
  definition_id: string
  definition_name: string
  quantity: number
}

export interface Contents {
  sub_locations: Location[]
  instances: InstanceSummary[]
}

export interface DeleteBlock {
  child_count: number
  instance_count: number
}

export interface BrowseNode {
  id: string
  name: string
  description: string | null
  kind: 'location'
  children: BrowseNode[]
  stacks: BrowseStack[]
  stack_count: number
  stack_truncated: boolean
}

export interface CreateLocationRequest {
  name: string
  description?: string | null
  parent_id?: string | null
}

export interface UpdateLocationRequest {
  name?: string
  description?: string | null
  parent_id?: string | null
}

export function fetchLocations(parentId?: string): Promise<Location[]> {
  const params = parentId ? `?parent_id=${encodeURIComponent(parentId)}` : ''
  return apiFetch<Location[]>(`/api/v1/locations${params}`)
}

export function fetchLocationTree(): Promise<TreeNode[]> {
  return apiFetch<TreeNode[]>('/api/v1/locations/tree')
}

export function fetchBrowse(): Promise<BrowseNode[]> {
  return apiFetch<BrowseNode[]>('/api/v1/browse')
}

export function fetchLocation(id: string): Promise<Location> {
  return apiFetch<Location>(`/api/v1/locations/${encodeURIComponent(id)}`)
}

export function fetchLocationChildren(id: string): Promise<Location[]> {
  return apiFetch<Location[]>(`/api/v1/locations/${encodeURIComponent(id)}/children`)
}

export function fetchLocationContents(id: string): Promise<Contents> {
  return apiFetch<Contents>(`/api/v1/locations/${encodeURIComponent(id)}/contents`)
}

export function fetchLocationBreadcrumb(id: string): Promise<BreadcrumbNode[]> {
  return apiFetch<BreadcrumbNode[]>(`/api/v1/locations/${encodeURIComponent(id)}/breadcrumb`)
}

export function createLocation(data: CreateLocationRequest): Promise<Location> {
  return apiFetch<Location>('/api/v1/locations', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateLocation(id: string, data: UpdateLocationRequest): Promise<Location> {
  return apiFetch<Location>(`/api/v1/locations/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteLocation(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/locations/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}
