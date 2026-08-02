import { useState, useEffect, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchDefinitions, fetchDefinition } from '../../api/definitions'
import type { CreateInstanceRequest } from '../../api/instances'
import type { DefinitionDetail, DefinitionField } from '../../api/definitions'
import styles from './InstanceModals.module.css'

interface CreateInstanceModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  locationId?: string | null
  parentInstanceId?: string | null
  onSave: (data: CreateInstanceRequest) => void
}

export function CreateInstanceModal({
  open,
  onOpenChange,
  locationId,
  parentInstanceId,
  onSave,
}: CreateInstanceModalProps) {
  const [definitionId, setDefinitionId] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [fieldValues, setFieldValues] = useState<Record<string, string | null>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [searchQuery, setSearchQuery] = useState('')

  const { data: definitions = [] } = useQuery({
    queryKey: ['definitions'],
    queryFn: fetchDefinitions,
    enabled: open,
  })

  const selectedDefId = definitionId || ''
  const { data: definitionDetail } = useQuery({
    queryKey: ['definitions', selectedDefId],
    queryFn: () => fetchDefinition(selectedDefId),
    enabled: open && selectedDefId !== '',
  })

  useEffect(() => {
    if (definitionDetail) {
      const defaults: Record<string, string | null> = {}
      for (const f of definitionDetail.fields) {
        if (f.default_value !== null && f.default_value !== undefined) {
          defaults[f.id] = f.default_value
        }
      }
      setFieldValues(defaults)
    }
  }, [definitionDetail])

  const filteredDefinitions = useMemo(() => {
    if (!searchQuery.trim()) return definitions
    const q = searchQuery.toLowerCase()
    return definitions.filter((d) => d.name.toLowerCase().includes(q))
  }, [definitions, searchQuery])

  const selectedDefinition = definitions.find((d) => d.id === definitionId)

  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!definitionId) {
      errs.definition = 'Please select a definition'
    }
    if (quantity < 1) {
      errs.quantity = 'Quantity must be at least 1'
    }
    if (definitionDetail) {
      for (const f of definitionDetail.fields) {
        const val = fieldValues[f.id]
        if (f.is_required && (val === undefined || val === null || val === '')) {
          errs[f.id] = `${f.field_name} is required`
        }
      }
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
      definition_id: definitionId,
      quantity,
      location_id: locationId || undefined,
      parent_instance_id: parentInstanceId || undefined,
      field_values: fvs,
    })
  }

  if (!open) return null

  return (
    <div className={styles.overlay} onClick={() => onOpenChange(false)}>
      <div className={styles.dialog} onClick={(e) => e.stopPropagation()}>
        <h2 className={styles.title}>Add Item</h2>

        {locationId && (
          <div className={styles.contextInfo}>
            Adding to this location
          </div>
        )}
        {parentInstanceId && (
          <div className={styles.contextInfo}>
            Adding inside this container
          </div>
        )}

        <div className={styles.fieldGroup}>
          <label className={styles.label}>
            Definition<span className={styles.required}>*</span>
          </label>
          {!definitionId ? (
            <>
              <input
                type="text"
                className={styles.input}
                placeholder="Search definitions..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <div className={styles.searchResults}>
                {filteredDefinitions.map((def) => (
                  <div
                    key={def.id}
                    className={styles.searchItem}
                    onClick={() => {
                      setDefinitionId(def.id)
                      setSearchQuery('')
                    }}
                  >
                    {def.name} {def.unit ? `(${def.unit})` : ''}
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className={styles.select} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>{selectedDefinition?.name} {selectedDefinition?.unit ? `(${selectedDefinition.unit})` : ''}</span>
              <button
                className={styles.cancelButton}
                onClick={() => setDefinitionId('')}
                style={{ padding: '2px 8px', minHeight: 'auto', fontSize: 13 }}
              >
                Change
              </button>
            </div>
          )}
          {errors.definition && <div className={styles.errorText}>{errors.definition}</div>}
        </div>

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

        {definitionDetail?.fields.map((field) => (
          <FieldInput
            key={field.id}
            field={field}
            value={fieldValues[field.id] ?? null}
            onChange={(val) => setFieldValues((prev) => ({ ...prev, [field.id]: val }))}
            error={errors[field.id]}
          />
        ))}

        <div className={styles.actions}>
          <button className={styles.cancelButton} onClick={() => onOpenChange(false)}>
            Cancel
          </button>
          <button className={styles.saveButton} onClick={handleSubmit}>
            Add Item
          </button>
        </div>
      </div>
    </div>
  )
}

function FieldInput({
  field,
  value,
  onChange,
  error,
}: {
  field: DefinitionField
  value: string | null
  onChange: (value: string | null) => void
  error?: string
}) {
  return (
    <div className={styles.fieldGroup}>
      <label className={styles.label}>
        {field.field_name}
        {field.is_required && <span className={styles.required}>*</span>}
      </label>
      {field.field_type === 'boolean' ? (
        <label className={styles.checkboxLabel}>
          <input
            type="checkbox"
            className={styles.checkbox}
            checked={value === 'true'}
            onChange={(e) => onChange(e.target.checked ? 'true' : 'false')}
          />
          Yes
        </label>
      ) : field.field_type === 'enum' && field.enum_values ? (
        <select
          className={styles.select}
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value || null)}
        >
          <option value="">-- Select --</option>
          {field.enum_values.map((ev) => (
            <option key={ev} value={ev}>{ev}</option>
          ))}
        </select>
      ) : field.field_type === 'date' ? (
        <input
          type="date"
          className={styles.input}
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value || null)}
        />
      ) : (
        <input
          type={field.field_type === 'number' ? 'number' : 'text'}
          className={styles.input}
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value || null)}
        />
      )}
      {error && <div className={styles.errorText}>{error}</div>}
    </div>
  )
}
