import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  PlusIcon,
  Pencil1Icon,
  TrashIcon,
  CubeIcon,
  ArchiveIcon,
} from '@radix-ui/react-icons'
import { fetchStacks } from '../api/stacks'
import {
  fetchLocation,
  fetchLocationContents,
  fetchLocationBreadcrumb,
  updateLocation,
  deleteLocation,
} from '../api/locations'
import type {
  UpdateLocationRequest,
  CreateLocationRequest,
} from '../api/locations'
import { Breadcrumb } from '../components/breadcrumb/Breadcrumb'
import { LocationTree } from '../components/locations/LocationTree'
import { CreateEditModal } from '../components/locations/CreateEditModal'
import { DeleteDialog } from '../components/locations/DeleteDialog'
import { CreateInstanceModal } from '../components/instances/CreateInstanceModal'
import { createInstance } from '../api/instances'
import type { CreateInstanceRequest } from '../api/instances'
import { ApiError } from '../api/client'
import { useToast } from '../context/ToastContext'
import styles from './LocationDetailPage.module.css'

export function LocationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [createInstanceOpen, setCreateInstanceOpen] = useState(false)

  const {
    data: location,
    isLoading: locationLoading,
    error: locationError,
  } = useQuery({
    queryKey: ['locations', id],
    queryFn: () => fetchLocation(id!),
    enabled: !!id,
  })

  const { data: contents, isLoading: contentsLoading } = useQuery({
    queryKey: ['locations', id, 'contents'],
    queryFn: () => fetchLocationContents(id!),
    enabled: !!id,
  })

  const { data: stacksData } = useQuery({
    queryKey: ['stacks', { location_id: id }],
    queryFn: () => fetchStacks({ location_id: id! }),
    enabled: !!id,
  })

  const { data: breadcrumb } = useQuery({
    queryKey: ['locations', id, 'breadcrumb'],
    queryFn: () => fetchLocationBreadcrumb(id!),
    enabled: !!id,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateLocationRequest) =>
      import('../api/locations').then((m) => m.createLocation(data)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setCreateModalOpen(false)
      addToast('Sub-location created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create sub-location', 'error')
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (data: UpdateLocationRequest) => {
      if (!id) throw new Error('No location ID')
      return updateLocation(id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setEditModalOpen(false)
      addToast('Location updated', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to update location', 'error')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!id) throw new Error('No location ID')
      return deleteLocation(id)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setDeleteDialogOpen(false)
      addToast('Location deleted', 'success')
      navigate('/locations')
    },
    onError: (err: unknown) => {
      setDeleteDialogOpen(false)
      if (err instanceof ApiError && err.status === 409) {
        addToast(
          `Cannot delete "${location?.name}": it contains sub-locations or items. Move them first.`,
          'error',
        )
      } else if (err instanceof Error) {
        addToast(err.message || 'Failed to delete location', 'error')
      } else {
        addToast('Failed to delete location', 'error')
      }
    },
  })

  const createInstanceMutation = useMutation({
    mutationFn: (data: CreateInstanceRequest) => createInstance(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      setCreateInstanceOpen(false)
      addToast('Item added', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to add item', 'error')
    },
  })

  if (locationLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading location...</div>
      </div>
    )
  }

  if (locationError || !location) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Location not found</div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      {breadcrumb && breadcrumb.length > 0 && (
        <Breadcrumb nodes={breadcrumb} />
      )}

      <div className={styles.header}>
        <div>
          <h1 className={styles.heading}>{location.name}</h1>
          {location.description && (
            <p className={styles.description}>{location.description}</p>
          )}
        </div>
        <div className={styles.headerActions}>
          <button
            className={styles.actionButton}
            onClick={() => setCreateModalOpen(true)}
            title="Add sub-location"
          >
            <PlusIcon width={20} height={20} />
            <span className={styles.buttonLabel}>Add Sub</span>
          </button>
          <button
            className={styles.actionButton}
            onClick={() => setEditModalOpen(true)}
            title="Edit location"
          >
            <Pencil1Icon width={20} height={20} />
            <span className={styles.buttonLabel}>Edit</span>
          </button>
          <button
            className={styles.deleteButton}
            onClick={() => setDeleteDialogOpen(true)}
            title="Delete location"
          >
            <TrashIcon width={20} height={20} />
            <span className={styles.buttonLabel}>Delete</span>
          </button>
        </div>
      </div>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>
          <ArchiveIcon width={20} height={20} />
          Sub-locations
        </h2>
        {contentsLoading ? (
          <div className={styles.loading}>Loading...</div>
        ) : contents?.sub_locations && contents.sub_locations.length > 0 ? (
          <LocationTree
            rootLocations={contents.sub_locations}
            onAddChild={(parentId) => {
              navigate(`/locations/${parentId}`)
            }}
          />
        ) : (
          <p className={styles.empty}>No sub-locations yet</p>
        )}
      </section>

      <section className={styles.section}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 'var(--space-md)', paddingBottom: 'var(--space-sm)', borderBottom: '1px solid var(--color-border)' }}>
          <h2 className={styles.sectionTitle} style={{ margin: 0, padding: 0, border: 'none' }}>
            <CubeIcon width={20} height={20} />
            Items
          </h2>
          <button
            className={styles.actionButton}
            onClick={() => setCreateInstanceOpen(true)}
            title="Add item"
          >
            <PlusIcon width={20} height={20} />
            Add Item
          </button>
        </div>
        {stacksData?.stacks && stacksData.stacks.length > 0 ? (
          <div className={styles.instanceList}>
            {stacksData.stacks.map((stack) => {
              const params = new URLSearchParams()
              params.set('definition_id', stack.definition_id)
              params.set('location_id', id!)
              return (
                <a
                  key={stack.definition_id}
                  href={`/stacks?${params.toString()}`}
                  className={styles.instanceItem}
                  onClick={(e) => {
                    e.preventDefault()
                    navigate(`/stacks?${params.toString()}`)
                  }}
                >
                  <span className={styles.instanceName}>
                    {stack.definition_name}
                  </span>
                  <div className={styles.stackMeta}>
                    <span className={styles.instanceQuantity}>
                      x{stack.total_quantity}
                    </span>
                    <span className={styles.countLabel}>
                      {stack.instance_count} instance{stack.instance_count !== 1 ? 's' : ''}
                    </span>
                  </div>
                </a>
              )
            })}
          </div>
        ) : (
          <p className={styles.empty}>No items in this location</p>
        )}
      </section>

      <CreateEditModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        parentId={id ?? null}
        onSave={(data) => createMutation.mutate(data as CreateLocationRequest)}
      />

      <CreateEditModal
        open={editModalOpen}
        onOpenChange={setEditModalOpen}
        location={location}
        onSave={(data, locId) => {
          if (locId) {
            updateMutation.mutate(data as UpdateLocationRequest)
          }
        }}
      />

      <DeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        locationName={location.name}
        onConfirm={() => deleteMutation.mutate()}
      />

      <CreateInstanceModal
        open={createInstanceOpen}
        onOpenChange={setCreateInstanceOpen}
        locationId={id ?? null}
        onSave={(data) => createInstanceMutation.mutate(data)}
      />
    </div>
  )
}
