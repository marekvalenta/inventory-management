import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { PlusIcon } from '@radix-ui/react-icons'
import { fetchDefinitions, createDefinition } from '../api/definitions'
import type { CreateDefinitionRequest } from '../api/definitions'
import { fetchTags } from '../api/tags'
import { TagBadge } from '../components/tags/TagBadge'
import { CreateDefinitionModal } from '../components/definitions/CreateDefinitionModal'
import { useToast } from '../context/ToastContext'
import styles from './DefinitionListPage.module.css'

export function DefinitionListPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [modalOpen, setModalOpen] = useState(false)

  const { data: definitions, isLoading, error } = useQuery({
    queryKey: ['definitions'],
    queryFn: fetchDefinitions,
  })

  const { data: tags } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateDefinitionRequest) => createDefinition(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['definitions'] })
      setModalOpen(false)
      addToast('Definition created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create definition', 'error')
    },
  })

  const handleAddClick = () => {
    setModalOpen(true)
  }

  const handleSave = (data: CreateDefinitionRequest) => {
    createMutation.mutate(data)
  }

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Definitions</h1>
        </div>
        <div className={styles.empty}>Loading definitions...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Definitions</h1>
        </div>
        <div className={styles.empty}>Failed to load definitions</div>
      </div>
    )
  }

  const defList = definitions ?? []
  const tagList = tags ?? []

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Definitions</h1>
        <button className={styles.addButton} onClick={handleAddClick}>
          <PlusIcon width={20} height={20} />
          <span>Add Definition</span>
        </button>
      </div>

      {defList.length === 0 && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No definitions yet — add your first definition</p>
          <button className={styles.emptyButton} onClick={handleAddClick}>
            <PlusIcon width={18} height={18} />
            <span>Add First Definition</span>
          </button>
        </div>
      )}

      {defList.length > 0 && (
        <div className={styles.list}>
          {defList.map((def) => (
            <Link key={def.id} to={`/definitions/${def.id}`} className={styles.card}>
              <div className={styles.cardTop}>
                <span className={styles.cardName}>{def.name}</span>
                {def.unit && <span className={styles.cardUnit}>{def.unit}</span>}
              </div>
              {def.tags.length > 0 && (
                <div className={styles.tagRow}>
                  {def.tags.map((tag) => (
                    <TagBadge key={tag.id} tag={tag} />
                  ))}
                </div>
              )}
              <div className={styles.cardBottom}>
                {def.total_instances} {def.total_instances === 1 ? 'instance' : 'instances'}
              </div>
            </Link>
          ))}
        </div>
      )}

      <CreateDefinitionModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        tags={tagList}
        onSave={handleSave}
        isPending={createMutation.isPending}
      />
    </div>
  )
}
