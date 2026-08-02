import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { PlusIcon, Cross2Icon } from '@radix-ui/react-icons'
import { fetchDefinitions, createDefinition } from '../api/definitions'
import type { CreateDefinitionRequest, CreateFieldInput } from '../api/definitions'
import { fetchTags } from '../api/tags'
import type { Tag } from '../api/tags'
import { TagBadge } from '../components/tags/TagBadge'
import { FieldEditor } from '../components/definitions/FieldEditor'
import { useToast } from '../context/ToastContext'
import styles from './DefinitionListPage.module.css'

export function DefinitionListPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [showForm, setShowForm] = useState(false)
  const [formName, setFormName] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formUnit, setFormUnit] = useState('')
  const [formIsContainer, setFormIsContainer] = useState(false)
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([])
  const [formNameError, setFormNameError] = useState('')
  const [ownFields, setOwnFields] = useState<CreateFieldInput[]>([])

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
      setShowForm(false)
      resetForm()
      addToast('Definition created', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to create definition', 'error')
    },
  })

  const resetForm = () => {
    setFormName('')
    setFormDescription('')
    setFormUnit('')
    setFormIsContainer(false)
    setSelectedTagIds([])
    setFormNameError('')
    setOwnFields([])
  }

  const handleAddClick = () => {
    resetForm()
    setShowForm(true)
  }

  const handleCancel = () => {
    setShowForm(false)
    resetForm()
  }

  const addOwnField = () => {
    setOwnFields((prev) => [
      ...prev,
      {
        field_name: '',
        field_type: 'text',
        enum_values: null,
        is_required: false,
        display_order: prev.length,
        default_value: null,
        is_child_editable: false,
      },
    ])
  }

  const updateOwnField = (index: number, updates: Partial<CreateFieldInput>) => {
    setOwnFields((prev) =>
      prev.map((f, i) => (i === index ? { ...f, ...updates } : f)),
    )
  }

  const removeOwnField = (index: number) => {
    setOwnFields((prev) => prev.filter((_, i) => i !== index))
  }

  const moveField = (index: number, direction: 'up' | 'down') => {
    setOwnFields((prev) => {
      const newFields = [...prev]
      const targetIndex = direction === 'up' ? index - 1 : index + 1
      if (targetIndex < 0 || targetIndex >= newFields.length) return prev
      ;[newFields[index], newFields[targetIndex]] = [newFields[targetIndex], newFields[index]]
      return newFields.map((f, i) => ({ ...f, display_order: i }))
    })
  }

  const validate = (): boolean => {
    const trimmed = formName.trim()
    if (trimmed.length < 2) {
      setFormNameError('Name must be at least 2 characters')
      return false
    }
    if (trimmed.length > 200) {
      setFormNameError('Name must be at most 200 characters')
      return false
    }
    setFormNameError('')
    return true
  }

  const handleSave = () => {
    if (!validate()) return

    const unit = formUnit.trim() || null

    createMutation.mutate({
      name: formName.trim(),
      description: formDescription.trim() || null,
      unit: unit && unit.length <= 20 ? unit : null,
      is_container: formIsContainer,
      tag_ids: selectedTagIds,
      fields: ownFields.length > 0 ? ownFields : undefined,
    })
  }

  const toggleTag = (tagId: string) => {
    setSelectedTagIds((prev) =>
      prev.includes(tagId) ? prev.filter((id) => id !== tagId) : [...prev, tagId],
    )
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

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Definitions</h1>
        {!showForm && (
          <button className={styles.addButton} onClick={handleAddClick}>
            <PlusIcon width={20} height={20} />
            <span>Add</span>
          </button>
        )}
      </div>

      {!showForm && defList.length === 0 && (
        <div className={styles.empty}>
          <p className={styles.emptyText}>No definitions yet — add your first definition</p>
          <button className={styles.emptyButton} onClick={handleAddClick}>
            <PlusIcon width={18} height={18} />
            <span>Add First Definition</span>
          </button>
        </div>
      )}

      {showForm && (
        <div className={styles.formCard}>
          <div className={styles.formField}>
            <label className={styles.formLabel} htmlFor="def-name">Name</label>
            <input
              id="def-name"
              className={styles.formInput}
              type="text"
              value={formName}
              onChange={(e) => {
                setFormName(e.target.value)
                setFormNameError('')
              }}
              placeholder="e.g. Screw, Toolbox"
              autoFocus
              maxLength={200}
            />
            {formNameError && <span className={styles.fieldError}>{formNameError}</span>}
          </div>
          <div className={styles.formField}>
            <label className={styles.formLabel} htmlFor="def-desc">Description</label>
            <textarea
              id="def-desc"
              className={styles.formTextarea}
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder="Optional description"
              maxLength={2000}
            />
          </div>
          <div className={styles.formField}>
            <label className={styles.formLabel} htmlFor="def-unit">Unit</label>
            <input
              id="def-unit"
              className={styles.formInput}
              type="text"
              value={formUnit}
              onChange={(e) => setFormUnit(e.target.value)}
              placeholder="e.g. pcs, kg, m"
              maxLength={20}
            />
          </div>
          <label className={styles.formCheckbox}>
            <input
              type="checkbox"
              checked={formIsContainer}
              onChange={(e) => setFormIsContainer(e.target.checked)}
            />
            Is Container (can hold other items)
          </label>

          {tags && tags.length > 0 && (
            <div className={styles.formField}>
              <label className={styles.formLabel}>Tags</label>
              <div className={styles.tagRow}>
                {tags.map((tag: Tag) => (
                  <button
                    key={tag.id}
                    type="button"
                    onClick={() => toggleTag(tag.id)}
                    className={styles.tagRow}
                    style={{
                      opacity: selectedTagIds.includes(tag.id) ? 1 : 0.4,
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      padding: 0,
                    }}
                  >
                    <TagBadge tag={tag} size="md" />
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className={styles.formField}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <label className={styles.formLabel}>Fields</label>
            </div>
            {ownFields.map((field, idx) => (
              <FieldEditor
                key={idx}
                field={field}
                index={idx}
                total={ownFields.length}
                onChange={(updates) => updateOwnField(idx, updates)}
                onRemove={() => removeOwnField(idx)}
                onMoveUp={() => moveField(idx, 'up')}
                onMoveDown={() => moveField(idx, 'down')}
                showInheritedControls={true}
              />
            ))}
            <button className={styles.smallButton} onClick={addOwnField}>
              <PlusIcon width={14} height={14} />
              Add Field
            </button>
          </div>

          <div className={styles.formActions}>
            <button className={styles.cancelButton} onClick={handleCancel}>
              <Cross2Icon width={16} height={16} />
              <span>Cancel</span>
            </button>
            <button
              className={styles.saveButton}
              onClick={handleSave}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>
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
    </div>
  )
}
