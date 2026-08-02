import { useState, useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Cross2Icon } from '@radix-ui/react-icons'
import type { Location, CreateLocationRequest, UpdateLocationRequest } from '../../api/locations'
import styles from './CreateEditModal.module.css'

interface CreateEditModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSave: (data: CreateLocationRequest | UpdateLocationRequest, id?: string) => void
  location?: Location | null
  parentId?: string | null
}

export function CreateEditModal({
  open,
  onOpenChange,
  onSave,
  location,
  parentId,
}: CreateEditModalProps) {
  const isEdit = !!location
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [nameError, setNameError] = useState('')
  const [descError, setDescError] = useState('')

  useEffect(() => {
    if (open) {
      setName(location?.name ?? '')
      setDescription(location?.description ?? '')
      setNameError('')
      setDescError('')
    }
  }, [open, location])

  const validate = (): boolean => {
    let valid = true
    const trimmed = name.trim()
    if (trimmed.length < 2) {
      setNameError('Name must be at least 2 characters')
      valid = false
    } else if (trimmed.length > 200) {
      setNameError('Name must be at most 200 characters')
      valid = false
    } else {
      setNameError('')
    }
    if (description.length > 2000) {
      setDescError('Description must be at most 2000 characters')
      valid = false
    } else {
      setDescError('')
    }
    return valid
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!validate()) return

    const data: CreateLocationRequest = {
      name: name.trim(),
      description: description.trim() || null,
    }

    if (!isEdit) {
      ;(data as CreateLocationRequest).parent_id = parentId ?? null
    }

    onSave(data, location?.id)
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.content}>
          <Dialog.Title className={styles.title}>
            {isEdit ? 'Edit Location' : 'Add Sub-Location'}
          </Dialog.Title>
          <Dialog.Description className={styles.description}>
            {isEdit
              ? 'Update the location name and description.'
              : 'Create a new location inside the current one.'}
          </Dialog.Description>
          <form onSubmit={handleSubmit}>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="loc-name">
                Name
              </label>
              <input
                id="loc-name"
                className={styles.input}
                type="text"
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  setNameError('')
                }}
                placeholder="e.g. Living Room, Shelf A"
                autoFocus
              />
              {nameError && <span className={styles.error}>{nameError}</span>}
            </div>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="loc-desc">
                Description
              </label>
              <textarea
                id="loc-desc"
                className={styles.textarea}
                value={description}
                onChange={(e) => {
                  setDescription(e.target.value)
                  setDescError('')
                }}
                placeholder="Optional description"
                rows={3}
              />
              {descError && <span className={styles.error}>{descError}</span>}
            </div>
            <div className={styles.actions}>
              <Dialog.Close asChild>
                <button type="button" className={styles.cancelButton}>
                  Cancel
                </button>
              </Dialog.Close>
              <button type="submit" className={styles.saveButton}>
                {isEdit ? 'Save Changes' : 'Create'}
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
