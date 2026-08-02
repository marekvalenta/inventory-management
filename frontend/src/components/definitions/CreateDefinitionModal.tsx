import { useState, useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Cross2Icon, PlusIcon } from '@radix-ui/react-icons'
import type { CreateDefinitionRequest, CreateFieldInput } from '../../api/definitions'
import type { Tag } from '../../api/tags'
import { TagBadge } from '../tags/TagBadge'
import { FieldEditor } from '../definitions/FieldEditor'
import styles from './CreateDefinitionModal.module.css'

interface CreateDefinitionModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tags: Tag[]
  onSave: (data: CreateDefinitionRequest) => void
  isPending: boolean
}

export function CreateDefinitionModal({
  open,
  onOpenChange,
  tags,
  onSave,
  isPending,
}: CreateDefinitionModalProps) {
  const [formName, setFormName] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formUnit, setFormUnit] = useState('')
  const [formIsContainer, setFormIsContainer] = useState(false)
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([])
  const [formNameError, setFormNameError] = useState('')
  const [ownFields, setOwnFields] = useState<CreateFieldInput[]>([])

  useEffect(() => {
    if (open) {
      setFormName('')
      setFormDescription('')
      setFormUnit('')
      setFormIsContainer(false)
      setSelectedTagIds([])
      setFormNameError('')
      setOwnFields([])
    }
  }, [open])

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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!validate()) return

    const unit = formUnit.trim() || null
    onSave({
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

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.content}>
          <Dialog.Title className={styles.title}>
            Add Definition
          </Dialog.Title>
          <form onSubmit={handleSubmit} className={styles.form}>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="def-name">Name</label>
              <input
                id="def-name"
                className={styles.input}
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
              {formNameError && <span className={styles.error}>{formNameError}</span>}
            </div>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="def-desc">Description</label>
              <textarea
                id="def-desc"
                className={styles.textarea}
                value={formDescription}
                onChange={(e) => setFormDescription(e.target.value)}
                placeholder="Optional description"
                maxLength={2000}
              />
            </div>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="def-unit">Unit</label>
              <input
                id="def-unit"
                className={styles.input}
                type="text"
                value={formUnit}
                onChange={(e) => setFormUnit(e.target.value)}
                placeholder="e.g. pcs, kg, m"
                maxLength={20}
              />
            </div>
            <label className={styles.checkbox}>
              <input
                type="checkbox"
                checked={formIsContainer}
                onChange={(e) => setFormIsContainer(e.target.checked)}
              />
              Is Container (can hold other items)
            </label>

            {tags.length > 0 && (
              <div className={styles.field}>
                <label className={styles.label}>Tags</label>
                <div className={styles.tagRow}>
                  {tags.map((tag) => (
                    <button
                      key={tag.id}
                      type="button"
                      onClick={() => toggleTag(tag.id)}
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

            <div className={styles.field}>
              <label className={styles.label}>Fields</label>
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
              <button className={styles.smallButton} type="button" onClick={addOwnField}>
                <PlusIcon width={14} height={14} />
                Add Field
              </button>
            </div>

            <div className={styles.actions}>
              <Dialog.Close asChild>
                <button type="button" className={styles.cancelButton}>
                  Cancel
                </button>
              </Dialog.Close>
              <button type="submit" className={styles.saveButton} disabled={isPending}>
                {isPending ? 'Creating...' : 'Create'}
              </button>
            </div>
          </form>
          <Dialog.Close asChild>
            <button className={styles.closeButton} aria-label="Close">
              <Cross2Icon width={18} height={18} />
            </button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
