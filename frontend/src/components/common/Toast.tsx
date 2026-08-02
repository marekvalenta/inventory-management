import { useToast } from '../../context/ToastContext'
import styles from './Toast.module.css'

const icons: Record<string, string> = {
  success: '\u2713',
  error: '\u2717',
  warning: '\u2139',
}

export function ToastContainer() {
  const { toasts, removeToast } = useToast()

  if (toasts.length === 0) return null

  return (
    <div className={styles.container}>
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`${styles.toast} ${styles[`toast${toast.variant.charAt(0).toUpperCase() + toast.variant.slice(1)}`]}`}
          onClick={() => removeToast(toast.id)}
        >
          <span className={styles.icon}>{icons[toast.variant]}</span>
          <span>{toast.message}</span>
        </div>
      ))}
    </div>
  )
}
