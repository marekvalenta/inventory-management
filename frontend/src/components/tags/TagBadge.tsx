import type { Tag } from '../../api/tags'
import styles from './TagBadge.module.css'

interface TagBadgeProps {
  tag: Tag
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
