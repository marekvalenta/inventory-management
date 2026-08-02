import { Link } from 'react-router-dom'
import styles from './NotFoundPage.module.css'

export function NotFoundPage() {
  return (
    <div className={styles.container}>
      <h1 className={styles.code}>404</h1>
      <p className={styles.heading}>Page not found</p>
      <p className={styles.description}>
        The page you are looking for does not exist.
      </p>
      <Link to="/" className={styles.link}>
        Go Home
      </Link>
    </div>
  )
}
