import { ChevronRightIcon, HomeIcon } from '@radix-ui/react-icons'
import { Link } from 'react-router-dom'
import styles from './Breadcrumb.module.css'

interface BreadcrumbNode {
  id: string
  name: string
  kind?: 'location' | 'instance'
}

interface BreadcrumbProps {
  nodes: BreadcrumbNode[]
}

export function Breadcrumb({ nodes }: BreadcrumbProps) {
  return (
    <nav className={styles.breadcrumb} aria-label="Breadcrumb">
      {nodes.map((node, index) => {
        const isLast = index === nodes.length - 1
        const href = node.kind === 'instance'
          ? `/instances/${node.id}`
          : `/locations/${node.id}`
        return (
          <span key={node.id} className={styles.item}>
            {index > 0 && <ChevronRightIcon className={styles.separator} width={16} height={16} />}
            {index === 0 && <HomeIcon className={styles.homeIcon} width={16} height={16} />}
            {isLast ? (
              <span className={styles.current}>{node.name}</span>
            ) : (
              <Link to={href} className={styles.link}>
                {node.name}
              </Link>
            )}
          </span>
        )
      })}
    </nav>
  )
}
