import * as AlertDialog from '@radix-ui/react-alert-dialog'
import styles from './DeleteDialog.module.css'

interface DeleteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  locationName: string
}

export function DeleteDialog({
  open,
  onOpenChange,
  onConfirm,
  locationName,
}: DeleteDialogProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className={styles.overlay} />
        <AlertDialog.Content className={styles.content}>
          <AlertDialog.Title className={styles.title}>
            Delete "{locationName}"?
          </AlertDialog.Title>
          <AlertDialog.Description className={styles.description}>
            This action cannot be undone. The location must be empty (no
            sub-locations or items) to be deleted.
          </AlertDialog.Description>
          <div className={styles.actions}>
            <AlertDialog.Cancel asChild>
              <button className={styles.cancelButton}>Cancel</button>
            </AlertDialog.Cancel>
            <AlertDialog.Action asChild>
              <button className={styles.deleteButton} onClick={onConfirm}>
                Delete
              </button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
