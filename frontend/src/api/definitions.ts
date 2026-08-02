import { apiFetch } from './client'

export interface DefinitionSummary {
  id: string
  name: string
  description: string | null
  parent_def_id: string | null
  parent_def_name: string | null
  unit: string | null
  is_container: boolean
  total_instances: number
  tags: DefinitionTag[]
  created_at: string
  updated_at: string
}

export interface DefinitionTag {
  id: string
  name: string
  color: string | null
  created_at: string
  updated_at: string
}

export interface DefinitionField {
  id: string
  field_name: string
  field_type: 'text' | 'number' | 'boolean' | 'date' | 'enum'
  enum_values: string[] | null
  is_required: boolean
  display_order: number
  default_value: string | null
  is_child_editable: boolean
  inherited_from_def_id: string | null
}

export interface LocationInstanceCount {
  location_id: string
  location_name: string
  instance_count: number
  total_quantity: number
}

export interface ParentInstanceCount {
  parent_instance_id: string
  parent_instance_name: string
  location_id: string
  location_name: string
  instance_count: number
  total_quantity: number
}

export interface InstancesSummary {
  total_instances: number
  total_quantity: number
  by_location: LocationInstanceCount[]
  by_parent_instance: ParentInstanceCount[]
}

export interface DefinitionDetail {
  id: string
  name: string
  description: string | null
  parent_def_id: string | null
  parent_def_name: string | null
  unit: string | null
  is_container: boolean
  created_at: string
  updated_at: string
  fields: DefinitionField[]
  tags: DefinitionTag[]
  instances_summary: InstancesSummary
  child_definition_count: number
}

export interface CreateFieldInput {
  field_name: string
  field_type: 'text' | 'number' | 'boolean' | 'date' | 'enum'
  enum_values: string[] | null
  is_required: boolean
  display_order: number
  default_value: string | null
  is_child_editable: boolean
}

export interface CreateDefinitionRequest {
  name: string
  description?: string | null
  parent_def_id?: string | null
  unit?: string | null
  is_container?: boolean
  fields?: CreateFieldInput[]
  tag_ids?: string[]
}

export interface UpdateDefinitionRequest {
  name?: string
  description?: string | null
  parent_def_id?: string | null
  unit?: string | null
  is_container?: boolean
  fields?: CreateFieldInput[]
  tag_ids?: string[]
}

export interface OverrideInput {
  parent_field_id: string
  default_value: string | null
}

export interface OverrideResponse {
  definition_id: string
  parent_field_id: string
  default_value: string | null
}

export function fetchDefinitions(): Promise<DefinitionSummary[]> {
  return apiFetch<DefinitionSummary[]>('/api/v1/definitions')
}

export function fetchDefinition(id: string): Promise<DefinitionDetail> {
  return apiFetch<DefinitionDetail>(`/api/v1/definitions/${encodeURIComponent(id)}`)
}

export function createDefinition(data: CreateDefinitionRequest): Promise<DefinitionDetail> {
  return apiFetch<DefinitionDetail>('/api/v1/definitions', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateDefinition(id: string, data: UpdateDefinitionRequest): Promise<DefinitionDetail> {
  return apiFetch<DefinitionDetail>(`/api/v1/definitions/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deleteDefinition(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/definitions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function updateOverrides(id: string, overrides: OverrideInput[]): Promise<{ overrides: OverrideResponse[] }> {
  return apiFetch<{ overrides: OverrideResponse[] }>(
    `/api/v1/definitions/${encodeURIComponent(id)}/overrides`,
    {
      method: 'PUT',
      body: JSON.stringify({ overrides }),
    },
  )
}
