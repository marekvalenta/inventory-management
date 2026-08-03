import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronRightIcon, ArchiveIcon, CubeIcon } from '@radix-ui/react-icons'
import { Link } from 'react-router-dom'
import { fetchStacks } from '../../api/stacks'
import type { BrowseStack } from '../../api/stacks'
import type { BrowseNode } from '../../api/locations'
import styles from './BrowseTree.module.css'

interface BrowseTreeProps {
  rootNodes: BrowseNode[]
  onAddLocation: (parentId: string) => void
  onAddInstance: (locationId: string | null, parentInstanceId: string | null) => void
}

function StackNode({
  stack,
  depth,
  locationId,
  onAddInstance,
}: {
  stack: BrowseStack
  depth: number
  locationId?: string | null
  onAddInstance: (parentInstanceId: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const hasLoaded = useHasLoaded()

  const { data: subStacksData } = useQuery({
    queryKey: ['stacks', { parentInstanceId: stack.definition_id }],
    queryFn: () => fetchStacks({ parent_instance_id: '' }),
    enabled: false,
  })

  const isContainer = stack.is_container && stack.child_count > 0
  const isSingleInstance = stack.instance_count === 1 && stack.single_instance_id

  let linkUrl: string
  if (isSingleInstance) {
    linkUrl = `/instances/${stack.single_instance_id}`
  } else {
    const params = new URLSearchParams()
    params.set('definition_id', stack.definition_id)
    if (locationId) {
      params.set('location_id', locationId)
    }
    linkUrl = `/stacks?${params.toString()}`
  }

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
        <span className={styles.instanceIcon}>
          <CubeIcon width={18} height={18} />
        </span>
        <Link to={linkUrl} className={styles.name}>
          {stack.definition_name}
        </Link>
        <span className={styles.quantityBadge}>&times;{stack.total_quantity}</span>
        {!isSingleInstance && (
          <span className={styles.countLabel}>{stack.instance_count} instances</span>
        )}
        <button
          className={styles.addButton}
          onClick={(e) => {
            e.preventDefault()
            onAddInstance('')
          }}
          aria-label={`Add item inside ${stack.definition_name}`}
          title="Add item"
        >
          <CubeIcon width={15} height={15} />
          Add
        </button>
      </div>
      {expanded && isContainer && (
        subStacksData?.stacks?.map((sub) => (
          <StackNode
            key={`${sub.definition_id}-container`}
            stack={sub}
            depth={depth + 1}
            onAddInstance={onAddInstance}
          />
        ))
      )}
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

  const hasSubLocations = (node.children ?? []).length > 0
  const hasStacks = (node.stacks ?? []).length > 0
  const hasContent = hasSubLocations || hasStacks

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
          {hasSubLocations && hasStacks && (
            <div className={styles.divider} style={{ paddingLeft: `${(depth + 1) * 1.5}rem` }} />
          )}
          {node.stacks.map((stack) => (
            <StackNode
              key={stack.definition_id}
              stack={stack}
              depth={depth + 1}
              locationId={node.id}
              onAddInstance={(parentId) => onAddInstance(null, parentId)}
            />
          ))}
          {node.stack_truncated && (
            <div
              className={styles.truncatedHint}
              style={{ paddingLeft: `${(depth + 1) * 1.5}rem` }}
            >
              <Link to={`/locations/${node.id}`}>
                +{node.stack_count - node.stacks.length} more stacks
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
