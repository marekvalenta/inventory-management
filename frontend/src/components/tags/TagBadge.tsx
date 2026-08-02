import styles from './TagBadge.module.css'

interface TagBadgeTag {
  id: string
  name: string
  color: string | null
}

interface TagBadgeProps {
  tag: TagBadgeTag
  size?: 'sm' | 'md'
}

export function TagBadge({ tag, size = 'sm' }: TagBadgeProps) {
  const swatchColor = tag.color || '#605C57'

  return (
    <span className={`${styles.badge} ${size === 'md' ? styles.badgeMd : ''}`}>
      <span
        className={styles.swatch}
        style={{ backgroundColor: swatchColor }}
      />
      <span className={styles.name}>{tag.name}</span>
    </span>
  )
}
