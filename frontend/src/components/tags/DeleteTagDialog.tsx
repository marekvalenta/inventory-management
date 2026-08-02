import { useState, useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import type { Tag } from '../../api/tags'
import styles from './DeleteTagDialog.module.css'

interface DeleteTagDialogProps {
  open: boolean
  tag: Tag | null
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
}

export function DeleteTagDialog({ open, tag, onOpenChange, onConfirm }: DeleteTagDialogProps) {
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    if (!open) {
      setConfirming(false)
    }
  }, [open])

  if (!tag) return null

  const linkedCount = tag.linked_definitions_count

  const handleConfirm = () => {
    setConfirming(true)
    onConfirm()
  }

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.content}>
          <Dialog.Title className={styles.title}>Delete Tag</Dialog.Title>
          <Dialog.Description className={styles.description}>
            {linkedCount > 0
              ? `Tag '${tag.name}' is used by ${linkedCount} item definition${linkedCount === 1 ? '' : 's'}. Deleting it will remove this tag from all of them.`
              : `Delete tag '${tag.name}'?`}
          </Dialog.Description>
          <div className={styles.actions}>
            <Dialog.Close asChild>
              <button type="button" className={styles.cancelButton} disabled={confirming}>
                Cancel
              </button>
            </Dialog.Close>
            <button
              type="button"
              className={styles.deleteButton}
              onClick={handleConfirm}
              disabled={confirming}
            >
              {linkedCount > 0 ? 'Delete Anyway' : 'Delete'}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
