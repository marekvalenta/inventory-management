import styles from './HomePage.module.css'

export function HomePage() {
  return (
    <div className={styles.empty}>
      <h2 className={styles.title}>Inventory</h2>
      <p className={styles.subtitle}>Select a section from the navigation to get started.</p>
    </div>
  )
}
