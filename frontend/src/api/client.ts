export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    let body: { error?: string; code?: string }
    try {
      body = await res.json()
    } catch {
      throw new ApiError(res.status, 'unknown', 'An unexpected error occurred')
    }
    throw new ApiError(res.status, body.code ?? 'unknown', body.error ?? 'An unexpected error occurred')
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json()
}
