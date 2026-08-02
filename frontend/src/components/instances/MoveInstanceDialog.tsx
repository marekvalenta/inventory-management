import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { fetchInstances, moveInstance } from '../../api/instances'
import { fetchLocations } from '../../api/locations'
import type { InstanceDetail, MoveInstanceRequest } from '../../api/instances'
import { useToast } from '../../context/ToastContext'
import { ApiError } from '../../api/client'
import styles from './InstanceModals.module.css'

interface MoveInstanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instance: InstanceDetail
}

export function MoveInstanceDialog({
  open,
  onOpenChange,
  instance,
}: MoveInstanceDialogProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [quantity, setQuantity] = useState(instance.quantity)
  const [moveMode, setMoveMode] = useState<'location' | 'container'>('location')
  const [targetLocationId, setTargetLocationId] = useState('')
  const [targetContainerId, setTargetContainerId] = useState('')
  const [containerSearch, setContainerSearch] = useState('')

  const { data: locations = [] } = useQuery({
    queryKey: ['locations'],
    queryFn: () => fetchLocations(),
    enabled: open && moveMode === 'location',
  })

  const { data: containerInstances } = useQuery({
    queryKey: ['instances'],
    queryFn: () => fetchInstances(),
    enabled: open && moveMode === 'container',
  })

  const filteredContainers = useMemo(() => {
    if (!containerInstances?.instances) return []
    const containers = containerInstances.instances
      .filter((inst) => inst.id !== instance.id && !inst.parent_instance_id)
    if (!containerSearch.trim()) return containers
    const q = containerSearch.toLowerCase()
    return containers.filter((c) => c.definition_name.toLowerCase().includes(q))
  }, [containerInstances, containerSearch, instance.id])

  const availableLocations = useMemo(() => {
    if (instance.location_id) {
      return locations.filter((l) => l.id !== instance.location_id)
    }
    return locations
  }, [locations, instance.location_id])

  const moveMutation = useMutation({
    mutationFn: (data: MoveInstanceRequest) => moveInstance(instance.id, data),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['instances'] })
      queryClient.invalidateQueries({ queryKey: ['locations'] })
      addToast(`Moved ${quantity} to ${result.target.definition_name}`, 'success')

      if (!result.source) {
        navigate(result.target.location_id
          ? `/locations/${result.target.location_id}`
          : `/instances/${result.target.id}`)
      } else {
        onOpenChange(false)
      }
    },
    onError: (err: Error) => {
      addToast(err.message || 'Move failed', 'error')
    },
  })

  function handleSubmit() {
    moveMutation.mutate({
      quantity,
      target_location_id: moveMode === 'location' ? targetLocationId || undefined : undefined,
      target_parent_instance_id: moveMode === 'container' ? targetContainerId || undefined : undefined,
    })
  }

  if (!open) return null

  return (
    <div className={styles.overlay} onClick={() => onOpenChange(false)}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h2 className={styles.title}>Move {instance.definition_name}</h2>

        <div className={styles.fieldGroup}>
          <label className={styles.label}>How many to move?</label>
          <input
            type="number"
            className={styles.input}
            min={1}
            max={instance.quantity}
            value={quantity}
            onChange={(e) => {
              const v = Math.max(1, Math.min(instance.quantity, parseInt(e.target.value) || 0))
              setQuantity(v)
            }}
          />
          <div className={styles.errorText} style={{ color: 'var(--color-text-secondary)' }}>
            Available: {instance.quantity}
          </div>
        </div>

        <div className={styles.fieldGroup}>
          <label className={styles.label}>Move to:</label>
          <div className={styles.moveModeToggle}>
            <button
              className={moveMode === 'location' ? styles.moveModeButtonActive : styles.moveModeButton}
              onClick={() => setMoveMode('location')}
            >
              Location
            </button>
            <button
              className={moveMode === 'container' ? styles.moveModeButtonActive : styles.moveModeButton}
              onClick={() => setMoveMode('container')}
            >
              Container
            </button>
          </div>
        </div>

        {moveMode === 'location' ? (
          <div className={styles.fieldGroup}>
            <label className={styles.label}>Target location</label>
            <select
              className={styles.select}
              value={targetLocationId}
              onChange={(e) => setTargetLocationId(e.target.value)}
            >
              <option value="">-- Select location --</option>
              {availableLocations.map((loc) => (
                <option key={loc.id} value={loc.id}>{loc.name}</option>
              ))}
            </select>
          </div>
        ) : (
          <div className={styles.fieldGroup}>
            <label className={styles.label}>Target container</label>
            <input
              type="text"
              className={styles.input}
              placeholder="Search containers..."
              value={containerSearch}
              onChange={(e) => setContainerSearch(e.target.value)}
            />
            <div className={styles.searchResults}>
              {filteredContainers.map((c) => (
                <div
                  key={c.id}
                  className={`${styles.searchItem} ${targetContainerId === c.id ? styles.moveModeButtonActive : ''}`}
                  onClick={() => setTargetContainerId(c.id)}
                >
                  {c.definition_name} x{c.quantity}
                  {c.location_name ? ` (${c.location_name})` : ''}
                </div>
              ))}
              {filteredContainers.length === 0 && (
                <div className={styles.searchItem} style={{ color: 'var(--color-text-disabled)' }}>
                  No containers found
                </div>
              )}
            </div>
          </div>
        )}

        <div className={styles.actions}>
          <button className={styles.cancelButton} onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button
            className={styles.saveButton}
            onClick={handleSubmit}
            disabled={moveMutation.isPending || (moveMode === 'location' ? !targetLocationId : !targetContainerId)}
          >
            {moveMutation.isPending ? 'Moving...' : 'Move'}
          </button>
        </div>
      </div>
    </div>
  )
}
