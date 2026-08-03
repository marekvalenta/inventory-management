import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PlusIcon } from '@radix-ui/react-icons'
import { fetchBrowse, createLocation } from '../api/locations'
import { createInstance } from '../api/instances'
import type { CreateLocationRequest } from '../api/locations'
import type { CreateInstanceRequest } from '../api/instances'
import { BrowseTree } from '../components/locations/BrowseTree'
import { CreateEditModal } from '../components/locations/CreateEditModal'
import { CreateInstanceModal } from '../components/instances/CreateInstanceModal'
import { useToast } from '../context/ToastContext'
import styles from './LocationsPage.module.css'

export function LocationsPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()
  const [locationModalOpen, setLocationModalOpen] = useState(false)
  const [createForParent, setCreateForParent] = useState<string | null>(null)

  const [instanceModalOpen, setInstanceModalOpen] = useState(false)
  const [instanceLocationId, setInstanceLocationId] = useState<string | null>(null)
  const [instanceParentId, setInstanceParentId] = useState<string | null>(null)

  const { data: browse, isLoading, error } = useQuery({
    queryKey: ['browse'],
    queryFn: fetchBrowse,
  })

  const createLocationMutation = useMutation({
    mutationFn: (data: CreateLocationRequest) => createLocation(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browse'] })
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setLocationModalOpen(false)
      setCreateForParent(null)
      addToast('Location created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create location', 'error')
    },
  })

  const createInstanceMutation = useMutation({
    mutationFn: (data: CreateInstanceRequest) => createInstance(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browse'] })
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      setInstanceModalOpen(false)
      setInstanceLocationId(null)
      setInstanceParentId(null)
      addToast('Item added', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to add item', 'error')
    },
  })

  const handleAddLocation = useCallback((parentId: string) => {
    setCreateForParent(parentId)
    setLocationModalOpen(true)
  }, [])

  const handleAddRoot = () => {
    setCreateForParent(null)
    setLocationModalOpen(true)
  }

  const handleAddInstance = useCallback(
    (locationId: string | null, parentInstanceId: string | null) => {
      setInstanceLocationId(locationId)
      setInstanceParentId(parentInstanceId)
      setInstanceModalOpen(true)
    },
    [],
  )

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Browse</h1>
        </div>
        <div className={styles.loading}>Loading locations...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Browse</h1>
        </div>
        <div className={styles.error}>Failed to load locations</div>
      </div>
    )
  }

  const rootNodes = browse ?? []

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Locations</h1>
        <button className={styles.createButton} onClick={handleAddRoot}>
          <PlusIcon width={20} height={20} />
          <span>Add Top-Level</span>
        </button>
      </div>
      <BrowseTree
        rootNodes={rootNodes}
        onAddLocation={handleAddLocation}
        onAddInstance={handleAddInstance}
      />
      <CreateEditModal
        open={locationModalOpen}
        onOpenChange={setLocationModalOpen}
        parentId={createForParent}
        onSave={(data) => createLocationMutation.mutate(data as CreateLocationRequest)}
      />
      <CreateInstanceModal
        open={instanceModalOpen}
        onOpenChange={setInstanceModalOpen}
        locationId={instanceLocationId}
        parentInstanceId={instanceParentId}
        onSave={(data) => createInstanceMutation.mutate(data)}
      />
    </div>
  )
}
