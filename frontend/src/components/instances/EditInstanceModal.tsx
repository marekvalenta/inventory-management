import { useState, useEffect } from 'react'
import type { InstanceDetail, UpdateInstanceRequest } from '../../api/instances'
import styles from './InstanceModals.module.css'

interface EditInstanceModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instance: InstanceDetail
  onSave: (data: UpdateInstanceRequest) => void
}

export function EditInstanceModal({
  open,
  onOpenChange,
  instance,
  onSave,
}: EditInstanceModalProps) {
  const [quantity, setQuantity] = useState(instance.quantity)
  const [fieldValues, setFieldValues] = useState<Record<string, string | null>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (open) {
      setQuantity(instance.quantity)
      const vals: Record<string, string | null> = {}
      for (const fv of instance.field_values) {
        vals[fv.field_id] = fv.value
      }
      setFieldValues(vals)
    }
  }, [open, instance])

  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (quantity < 1) {
      errs.quantity = 'Quantity must be at least 1'
    }
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  function handleSubmit() {
    if (!validate()) return
    const fvs = Object.entries(fieldValues).map(([fieldId, value]) => ({
      field_id: fieldId,
      value,
    }))
    onSave({
      quantity,
      field_values: fvs,
    })
  }

  if (!open) return null

  return (
    <div className={styles.overlay} onClick={() => onOpenChange(false)}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h2 className={styles.title}>Edit {instance.definition_name}</h2>

        <div className={styles.fieldGroup}>
          <label className={styles.label}>Quantity</label>
          <input
            type="number"
            className={styles.input}
            min={1}
            value={quantity}
            onChange={(e) => setQuantity(Math.max(1, parseInt(e.target.value) || 0))}
          />
          {errors.quantity && <div className={styles.errorText}>{errors.quantity}</div>}
        </div>

        {instance.field_values.map((fv) => (
          <div key={fv.field_id} className={styles.fieldGroup}>
            <label className={styles.label}>
              {fv.field_name}
              {fv.field_type === 'enum' ? '' : ` (${fv.field_type})`}
            </label>
            {fv.field_type === 'boolean' ? (
              <label className={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  className={styles.checkbox}
                  checked={fieldValues[fv.field_id] === 'true'}
                  onChange={(e) =>
                    setFieldValues((prev) => ({
                      ...prev,
                      [fv.field_id]: e.target.checked ? 'true' : 'false',
                    }))
                  }
                />
                Yes
              </label>
            ) : fv.field_type === 'enum' && fv.enum_values ? (
              <select
                className={styles.select}
                value={fieldValues[fv.field_id] ?? ''}
                onChange={(e) =>
                  setFieldValues((prev) => ({
                    ...prev,
                    [fv.field_id]: e.target.value || null,
                  }))
                }
              >
                <option value="">-- Select --</option>
                {fv.enum_values.map((ev) => (
                  <option key={ev} value={ev}>{ev}</option>
                ))}
              </select>
            ) : fv.field_type === 'date' ? (
              <input
                type="date"
                className={styles.input}
                value={fieldValues[fv.field_id] ?? ''}
                onChange={(e) =>
                  setFieldValues((prev) => ({
                    ...prev,
                    [fv.field_id]: e.target.value || null,
                  }))
                }
              />
            ) : (
              <input
                type={fv.field_type === 'number' ? 'number' : 'text'}
                className={styles.input}
                value={fieldValues[fv.field_id] ?? ''}
                onChange={(e) =>
                  setFieldValues((prev) => ({
                    ...prev,
                    [fv.field_id]: e.target.value || null,
                  }))
                }
              />
            )}
          </div>
        ))}

        <div className={styles.actions}>
          <button className={styles.cancelButton} onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button className={styles.saveButton} onClick={handleSubmit}>
            Save Changes
          </button>
        </div>
      </div>
    </div>
  )
}
