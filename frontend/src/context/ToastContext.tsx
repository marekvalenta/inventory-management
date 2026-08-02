import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

export interface Toast {
  id: number
  message: string
  variant: 'success' | 'error' | 'warning'
}

interface ToastContextValue {
  toasts: Toast[]
  addToast: (message: string, variant: Toast['variant']) => void
  removeToast: (id: number) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const addToast = useCallback(
    (message: string, variant: Toast['variant']) => {
      const id = nextId++
      setToasts((prev) => [...prev, { id, message, variant }])
      setTimeout(() => removeToast(id), 4000)
    },
    [removeToast],
  )

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast }}>
      {children}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return ctx
}
