import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Pencil1Icon,
  TrashIcon,
  MoveIcon,
  CubeIcon,
  Link2Icon,
  PlusIcon,
} from '@radix-ui/react-icons'
import {
  fetchInstance,
  fetchInstanceContents,
  deleteInstance,
  updateInstance,
  createInstance,
} from '../api/instances'
import type {
  UpdateInstanceRequest,
  CreateInstanceRequest,
} from '../api/instances'
import { Breadcrumb } from '../components/breadcrumb/Breadcrumb'
import { CreateInstanceModal } from '../components/instances/CreateInstanceModal'
import { EditInstanceModal } from '../components/instances/EditInstanceModal'
import { MoveInstanceDialog } from '../components/instances/MoveInstanceDialog'
import { DeleteInstanceDialog } from '../components/instances/DeleteInstanceDialog'
import { ApiError } from '../api/client'
import { useToast } from '../context/ToastContext'
import styles from './InstanceDetailPage.module.css'

export function InstanceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [editModalOpen, setEditModalOpen] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [moveDialogOpen, setMoveDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const {
    data: instance,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['instances', id],
    queryFn: () => fetchInstance(id!),
    enabled: !!id,
  })

  const { data: contents } = useQuery({
    queryKey: ['instances', id, 'contents'],
    queryFn: () => fetchInstanceContents(id!),
    enabled: !!id,
  })

  const updateMutation = useMutation({
    mutationFn: async (data: UpdateInstanceRequest) => {
      if (!id) throw new Error('No instance ID')
      return updateInstance(id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', id] })
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      setEditModalOpen(false)
      addToast('Instance updated', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to update instance', 'error')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!id) throw new Error('No instance ID')
      return deleteInstance(id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setDeleteDialogOpen(false)
      addToast('Instance deleted', 'success')
      if (instance?.location_id) {
        navigate(`/locations/${instance.location_id}`)
      } else if (instance?.parent_instance_id) {
        navigate(`/instances/${instance.parent_instance_id}`)
      } else {
        navigate('/locations')
      }
    },
    onError: (err: unknown) => {
      setDeleteDialogOpen(false)
      if (err instanceof Error) {
        addToast(err.message || 'Failed to delete instance', 'error')
      } else {
        addToast('Failed to delete instance', 'error')
      }
    },
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateInstanceRequest) => createInstance(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', id] })
      queryClient.invalidateQueries({ queryKey: ['instances', id, 'contents'] })
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      setCreateModalOpen(false)
      addToast('Item added', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to add item', 'error')
    },
  })

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading instance...</div>
      </div>
    )
  }

  if (error || !instance) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <div className={styles.container}>
          <div className={styles.error}>Instance not found</div>
        </div>
      )
    }
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          {error instanceof Error ? error.message : 'Failed to load instance'}
        </div>
      </div>
    )
  }

  const isContainer = instance.child_instance_count !== undefined

  return (
    <div className={styles.container}>
      {instance.breadcrumb && instance.breadcrumb.length > 0 && (
        <Breadcrumb nodes={instance.breadcrumb} />
      )}

      <div className={styles.header}>
        <div className={styles.headerInfo}>
          <h1 className={styles.heading}>{instance.definition_name}</h1>
          <Link to={`/definitions/${instance.definition_id}`} className={styles.definitionLink}>
            <Link2Icon width={16} height={16} />
            View definition
          </Link>
          <div className={styles.metaRow}>
            <span className={styles.quantityBadge}>
              <CubeIcon width={16} height={16} />
              x{instance.quantity}
            </span>
            {instance.unit && (
              <span className={styles.unitBadge}>{instance.unit}</span>
            )}
          </div>
        </div>
        <div className={styles.headerActions}>
          <button
            className={styles.actionButton}
            onClick={() => setEditModalOpen(true)}
            title="Edit instance"
          >
            <Pencil1Icon width={20} height={20} />
            Edit
          </button>
          <button
            className={styles.actionButton}
            onClick={() => setMoveDialogOpen(true)}
            title="Move instance"
          >
            <MoveIcon width={20} height={20} />
            Move
          </button>
          <button
            className={styles.deleteButton}
            onClick={() => setDeleteDialogOpen(true)}
            title="Delete instance"
          >
            <TrashIcon width={20} height={20} />
            Delete
          </button>
        </div>
      </div>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Placement</h2>
        <div className={styles.placementInfo}>
          {instance.location_id ? (
            <>
              Located in:{' '}
              <Link to={`/locations/${instance.location_id}`} className={styles.placementLink}>
                {instance.location_name || 'Unknown location'}
              </Link>
            </>
          ) : instance.parent_instance_id ? (
            <>
              Inside:{' '}
              <Link to={`/instances/${instance.parent_instance_id}`} className={styles.placementLink}>
                {instance.parent_instance_name || 'Unknown container'}
              </Link>
              {instance.location_name && (
                <> (in{' '}
                  <Link to={`/locations/${instance.location_id}`} className={styles.placementLink}>
                    {instance.location_name}
                  </Link>
                  )
                </>
              )}
            </>
          ) : null}
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Field Values</h2>
        {instance.field_values.length > 0 ? (
          <div className={styles.fieldTable}>
            {instance.field_values.map((fv) => (
              <div key={fv.field_id} className={styles.fieldRow}>
                <div>
                  <div className={styles.fieldName}>
                    {fv.field_name}
                  </div>
                  <div className={styles.fieldType}>{fv.field_type}</div>
                </div>
                <div className={fv.value !== null ? styles.fieldValue : styles.fieldValueEmpty}>
                  {fv.value !== null ? fv.value : 'No value'}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className={styles.contentsEmpty}>No fields defined for this definition</p>
        )}
      </section>

      {isContainer && instance.child_instance_count >= 0 && (
        <section className={styles.section}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 'var(--space-md)', paddingBottom: 'var(--space-sm)', borderBottom: '1px solid var(--color-border)' }}>
            <h2 className={styles.sectionTitle} style={{ margin: 0, padding: 0, border: 'none' }}>
              Items inside ({instance.child_instance_count})
            </h2>
            <button
              className={styles.actionButton}
              onClick={() => setCreateModalOpen(true)}
              title="Add item to container"
            >
              <PlusIcon width={20} height={20} />
              Add Item
            </button>
          </div>
          {contents?.instances && contents.instances.length > 0 ? (
            <div className={styles.contentsList}>
              {contents.instances.map((child) => (
                <a
                  key={child.id}
                  href={`/instances/${child.id}`}
                  className={styles.contentItem}
                  onClick={(e) => {
                    e.preventDefault()
                    navigate(`/instances/${child.id}`)
                  }}
                >
                  <span className={styles.contentName}>
                    {child.definition_name}
                  </span>
                  <span className={styles.contentQuantity}>
                    x{child.quantity}
                  </span>
                </a>
              ))}
            </div>
          ) : (
            <p className={styles.contentsEmpty}>Container is empty</p>
          )}
        </section>
      )}

      <div className={styles.footer}>
        <span>Created: {new Date(instance.created_at).toLocaleString()}</span>
        <span>Updated: {new Date(instance.updated_at).toLocaleString()}</span>
      </div>

      {editModalOpen && (
        <EditInstanceModal
          open={editModalOpen}
          onOpenChange={setEditModalOpen}
          instance={instance}
          onSave={(data) => updateMutation.mutate(data)}
        />
      )}

      {createModalOpen && (
        <CreateInstanceModal
          open={createModalOpen}
          onOpenChange={setCreateModalOpen}
          parentInstanceId={id}
          onSave={(data) => createMutation.mutate(data)}
        />
      )}

      {moveDialogOpen && (
        <MoveInstanceDialog
          open={moveDialogOpen}
          onOpenChange={setMoveDialogOpen}
          instance={instance}
        />
      )}

      {deleteDialogOpen && (
        <DeleteInstanceDialog
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          instanceName={instance.definition_name}
          childCount={instance.child_instance_count}
          quantity={instance.quantity}
          onConfirm={() => deleteMutation.mutate()}
        />
      )}
    </div>
  )
}
