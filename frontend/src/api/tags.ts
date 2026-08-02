import { apiFetch } from './client'

export interface Tag {
  id: string
  name: string
  color: string | null
  linked_definitions_count: number
  created_at: string
  updated_at: string
}

export interface CreateTagRequest {
  name: string
  color?: string | null
}

export interface UpdateTagRequest {
  name?: string
  color?: string | null
}

export interface DeleteTagResponse {
  deleted: boolean
  linked_definitions_count: number
}

export function fetchTags(): Promise<Tag[]> {
  return apiFetch<Tag[]>('/api/v1/tags')
}

export function fetchTag(id: string): Promise<Tag> {
  return apiFetch<Tag>(`/api/v1/tags/${encodeURIComponent(id)}`)
}

export function createTag(data: CreateTagRequest): Promise<Tag> {
  return apiFetch<Tag>('/api/v1/tags', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateTag(id: string, data: UpdateTagRequest): Promise<Tag> {
  return apiFetch<Tag>(`/api/v1/tags/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteTag(id: string): Promise<DeleteTagResponse> {
  return apiFetch<DeleteTagResponse>(`/api/v1/tags/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}
