import type { CreateFieldInput } from '../../api/definitions'
import styles from './FieldEditor.module.css'

const FIELD_TYPES = ['text', 'number', 'boolean', 'date', 'enum'] as const

interface FieldEditorProps {
  field: CreateFieldInput
  index: number
  total: number
  onChange: (updates: Partial<CreateFieldInput>) => void
  onRemove: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  showInheritedControls: boolean
}

export function FieldEditor({
  field,
  index,
  total,
  onChange,
  onRemove,
  onMoveUp,
  onMoveDown,
  showInheritedControls,
}: FieldEditorProps) {
  return (
    <div className={styles.card}>
      <div className={styles.row}>
        <div className={styles.flex1}>
          <label className={styles.label}>Field Name</label>
          <input
            type="text"
            value={field.field_name}
            onChange={(e) => onChange({ field_name: e.target.value })}
            className={styles.input}
            maxLength={100}
          />
        </div>
        <div className={styles.flex1}>
          <label className={styles.label}>Type</label>
          <select
            value={field.field_type}
            onChange={(e) => onChange({ field_type: e.target.value as CreateFieldInput['field_type'] })}
            className={styles.input}
          >
            {FIELD_TYPES.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </div>
      </div>

      {field.field_type === 'enum' && (
        <div className={styles.fieldGroup}>
          <label className={styles.label}>Enum Values (JSON array)</label>
          <input
            type="text"
            value={Array.isArray(field.enum_values) ? JSON.stringify(field.enum_values) : ''}
            onChange={(e) => {
              try {
                const parsed = JSON.parse(e.target.value)
                if (Array.isArray(parsed)) {
                  onChange({ enum_values: parsed })
                }
              } catch {
                onChange({ enum_values: [] })
              }
            }}
            className={styles.input}
            placeholder='["Option A", "Option B"]'
          />
        </div>
      )}

      <div className={styles.checkboxRow}>
        <label className={styles.checkbox}>
          <input
            type="checkbox"
            checked={field.is_required}
            onChange={(e) => onChange({ is_required: e.target.checked })}
          />
          Required
        </label>
        {showInheritedControls && (
          <label className={styles.checkbox}>
            <input
              type="checkbox"
              checked={field.is_child_editable}
              onChange={(e) => onChange({ is_child_editable: e.target.checked })}
            />
            Child Editable
          </label>
        )}
      </div>

      <div className={styles.fieldGroup}>
        <label className={styles.label}>Default Value</label>
        <input
          type="text"
          value={field.default_value ?? ''}
          onChange={(e) => onChange({ default_value: e.target.value || null })}
          className={styles.input}
        />
      </div>

      <div className={styles.actions}>
        <div className={styles.moveGroup}>
          <button
            className={styles.smallButton}
            onClick={onMoveUp}
            disabled={index === 0}
          >
            Up
          </button>
          <button
            className={styles.smallButton}
            onClick={onMoveDown}
            disabled={index === total - 1}
          >
            Down
          </button>
        </div>
        <button className={styles.removeButton} onClick={onRemove}>
          Remove
        </button>
      </div>
    </div>
  )
}
