import styles from './InstanceModals.module.css'

interface DeleteInstanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instanceName: string
  childCount: number
  quantity: number
  onConfirm: () => void
}

export function DeleteInstanceDialog({
  open,
  onOpenChange,
  instanceName,
  childCount,
  quantity,
  onConfirm,
}: DeleteInstanceDialogProps) {
  if (!open) return null

  return (
    <div className={styles.overlay} onClick={() => onOpenChange(false)}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h2 className={styles.title}>Delete Instance</h2>

        {childCount > 0 ? (
          <>
            <div className={styles.warningText}>
              This instance contains {childCount} items. You must move them out before deleting.
            </div>
            <div className={styles.actions}>
              <button className={styles.saveButton} onClick={() => onOpenChange(false)}>
                OK
              </button>
            </div>
          </>
        ) : (
          <>
            <div className={styles.confirmationText}>
              Delete this instance of <strong>{instanceName}</strong>? Quantity: {quantity}
            </div>
            <div className={styles.actions}>
              <button className={styles.cancelButton} onClick={() => onOpenChange(false)}>
                Cancel
              </button>
              <button
                className={styles.saveButton}
                onClick={onConfirm}
                style={{ background: 'var(--color-danger)' }}
              >
                Delete
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
