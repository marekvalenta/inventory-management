import { useState, useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Cross2Icon } from '@radix-ui/react-icons'
import type { Tag, CreateTagRequest, UpdateTagRequest } from '../../api/tags'
import styles from './CreateEditTagModal.module.css'

interface CreateEditTagModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tag: Tag | null
  onSave: (data: CreateTagRequest | UpdateTagRequest) => void
}

function isValidHex(value: string): boolean {
  return /^#[0-9A-Fa-f]{6}$/.test(value)
}

export function CreateEditTagModal({
  open,
  onOpenChange,
  tag,
  onSave,
}: CreateEditTagModalProps) {
  const isEdit = tag !== null
  const [name, setName] = useState('')
  const [color, setColor] = useState('')
  const [nameError, setNameError] = useState('')

  useEffect(() => {
    if (open) {
      setName(tag?.name ?? '')
      setColor(tag?.color ?? '')
      setNameError('')
    }
  }, [open, tag])

  const validate = (): boolean => {
    const trimmed = name.trim()
    if (trimmed.length < 2) {
      setNameError('Name must be at least 2 characters')
      return false
    }
    if (trimmed.length > 100) {
      setNameError('Name must be at most 100 characters')
      return false
    }
    setNameError('')
    return true
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!validate()) return

    const trimmedName = name.trim()
    const trimmedColor = color.trim() || null

    if (isEdit) {
      const data: UpdateTagRequest = {}
      if (trimmedName !== tag!.name) data.name = trimmedName
      if (trimmedColor !== (tag!.color ?? null)) data.color = trimmedColor
      onSave(data)
    } else {
      onSave({ name: trimmedName, color: trimmedColor })
    }
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.content}>
          <Dialog.Title className={styles.title}>
            {isEdit ? 'Edit Tag' : 'Add Tag'}
          </Dialog.Title>
          <form onSubmit={handleSubmit}>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="tag-name">
                Name
              </label>
              <input
                id="tag-name"
                className={styles.input}
                type="text"
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  setNameError('')
                }}
                placeholder="e.g. Fasteners, Fragile"
                autoFocus
                maxLength={100}
              />
              {nameError && <span className={styles.error}>{nameError}</span>}
            </div>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="tag-color">
                Color
              </label>
              <div className={styles.colorRow}>
                <input
                  id="tag-color"
                  className={styles.input}
                  type="text"
                  value={color}
                  onChange={(e) => setColor(e.target.value)}
                  placeholder="#FF5733"
                  maxLength={10}
                />
                <span
                  className={styles.colorSwatch}
                  style={{ backgroundColor: isValidHex(color) ? color : '#605C57' }}
                />
              </div>
            </div>
            <div className={styles.actions}>
              <Dialog.Close asChild>
                <button type="button" className={styles.cancelButton}>
                  Cancel
                </button>
              </Dialog.Close>
              <button type="submit" className={styles.saveButton}>
                {isEdit ? 'Save Changes' : 'Save'}
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
