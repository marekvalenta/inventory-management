import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PlusIcon, Pencil1Icon, TrashIcon, Cross2Icon } from '@radix-ui/react-icons'
import { fetchTags, createTag, updateTag, deleteTag } from '../api/tags'
import type { Tag, CreateTagRequest, UpdateTagRequest } from '../api/tags'
import { TagBadge } from '../components/tags/TagBadge'
import { DeleteTagDialog } from '../components/tags/DeleteTagDialog'
import { useToast } from '../context/ToastContext'
import styles from './TagsPage.module.css'

type FormMode = 'create' | { edit: Tag }

function isValidHex(value: string): boolean {
  return /^#[0-9A-Fa-f]{6}$/.test(value)
}

export function TagsPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [formMode, setFormMode] = useState<FormMode | null>(null)
  const [formName, setFormName] = useState('')
  const [formColor, setFormColor] = useState('')
  const [formNameError, setFormNameError] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Tag | null>(null)

  const { data: tags, isLoading, error } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateTagRequest) => createTag(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tags'] })
      setFormMode(null)
      setFormName('')
      setFormColor('')
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
      setFormMode(null)
      setFormName('')
      setFormColor('')
      addToast('Tag updated', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to update tag', 'error')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTag(id),
    onSuccess: (_data, id) => {
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
    setFormMode('create')
    setFormName('')
    setFormColor('')
    setFormNameError('')
  }

  const handleStartEdit = (tag: Tag) => {
    setFormMode({ edit: tag })
    setFormName(tag.name)
    setFormColor(tag.color || '')
    setFormNameError('')
  }

  const handleCancel = () => {
    setFormMode(null)
    setFormName('')
    setFormColor('')
    setFormNameError('')
  }

  const validate = (): boolean => {
    const trimmed = formName.trim()
    if (trimmed.length < 2) {
      setFormNameError('Name must be at least 2 characters')
      return false
    }
    if (trimmed.length > 100) {
      setFormNameError('Name must be at most 100 characters')
      return false
    }
    setFormNameError('')
    return true
  }

  const handleSave = () => {
    if (!validate()) return

    const name = formName.trim()
    const color = formColor.trim() || null

    if (formMode === 'create') {
      createMutation.mutate({ name, color })
    } else if (typeof formMode === 'object') {
      const tag = formMode.edit
      const data: UpdateTagRequest = {}
      if (name !== tag.name) data.name = name
      if (color !== (tag.color || '')) data.color = color
      updateMutation.mutate({ id: tag.id, data })
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
  const showForm = formMode !== null

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Tags</h1>
        {!showForm && (
          <button className={styles.addButton} onClick={handleStartCreate}>
            <PlusIcon width={20} height={20} />
            <span>Add Tag</span>
          </button>
        )}
      </div>

      {!showForm && tagList.length === 0 && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No tags yet — add your first tag</p>
          <button className={styles.emptyButton} onClick={handleStartCreate}>
            <PlusIcon width={18} height={18} />
            <span>Add First Tag</span>
          </button>
        </div>
      )}

      {showForm && (
        <div className={styles.formCard}>
          <div className={styles.formRow}>
            <div className={styles.formField}>
              <label className={styles.formLabel} htmlFor="tag-name">Name</label>
              <input
                id="tag-name"
                className={styles.formInput}
                type="text"
                value={formName}
                onChange={(e) => {
                  setFormName(e.target.value)
                  setFormNameError('')
                }}
                onBlur={validate}
                placeholder="e.g. Fasteners, Fragile"
                autoFocus
                maxLength={100}
              />
              {formNameError && <span className={styles.fieldError}>{formNameError}</span>}
            </div>
            <div className={styles.formField}>
              <label className={styles.formLabel} htmlFor="tag-color">Color</label>
              <div className={styles.colorRow}>
                <input
                  id="tag-color"
                  className={styles.formInput}
                  type="text"
                  value={formColor}
                  onChange={(e) => setFormColor(e.target.value)}
                  placeholder="#FF5733"
                  maxLength={10}
                />
                <span
                  className={styles.colorSwatch}
                  style={{ backgroundColor: isValidHex(formColor) ? formColor : '#605C57' }}
                />
              </div>
            </div>
          </div>
          <div className={styles.formActions}>
            <button className={styles.cancelButton} onClick={handleCancel}>
              <Cross2Icon width={16} height={16} />
              <span>Cancel</span>
            </button>
            <button
              className={styles.saveButton}
              onClick={handleSave}
              disabled={createMutation.isPending || updateMutation.isPending}
            >
              {createMutation.isPending || updateMutation.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      )}

      {!showForm && tagList.length > 0 && (
        <button className={styles.inlineAdd} onClick={handleStartCreate}>
          <PlusIcon width={16} height={16} />
          <span>Add Tag</span>
        </button>
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
