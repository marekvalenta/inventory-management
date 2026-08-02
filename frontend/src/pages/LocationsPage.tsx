import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PlusIcon } from '@radix-ui/react-icons'
import { fetchLocationTree, createLocation } from '../api/locations'
import type { CreateLocationRequest } from '../api/locations'
import { LocationTree } from '../components/locations/LocationTree'
import { CreateEditModal } from '../components/locations/CreateEditModal'
import { useToast } from '../context/ToastContext'
import styles from './LocationsPage.module.css'

export function LocationsPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()
  const [modalOpen, setModalOpen] = useState(false)
  const [createForParent, setCreateForParent] = useState<string | null>(null)

  const { data: tree, isLoading, error } = useQuery({
    queryKey: ['locations', 'tree'],
    queryFn: fetchLocationTree,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateLocationRequest) => createLocation(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      setModalOpen(false)
      setCreateForParent(null)
      addToast('Location created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create location', 'error')
    },
  })

  const handleAddChild = useCallback((parentId: string) => {
    setCreateForParent(parentId)
    setModalOpen(true)
  }, [])

  const handleAddRoot = () => {
    setCreateForParent(null)
    setModalOpen(true)
  }

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Locations</h1>
        </div>
        <div className={styles.loading}>Loading locations...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Locations</h1>
        </div>
        <div className={styles.error}>Failed to load locations</div>
      </div>
    )
  }

  const rootLocations = tree ?? []

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Locations</h1>
        <button className={styles.createButton} onClick={handleAddRoot}>
          <PlusIcon width={20} height={20} />
          <span>Add Top-Level</span>
        </button>
      </div>
      <LocationTree
        rootLocations={rootLocations}
        onAddChild={handleAddChild}
      />
      <CreateEditModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        parentId={createForParent}
        onSave={(data) => createMutation.mutate(data as CreateLocationRequest)}
      />
    </div>
  )
}
