import { useState, useEffect } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CubeIcon,
  PlusIcon,
  TrashIcon,
  MoveIcon,
  Link2Icon,
} from '@radix-ui/react-icons'
import { fetchStackDetail } from '../api/stacks'
import type { StackDetail } from '../api/stacks'
import type { CreateInstanceRequest } from '../api/instances'
import { createInstance } from '../api/instances'
import { Breadcrumb } from '../components/breadcrumb/Breadcrumb'
import { CreateInstanceModal } from '../components/instances/CreateInstanceModal'
import { ApiError } from '../api/client'
import { useToast } from '../context/ToastContext'
import styles from './StackDetailPage.module.css'

export function StackDetailPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const definitionId = searchParams.get('definition_id') || ''
  const locationId = searchParams.get('location_id') || undefined
  const parentInstanceId = searchParams.get('parent_instance_id') || undefined
  const queryClient = useQueryClient()
  const { addToast } = useToast()
  const [page, setPage] = useState(1)
  const [showCreate, setShowCreate] = useState(false)
  const [showMove, setShowMove] = useState(false)
  const [showDelete, setShowDelete] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState('')

  const hasParent = !!(locationId || parentInstanceId)
  const validationError = !definitionId || (!locationId && !parentInstanceId) || (!!locationId && !!parentInstanceId)

  const {
    data: stack,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['stacks', 'detail', { definitionId, locationId, parentInstanceId, page }],
    queryFn: () =>
      fetchStackDetail({
        definition_id: definitionId,
        location_id: locationId,
        parent_instance_id: parentInstanceId,
        page,
      }),
    enabled: !!hasParent && !validationError,
  })

  const singleInstanceId =
    stack && stack.instance_count === 1 && stack.instances.length === 1
      ? stack.instances[0].id
      : null
  useEffect(() => {
    if (singleInstanceId) {
      navigate(`/instances/${singleInstanceId}`, { replace: true })
    }
  }, [singleInstanceId, navigate])

  if (!hasParent || validationError) {
    return (
      <div className={styles.container}>
        <div className={styles.empty}>
          Select an item type and location to view its stack.
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading stack...</div>
      </div>
    )
  }

  if (error || !stack) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <div className={styles.container}>
          <div className={styles.error}>
            No items of this type found at this location.
          </div>
        </div>
      )
    }
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          {error instanceof Error ? error.message : 'Failed to load stack'}
        </div>
      </div>
    )
  }

  if (singleInstanceId) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Redirecting to instance...</div>
      </div>
    )
  }

  const totalPages = stack.pagination.total_pages
  const stackKey = {
    definitionId,
    locationId: locationId || undefined,
    parentInstanceId: parentInstanceId || undefined,
  }

  return (
    <div className={styles.container}>
      {stack.breadcrumb && stack.breadcrumb.length > 0 && (
        <Breadcrumb nodes={stack.breadcrumb} />
      )}

      <div className={styles.header}>
        <div className={styles.headerInfo}>
          <h1 className={styles.heading}>{stack.definition_name}</h1>
          <Link
            to={`/definitions/${stack.definition_id}`}
            className={styles.definitionLink}
          >
            <Link2Icon width={16} height={16} />
            View definition
          </Link>
          <div className={styles.metaRow}>
            <span className={styles.quantityBadge}>
              <CubeIcon width={16} height={16} />
              x{stack.total_quantity}
            </span>
            {stack.unit && (
              <span className={styles.unitBadge}>{stack.unit}</span>
            )}
            <span className={styles.instanceCount}>
              {stack.instance_count} instance{stack.instance_count !== 1 ? 's' : ''}
            </span>
          </div>
        </div>
        <div className={styles.headerActions}>
          <button
            className={styles.actionButton}
            onClick={() => setShowCreate(true)}
            title="Add item to stack"
          >
            <PlusIcon width={20} height={20} />
            Add Item
          </button>
          {stack.total_quantity > 0 && (
            <>
              <button
                className={styles.actionButton}
                onClick={() => setShowMove(true)}
                title="Move items from stack"
              >
                <MoveIcon width={20} height={20} />
                Move Items
              </button>
              <button
                className={styles.deleteButton}
                onClick={() => setShowDelete(true)}
                title="Delete all items in stack"
              >
                <TrashIcon width={20} height={20} />
                Delete All
              </button>
            </>
          )}
        </div>
      </div>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Placement</h2>
        <div className={styles.placementInfo}>
          {stack.location_name ? (
            <>
              Located in:{' '}
              <Link
                to={`/locations/${stack.location_id}`}
                className={styles.placementLink}
              >
                {stack.location_name}
              </Link>
            </>
          ) : stack.parent_instance_name ? (
            <>
              Inside:{' '}
              <Link
                to={`/instances/${stack.parent_instance_id}`}
                className={styles.placementLink}
              >
                {stack.parent_instance_name}
              </Link>
              {stack.location_name && (
                <>
                  {' '}
                  (in{' '}
                  <Link
                    to={`/locations/${stack.location_id}`}
                    className={styles.placementLink}
                  >
                    {stack.location_name}
                  </Link>
                  )
                </>
              )}
            </>
          ) : null}
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>
          Instances ({stack.instance_count})
        </h2>
        {stack.instances.length === 0 ? (
          <p className={styles.contentsEmpty}>No instances in this stack</p>
        ) : (
          <>
            <div className={styles.contentsList}>
              {stack.instances.map((inst) => (
                <a
                  key={inst.id}
                  href={`/instances/${inst.id}`}
                  className={styles.contentItem}
                  onClick={(e) => {
                    e.preventDefault()
                    navigate(`/instances/${inst.id}`)
                  }}
                >
                  <div className={styles.contentFields}>
                    {inst.field_values && inst.field_values.length > 0 ? (
                      inst.field_values.map((fv) => (
                        <span key={fv.field_id} className={styles.fieldChip}>
                          {fv.field_name}: {fv.value ?? '\u2014'}
                        </span>
                      ))
                    ) : (
                      <span className={styles.fieldChipEmpty}>
                        No field values
                      </span>
                    )}
                  </div>
                  <div className={styles.contentRight}>
                    <span className={styles.contentQuantity}>
                      x{inst.quantity}
                    </span>
                    <span className={styles.contentDate}>
                      {new Date(inst.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </a>
              ))}
            </div>
            {totalPages > 1 && (
              <div className={styles.pagination}>
                <button
                  className={styles.pageButton}
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <span className={styles.pageInfo}>
                  Page {page} of {totalPages}
                </span>
                <button
                  className={styles.pageButton}
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </section>

      <div className={styles.footer}>
        <span>
          Total: {stack.total_quantity} item{stack.total_quantity !== 1 ? 's' : ''} in{' '}
          {stack.instance_count} instance{stack.instance_count !== 1 ? 's' : ''}
        </span>
      </div>

      {showCreate && (
        <CreateInstanceModal
          open={showCreate}
          onOpenChange={setShowCreate}
          locationId={locationId || null}
          parentInstanceId={parentInstanceId || null}
          onSave={async (data: CreateInstanceRequest) => {
            if (definitionId) {
              data.definition_id = definitionId
            }
            await createInstance(data)
            setShowCreate(false)
            queryClient.invalidateQueries({ queryKey: ['stacks'] })
            queryClient.invalidateQueries({ queryKey: ['browse'] })
            addToast('Item added', 'success')
          }}
        />
      )}

      {showMove && (
        <SimpleMoveDialog
          stack={stack}
          stackKey={stackKey}
          onClose={() => setShowMove(false)}
          onMoved={() => {
            setShowMove(false)
            queryClient.invalidateQueries({ queryKey: ['stacks'] })
            queryClient.invalidateQueries({ queryKey: ['browse'] })
            addToast('Items moved', 'success')
          }}
        />
      )}

      {showDelete && (
        <div
          className={styles.modalOverlay}
          onClick={() => setShowDelete(false)}
        >
          <div
            className={styles.modalContent}
            onClick={(e) => e.stopPropagation()}
          >
            <h3>Delete All {stack.definition_name}?</h3>
            <p>
              This will permanently remove {stack.instance_count} individual
              instance{stack.instance_count !== 1 ? 's' : ''} totalling{' '}
              {stack.total_quantity} item{stack.total_quantity !== 1 ? 's' : ''}
              .
            </p>
            <p className={styles.deleteWarning}>Type DELETE to confirm:</p>
            <input
              type="text"
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              className={styles.confirmInput}
              placeholder="DELETE"
            />
            <div className={styles.modalActions}>
              <button
                className={styles.modalCancelBtn}
                onClick={() => setShowDelete(false)}
              >
                Cancel
              </button>
              <button
                className={styles.modalDeleteBtn}
                disabled={deleteConfirm !== 'DELETE'}
                onClick={async () => {
                  try {
                    const { deleteStack } = await import('../api/stacks')
                    await deleteStack({
                      definition_id: definitionId,
                      location_id: locationId || undefined,
                      parent_instance_id: parentInstanceId || undefined,
                    })
                    setShowDelete(false)
                    setDeleteConfirm('')
                    queryClient.invalidateQueries({ queryKey: ['stacks'] })
                    queryClient.invalidateQueries({ queryKey: ['browse'] })
                    addToast('Stack deleted', 'success')
                  } catch {
                    addToast('Failed to delete items', 'error')
                  }
                }}
              >
                Delete All
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function SimpleMoveDialog({
  stack,
  stackKey,
  onClose,
  onMoved,
}: {
  stack: StackDetail
  stackKey: {
    definitionId: string
    locationId?: string
    parentInstanceId?: string
  }
  onClose: () => void
  onMoved: () => void
}) {
  const [quantity, setQuantity] = useState(1)
  const [targetLocationId, setTargetLocationId] = useState('')

  const { data: locations } = useQuery({
    queryKey: ['locations'],
    queryFn: async () => {
      const { fetchLocations } = await import('../api/locations')
      return fetchLocations()
    },
  })

  return (
    <div className={styles.modalOverlay} onClick={onClose}>
      <div
        className={styles.modalContent}
        onClick={(e) => e.stopPropagation()}
      >
        <h3>Move Items</h3>
        <p>
          Moving from stack: {stack.definition_name} ({stack.total_quantity}{' '}
          available)
        </p>
        <div className={styles.moveField}>
          <label>How many to move?</label>
          <input
            type="number"
            min={1}
            max={stack.total_quantity}
            value={quantity}
            onChange={(e) => setQuantity(Number(e.target.value))}
          />
        </div>
        <div className={styles.moveField}>
          <label>Target Location</label>
          <select
            value={targetLocationId}
            onChange={(e) => setTargetLocationId(e.target.value)}
          >
            <option value="">Select location...</option>
            {locations?.map((loc) => (
              <option key={loc.id} value={loc.id}>
                {loc.name}
              </option>
            ))}
          </select>
        </div>
        <div className={styles.modalActions}>
          <button className={styles.modalCancelBtn} onClick={onClose}>
            Cancel
          </button>
          <button
            className={styles.modalPrimaryBtn}
            disabled={
              !targetLocationId ||
              quantity < 1 ||
              quantity > stack.total_quantity
            }
            onClick={async () => {
              try {
                const { moveStack } = await import('../api/stacks')
                await moveStack({
                  definition_id: stackKey.definitionId,
                  source_location_id: stackKey.locationId || null,
                  source_parent_instance_id:
                    stackKey.parentInstanceId || null,
                  quantity,
                  target_location_id: targetLocationId,
                })
                onMoved()
              } catch (err) {
                // error handled by parent
              }
            }}
          >
            Move {quantity} item{quantity !== 1 ? 's' : ''}
          </button>
        </div>
      </div>
    </div>
  )
}
