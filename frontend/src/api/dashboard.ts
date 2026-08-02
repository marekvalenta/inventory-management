import { apiFetch } from './client'

export interface DashboardStats {
  locations_count: number
  definitions_count: number
  instances_count: number
  total_quantity: number
}

export interface LocationNode {
  id: string
  name: string
  instance_count: number
  direct_instance_count: number
  sub_location_count: number
  children: LocationNode[]
}

export interface DashboardData {
  stats: DashboardStats
  locations: LocationNode[]
  is_onboarding: boolean
}

export async function fetchDashboard(): Promise<DashboardData> {
  return apiFetch<DashboardData>('/api/v1/dashboard')
}
