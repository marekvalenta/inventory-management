import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronRightIcon, ArchiveIcon, CubeIcon } from '@radix-ui/react-icons'
import { Link } from 'react-router-dom'
import { fetchInstanceContents } from '../../api/instances'
import type { BrowseNode, BrowseInstance } from '../../api/locations'
import styles from './BrowseTree.module.css'

interface BrowseTreeProps {
  rootNodes: BrowseNode[]
  onAddLocation: (parentId: string) => void
  onAddInstance: (locationId: string | null, parentInstanceId: string | null) => void
}

function InstanceNode({
  inst,
  depth,
  onAddInstance,
}: {
  inst: BrowseInstance
  depth: number
  onAddInstance: (parentInstanceId: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const hasLoaded = useHasLoaded()

  const { data: contents } = useQuery({
    queryKey: ['instances', inst.id, 'contents'],
    queryFn: () => fetchInstanceContents(inst.id),
    enabled: expanded && hasLoaded.current,
  })

  const childInstances = contents?.instances ?? []
  const isContainer = inst.is_container && inst.child_count > 0
  const instanceIcon = isContainer ? <CubeIcon width={18} height={18} /> : <CubeIcon width={18} height={18} />

  return (
    <div className={styles.node}>
      <div
        className={styles.nodeRow}
        style={{ paddingLeft: `${depth * 1.5}rem` }}
      >
        <button
          className={styles.expandButton}
          onClick={() => {
            if (isContainer) {
              setExpanded((prev) => !prev)
              hasLoaded.current = true
            }
          }}
          aria-label={expanded ? 'Collapse' : 'Expand'}
          disabled={!isContainer}
          style={!isContainer ? { visibility: 'hidden' } : undefined}
        >
          <ChevronRightIcon
            width={20}
            height={20}
            style={{
              transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
              transition: 'transform 0.2s ease',
            }}
          />
        </button>
        <span className={styles.instanceIcon}>{instanceIcon}</span>
        <Link to={`/instances/${inst.id}`} className={styles.name}>
          {inst.definition_name}
        </Link>
        <span className={styles.quantityBadge}>&times;{inst.quantity}</span>
        <button
          className={styles.addButton}
          onClick={(e) => {
            e.preventDefault()
            onAddInstance(inst.id)
          }}
          aria-label={`Add item inside ${inst.definition_name}`}
          title="Add item"
        >
          <CubeIcon width={15} height={15} />
          Add
        </button>
      </div>
      {expanded &&
        childInstances.map((child) => (
          <InstanceNode
            key={child.id}
            inst={{
              id: child.id,
              definition_id: child.definition_id,
              definition_name: child.definition_name,
              quantity: child.quantity,
              is_container: false,
              child_count: 0,
            }}
            depth={depth + 1}
            onAddInstance={onAddInstance}
          />
        ))}
    </div>
  )
}

function LocationNode({
  node,
  depth,
  onAddLocation,
  onAddInstance,
}: {
  node: BrowseNode
  depth: number
  onAddLocation: (parentId: string) => void
  onAddInstance: (locationId: string | null, parentInstanceId: string | null) => void
}) {
  const [expanded, setExpanded] = useState(depth === 0)

  const hasSubLocations = node.children.length > 0
  const hasInstances = node.instances.length > 0
  const hasContent = hasSubLocations || hasInstances

  return (
    <div className={styles.node}>
      <div
        className={styles.nodeRow}
        style={{ paddingLeft: `${depth * 1.5}rem` }}
      >
        <button
          className={styles.expandButton}
          onClick={() => setExpanded((prev) => !prev)}
          aria-label={expanded ? 'Collapse' : 'Expand'}
          disabled={!hasContent}
          style={!hasContent ? { visibility: 'hidden' } : undefined}
        >
          <ChevronRightIcon
            width={20}
            height={20}
            style={{
              transform: expanded ? 'rotate(90deg)' : 'rotate(0deg)',
              transition: 'transform 0.2s ease',
            }}
          />
        </button>
        <Link to={`/locations/${node.id}`} className={styles.name}>
          {node.name}
        </Link>
        <div className={styles.actionGroup}>
          <button
            className={styles.addButton}
            onClick={(e) => {
              e.preventDefault()
              onAddLocation(node.id)
            }}
            aria-label={`Add sub-location to ${node.name}`}
            title="Add sub-location"
          >
            <ArchiveIcon width={15} height={15} />
            Add
          </button>
          <button
            className={styles.addButton}
            onClick={(e) => {
              e.preventDefault()
              onAddInstance(node.id, null)
            }}
            aria-label={`Add item to ${node.name}`}
            title="Add item"
          >
            <CubeIcon width={15} height={15} />
            Add
          </button>
        </div>
      </div>
      {expanded && (
        <>
          {node.children.map((child) => (
            <LocationNode
              key={child.id}
              node={child}
              depth={depth + 1}
              onAddLocation={onAddLocation}
              onAddInstance={onAddInstance}
            />
          ))}
          {hasSubLocations && hasInstances && (
            <div className={styles.divider} style={{ paddingLeft: `${(depth + 1) * 1.5}rem` }} />
          )}
          {node.instances.map((inst) => (
            <InstanceNode
              key={inst.id}
              inst={inst}
              depth={depth + 1}
              onAddInstance={(parentId) => onAddInstance(null, parentId)}
            />
          ))}
          {node.instance_truncated && (
            <div
              className={styles.truncatedHint}
              style={{ paddingLeft: `${(depth + 1) * 1.5}rem` }}
            >
              <Link to={`/locations/${node.id}`}>
                +{node.instance_count - node.instances.length} more items
              </Link>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function useHasLoaded() {
  const [ref] = useState(() => ({ current: false }))
  return ref
}

export function BrowseTree({
  rootNodes,
  onAddLocation,
  onAddInstance,
}: BrowseTreeProps) {
  if (rootNodes.length === 0) {
    return (
      <div className={styles.empty}>
        No locations or items yet. The root "Home" location is the starting point.
      </div>
    )
  }

  return (
    <div className={styles.tree}>
      {rootNodes.map((node) => (
        <LocationNode
          key={node.id}
          node={node}
          depth={0}
          onAddLocation={onAddLocation}
          onAddInstance={onAddInstance}
        />
      ))}
    </div>
  )
}
