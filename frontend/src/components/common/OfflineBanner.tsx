import { useOnlineStatus } from '../../context/OnlineStatusContext'
import styles from './OfflineBanner.module.css'

export function OfflineBanner() {
  const { isOnline } = useOnlineStatus()

  if (isOnline) return null

  return (
    <div className={styles.banner}>
      You are offline. Browsing is read-only.
    </div>
  )
}
