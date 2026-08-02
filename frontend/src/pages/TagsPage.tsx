import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PlusIcon, Pencil1Icon, TrashIcon } from '@radix-ui/react-icons'
import { fetchTags, createTag, updateTag, deleteTag } from '../api/tags'
import type { Tag, CreateTagRequest, UpdateTagRequest } from '../api/tags'
import { TagBadge } from '../components/tags/TagBadge'
import { DeleteTagDialog } from '../components/tags/DeleteTagDialog'
import { CreateEditTagModal } from '../components/tags/CreateEditTagModal'
import { useToast } from '../context/ToastContext'
import styles from './TagsPage.module.css'

export function TagsPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [modalOpen, setModalOpen] = useState(false)
  const [editingTag, setEditingTag] = useState<Tag | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Tag | null>(null)

  const { data: tags, isLoading, error } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateTagRequest) => createTag(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
      setModalOpen(false)
      setEditingTag(null)
      addToast('Tag created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create tag', 'error')
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTagRequest }) => updateTag(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
      setModalOpen(false)
      setEditingTag(null)
      addToast('Tag updated', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to update tag', 'error')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTag(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
      setDeleteTarget(null)
      addToast('Tag deleted', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to delete tag', 'error')
      setDeleteTarget(null)
    },
  })

  const handleStartCreate = () => {
    setEditingTag(null)
    setModalOpen(true)
  }

  const handleStartEdit = (tag: Tag) => {
    setEditingTag(tag)
    setModalOpen(true)
  }

  const handleModalSave = (data: CreateTagRequest | UpdateTagRequest) => {
    if (editingTag) {
      updateMutation.mutate({ id: editingTag.id, data: data as UpdateTagRequest })
    } else {
      createMutation.mutate(data as CreateTagRequest)
    }
  }

  const handleDeleteClick = (tag: Tag) => {
    setDeleteTarget(tag)
  }

  const handleConfirmDelete = () => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id)
    }
  }

  if (isLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Tags</h1>
        </div>
        <div className={styles.empty}>Loading tags...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h1 className={styles.heading}>Tags</h1>
        </div>
        <div className={styles.empty}>Failed to load tags</div>
      </div>
    )
  }

  const tagList = tags ?? []

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Tags</h1>
        <button className={styles.addButton} onClick={handleStartCreate}>
          <PlusIcon width={20} height={20} />
          <span>Add Tag</span>
        </button>
      </div>

      {tagList.length === 0 && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No tags yet — add your first tag</p>
          <button className={styles.emptyButton} onClick={handleStartCreate}>
            <PlusIcon width={18} height={18} />
            <span>Add First Tag</span>
          </button>
        </div>
      )}

      {tagList.length > 0 && (
        <div className={styles.list}>
          {tagList.map((tag) => (
            <div key={tag.id} className={styles.tagRow}>
              <div className={styles.tagInfo}>
                <TagBadge tag={tag} size="md" />
                {tag.linked_definitions_count > 0 && (
                  <span className={styles.defCount}>
                    {tag.linked_definitions_count}{' '}
                    {tag.linked_definitions_count === 1 ? 'definition' : 'definitions'}
                  </span>
                )}
              </div>
              <div className={styles.tagActions}>
                <button
                  className={styles.actionButton}
                  onClick={() => handleStartEdit(tag)}
                  aria-label={`Edit ${tag.name}`}
                >
                  <Pencil1Icon width={18} height={18} />
                </button>
                <button
                  className={styles.actionButton}
                  onClick={() => handleDeleteClick(tag)}
                  aria-label={`Delete ${tag.name}`}
                >
                  <TrashIcon width={18} height={18} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <CreateEditTagModal
        open={modalOpen}
        onOpenChange={(open) => {
          setModalOpen(open)
          if (!open) setEditingTag(null)
        }}
        tag={editingTag}
        onSave={handleModalSave}
      />

      <DeleteTagDialog
        open={deleteTarget !== null}
        tag={deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
        onConfirm={handleConfirmDelete}
      />
    </div>
  )
}
