import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronRightIcon, PlusIcon } from '@radix-ui/react-icons'
import { Link } from 'react-router-dom'
import { fetchLocationChildren } from '../../api/locations'
import type { Location } from '../../api/locations'
import styles from './LocationTree.module.css'

interface LocationTreeProps {
  rootLocations: Location[]
  onAddChild: (parentId: string) => void
}

function TreeNodeItem({
  location,
  depth,
  onAddChild,
}: {
  location: Location
  depth: number
  onAddChild: (parentId: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const hasLoaded = useHasLoaded()

  const { data: children, isFetching } = useQuery({
    queryKey: ['locations', location.id, 'children'],
    queryFn: () => fetchLocationChildren(location.id),
    enabled: !!expanded && !!hasLoaded,
  })

  const hasChildren = children && children.length > 0

  return (
    <div className={styles.node}>
      <div
        className={styles.nodeRow}
        style={{ paddingLeft: `${depth * var(--space-xl)}` }}
      >
        <button
          className={styles.expandButton}
          onClick={() => {
            setExpanded((prev) => !prev)
            hasLoaded.current = true
          }}
          aria-label={expanded ? 'Collapse' : 'Expand'}
        >
          <ChevronRightIcon
            width={20}
            height={20}
            style={{
              transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
              transition: `transform var(--transition-fast)`,
            }}
          />
        </button>
        <Link to={`/locations/${location.id}`} className={styles.name}>
          {location.name}
        </Link>
        <button
          className={styles.addButton}
          onClick={(e) => {
            e.preventDefault()
            onAddChild(location.id)
          }}
          aria-label={`Add sub-location to ${location.name}`}
        >
          <PlusIcon width={18} height={18} />
        </button>
      </div>
      {expanded && isFetching && <div className={styles.loading}>Loading...</div>}
      {expanded &&
        children?.map((child) => (
          <TreeNodeItem
            key={child.id}
            location={child}
            depth={depth + 1}
            onAddChild={onAddChild}
          />
        ))}
    </div>
  )
}

function useHasLoaded() {
  const [ref] = useState(() => ({ current: false }))
  return ref
}

export function LocationTree({
  rootLocations,
  onAddChild,
}: LocationTreeProps) {
  if (rootLocations.length === 0) {
    return (
      <div className={styles.empty}>
        No locations yet. The root "Home" location is the starting point.
      </div>
    )
  }

  return (
    <div className={styles.tree}>
      {rootLocations.map((loc) => (
        <TreeNodeItem
          key={loc.id}
          location={loc}
          depth={0}
          onAddChild={onAddChild}
        />
      ))}
    </div>
  )
}
