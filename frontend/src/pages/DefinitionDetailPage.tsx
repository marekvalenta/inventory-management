import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeftIcon, Pencil1Icon, TrashIcon, LockClosedIcon, Cross2Icon, PlusIcon } from '@radix-ui/react-icons'
import { fetchDefinition, updateDefinition, deleteDefinition, updateOverrides } from '../api/definitions'
import type { CreateFieldInput, OverrideInput } from '../api/definitions'
import { fetchTags } from '../api/tags'
import type { Tag } from '../api/tags'
import { TagBadge } from '../components/tags/TagBadge'
import { FieldEditor } from '../components/definitions/FieldEditor'
import { useToast } from '../context/ToastContext'
import styles from './DefinitionDetailPage.module.css'

export function DefinitionDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { addToast } = useToast()

  const [activeTab, setActiveTab] = useState<'fields' | 'tags' | 'instances'>('fields')
  const [showEditForm, setShowEditForm] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editUnit, setEditUnit] = useState('')
  const [editIsContainer, setEditIsContainer] = useState(false)
  const [editNameError, setEditNameError] = useState('')

  const [ownFields, setOwnFields] = useState<CreateFieldInput[]>([])
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([])
  const [overrideValues, setOverrideValues] = useState<Record<string, string>>({})

  const { data: definition, isLoading, error } = useQuery({
    queryKey: ['definitions', id],
    queryFn: () => fetchDefinition(id!),
    enabled: !!id,
  })

  const { data: allTags } = useQuery({
    queryKey: ['tags'],
    queryFn: fetchTags,
  })

  const updateMutation = useMutation({
    mutationFn: (data: { name?: string; description?: string | null; unit?: string | null; is_container?: boolean; fields?: CreateFieldInput[]; tag_ids?: string[] }) =>
      updateDefinition(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['definitions'] })
      queryClient.invalidateQueries({ queryKey: ['definitions', id] })
      setShowEditForm(false)
      addToast('Definition updated', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to update definition', 'error')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteDefinition(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['definitions'] })
      addToast('Definition deleted', 'success')
      navigate('/definitions')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to delete definition', 'error')
      setShowDeleteConfirm(false)
    },
  })

  const overrideMutation = useMutation({
    mutationFn: (overrides: OverrideInput[]) => updateOverrides(id!, overrides),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['definitions', id] })
      addToast('Overrides saved', 'success')
    },
    onError: (err: Error) => {
      addToast(err.message || 'Failed to save overrides', 'error')
    },
  })

  if (isLoading) {
    return (
      <div className={styles.container}>
        <Link to="/definitions" className={styles.backLink}>
          <ArrowLeftIcon width={18} height={18} />
          <span>Definitions</span>
        </Link>
        <div className={styles.loading}>Loading definition...</div>
      </div>
    )
  }

  if (error || !definition) {
    const message = error
      ? (error instanceof Error ? error.message : 'Failed to load definition')
      : 'Definition not found'
    return (
      <div className={styles.container}>
        <Link to="/definitions" className={styles.backLink}>
          <ArrowLeftIcon width={18} height={18} />
          <span>Definitions</span>
        </Link>
        <div className={styles.errorState}>
          <p className={styles.errorText}>{message}</p>
        </div>
      </div>
    )
  }

  const ownDefinedFields = definition.fields.filter((f) => !f.inherited_from_def_id)
  const inheritedFields = definition.fields.filter((f) => f.inherited_from_def_id)

  const isChild = !!definition.parent_def_id

  const startEdit = () => {
    setEditName(definition.name)
    setEditDescription(definition.description || '')
    setEditUnit(definition.unit || '')
    setEditIsContainer(definition.is_container)
    setOwnFields(ownDefinedFields.map((f) => ({
      field_name: f.field_name,
      field_type: f.field_type,
      enum_values: f.enum_values || null,
      is_required: f.is_required,
      display_order: f.display_order,
      default_value: f.default_value,
      is_child_editable: f.is_child_editable,
    })))
    setSelectedTagIds(definition.tags.map((t) => t.id))
    setEditNameError('')
    setShowEditForm(true)
  }

  const handleSaveEdit = () => {
    const trimmed = editName.trim()
    if (trimmed.length < 2) {
      setEditNameError('Name must be at least 2 characters')
      return
    }
    if (trimmed.length > 200) {
      setEditNameError('Name must be at most 200 characters')
      return
    }

    const data: Parameters<typeof updateMutation.mutate>[0] = {}

    if (trimmed !== definition.name) data.name = trimmed
    const desc = editDescription.trim() || null
    if (desc !== ((definition.description) || null)) data.description = desc
    const unit = editUnit.trim() || null
    if (unit !== ((definition.unit) || null)) data.unit = unit
    if (editIsContainer !== definition.is_container) data.is_container = editIsContainer

    const currentTagIds = definition.tags.map((t) => t.id)
    if (JSON.stringify(selectedTagIds.sort()) !== JSON.stringify(currentTagIds.sort())) {
      data.tag_ids = selectedTagIds
    }

    data.fields = ownFields

    updateMutation.mutate(data)
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

  const handleSaveOverrides = () => {
    const overrides: OverrideInput[] = []
    for (const field of inheritedFields) {
      if (field.is_child_editable) {
        const key = field.id
        const val = overrideValues[key]
        if (val !== undefined && val !== (field.default_value || '')) {
          overrides.push({
            parent_field_id: field.id,
            default_value: val || null,
          })
        }
      }
    }
    if (overrides.length > 0) {
      overrideMutation.mutate(overrides)
    }
  }

  const handleTagClick = (tagId: string) => {
    const newTags = selectedTagIds.includes(tagId)
      ? selectedTagIds.filter((id) => id !== tagId)
      : [...selectedTagIds, tagId]
    setSelectedTagIds(newTags)

    const data: Parameters<typeof updateMutation.mutate>[0] = { tag_ids: newTags }
    updateMutation.mutate(data)
  }

  const summary = definition.instances_summary

  return (
    <div className={styles.container}>
      <Link to="/definitions" className={styles.backLink}>
        <ArrowLeftIcon width={18} height={18} />
        <span>Definitions</span>
      </Link>

      <div className={styles.headerCard}>
        <div className={styles.headerTop}>
          <div>
            <h2 className={styles.name}>{definition.name}</h2>
            <div className={styles.meta}>
              {definition.unit && <span>{definition.unit}</span>}
              {definition.parent_def_id && definition.parent_def_name && (
                <>
                  {definition.unit && <span className={styles.metaSeparator}>|</span>}
                  <span>
                    Inherits from{' '}
                    <Link to={`/definitions/${definition.parent_def_id}`} className={styles.parentLink}>
                      {definition.parent_def_name}
                    </Link>
                  </span>
                </>
              )}
              {definition.child_definition_count > 0 && (
                <>
                  <span className={styles.metaSeparator}>|</span>
                  <span>{definition.child_definition_count} {definition.child_definition_count === 1 ? 'child' : 'children'}</span>
                </>
              )}
              {definition.is_container && (
                <>
                  <span className={styles.metaSeparator}>|</span>
                  <span>Container</span>
                </>
              )}
            </div>
            {definition.description && (
              <div className={styles.meta}>{definition.description}</div>
            )}
          </div>
        </div>

        {definition.tags.length > 0 && (
          <div className={styles.tagRow}>
            {definition.tags.map((tag) => (
              <TagBadge key={tag.id} tag={tag} />
            ))}
          </div>
        )}

        <div className={styles.actions}>
          <button className={styles.editButton} onClick={startEdit}>
            <Pencil1Icon width={16} height={16} />
            <span>Edit</span>
          </button>
          <button className={styles.deleteButton} onClick={() => setShowDeleteConfirm(true)}>
            <TrashIcon width={16} height={16} />
            <span>Delete</span>
          </button>
        </div>
      </div>

      {showDeleteConfirm && (
        <div className={styles.headerCard}>
          {summary.total_instances > 0 && (
            <p className={styles.dangerText}>
              Cannot delete: this definition has {summary.total_instances}{' '}
              {summary.total_instances === 1 ? 'instance' : 'instances'}. Remove all instances first.
            </p>
          )}
          {definition.child_definition_count > 0 && (
            <p className={styles.dangerText}>
              Cannot delete: {definition.child_definition_count}{' '}
              {definition.child_definition_count === 1 ? 'child definition inherits' : 'child definitions inherit'} from this definition.
            </p>
          )}
          {summary.total_instances === 0 && definition.child_definition_count === 0 && (
            <p className={styles.dangerText}>
              Delete definition &quot;{definition.name}&quot;?
            </p>
          )}
          <div className={styles.deleteActions}>
            <button className={styles.cancelButton} onClick={() => setShowDeleteConfirm(false)}>
              Cancel
            </button>
            {(summary.total_instances === 0 && definition.child_definition_count === 0) && (
              <button
                className={styles.deleteButton}
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            )}
          </div>
        </div>
      )}

      <div className={styles.tabBar}>
        <button
          className={activeTab === 'fields' ? styles.tabActive : styles.tab}
          onClick={() => setActiveTab('fields')}
        >
          Fields
        </button>
        <button
          className={activeTab === 'tags' ? styles.tabActive : styles.tab}
          onClick={() => setActiveTab('tags')}
        >
          Tags
        </button>
        <button
          className={activeTab === 'instances' ? styles.tabActive : styles.tab}
          onClick={() => setActiveTab('instances')}
        >
          Instances
        </button>
      </div>

      {activeTab === 'fields' && !showEditForm && (
        <div className={styles.section}>
          {inheritedFields.length > 0 && (
            <>
              <h3 className={styles.sectionHeader}>Inherited Fields</h3>
              <table className={styles.fieldTable}>
                <thead>
                  <tr>
                    <th>Field Name</th>
                    <th>Type</th>
                    <th>Required</th>
                    <th>Default Value</th>
                    <th>From</th>
                    {isChild && <th>Override</th>}
                  </tr>
                </thead>
                <tbody>
                  {inheritedFields.map((field) => (
                    <tr key={field.id} className={styles.inheritedRow}>
                      <td>
                        {!field.is_child_editable && (
                          <LockClosedIcon width={14} height={14} className={styles.lockIcon} />
                        )}
                        {field.field_name}
                      </td>
                      <td>{field.field_type}</td>
                      <td>{field.is_required ? 'Yes' : 'No'}</td>
                      <td>{field.default_value ?? '-'}</td>
                      <td className={styles.inheritedLabel}>
                        {field.inherited_from_def_id ? 'parent' : ''}
                      </td>
                      {isChild && (
                        <td>
                          {field.is_child_editable ? (
                            <input
                              type="text"
                              className={styles.formInput}
                              value={overrideValues[field.id] ?? field.default_value ?? ''}
                              onChange={(e) =>
                                setOverrideValues((prev) => ({ ...prev, [field.id]: e.target.value }))
                              }
                              style={{ width: '100%', minHeight: 32 }}
                            />
                          ) : (
                            <span className={styles.inheritedLabel}>Sealed</span>
                          )}
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
              {isChild && inheritedFields.some((f) => f.is_child_editable) && (
                <div style={{ marginTop: 'var(--space-md)' }}>
                  <button className={styles.smallButton} onClick={handleSaveOverrides}>
                    Save Overrides
                  </button>
                </div>
              )}
            </>
          )}

          <h3 className={styles.sectionHeader} style={{ marginTop: inheritedFields.length > 0 ? 'var(--space-xl)' : 0 }}>
            Own Fields
          </h3>
          {ownDefinedFields.length === 0 && (
            <div className={styles.emptySmall}>No own fields defined</div>
          )}
          {ownDefinedFields.length > 0 && (
            <table className={styles.fieldTable}>
              <thead>
                <tr>
                  <th>Field Name</th>
                  <th>Type</th>
                  <th>Required</th>
                  <th>Default Value</th>
                  <th>Child Editable</th>
                </tr>
              </thead>
              <tbody>
                {ownDefinedFields.map((field) => (
                  <tr key={field.id}>
                    <td>{field.field_name}</td>
                    <td>{field.field_type}</td>
                    <td>{field.is_required ? 'Yes' : 'No'}</td>
                    <td>{field.default_value ?? '-'}</td>
                    <td>{field.is_child_editable ? 'Yes' : 'No'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {activeTab === 'fields' && showEditForm && (
        <div className={styles.section}>
          <div className={styles.formCard}>
            <div className={styles.formField}>
              <label className={styles.formLabel} htmlFor="edit-name">Name</label>
              <input
                id="edit-name"
                className={styles.formInput}
                type="text"
                value={editName}
                onChange={(e) => { setEditName(e.target.value); setEditNameError('') }}
                maxLength={200}
              />
              {editNameError && <span className={styles.fieldError}>{editNameError}</span>}
            </div>
            <div className={styles.formField}>
              <label className={styles.formLabel} htmlFor="edit-desc">Description</label>
              <textarea
                id="edit-desc"
                className={styles.formTextarea}
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                maxLength={2000}
              />
            </div>
            <div className={styles.formField}>
              <label className={styles.formLabel} htmlFor="edit-unit">Unit</label>
              <input
                id="edit-unit"
                className={styles.formInput}
                type="text"
                value={editUnit}
                onChange={(e) => setEditUnit(e.target.value)}
                maxLength={20}
              />
            </div>
            <label className={styles.formCheckbox}>
              <input
                type="checkbox"
                checked={editIsContainer}
                onChange={(e) => setEditIsContainer(e.target.checked)}
              />
              Is Container
            </label>

            <h3 className={styles.sectionHeader} style={{ marginTop: 'var(--space-xl)' }}>Fields</h3>
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
                showInheritedControls={!isChild}
              />
            ))}
            <button className={styles.smallButton} onClick={addOwnField}>
              <PlusIcon width={14} height={14} />
              Add Field
            </button>

            <div className={styles.formActions} style={{ marginTop: 'var(--space-xl)' }}>
              <button
                className={styles.cancelButton}
                onClick={() => setShowEditForm(false)}
              >
                <Cross2Icon width={16} height={16} />
                <span>Cancel</span>
              </button>
              <button
                className={styles.saveButton}
                onClick={handleSaveEdit}
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? 'Saving...' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'tags' && (
        <div className={styles.section}>
          <h3 className={styles.sectionHeader}>Tags</h3>
          <div className={styles.tagSelect}>
            {allTags?.map((tag: Tag) => {
              const isAssigned = definition.tags.some((t) => t.id === tag.id)
              return (
                <button
                  key={tag.id}
                  className={`${styles.tagChip} ${!isAssigned ? styles.tagChipInactive : ''}`}
                  onClick={() => handleTagClick(tag.id)}
                >
                  <TagBadge tag={tag} size="md" />
                  {isAssigned && (
                    <span className={styles.tagChipRemove}>
                      <Cross2Icon width={12} height={12} />
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>
      )}

      {activeTab === 'instances' && (
        <div className={styles.section}>
          <p className={styles.summaryHeader}>
            {summary.total_instances} {summary.total_instances === 1 ? 'instance' : 'instances'}{' '}
            ({summary.total_quantity} total items)
          </p>

          {summary.by_location.length === 0 && summary.by_parent_instance.length === 0 && (
            <div className={styles.emptySmall}>No instances of this definition yet</div>
          )}

          {summary.by_location.length > 0 && (
            <>
              <h3 className={styles.sectionHeader}>By Location</h3>
              <table className={styles.locationTable}>
                <thead>
                  <tr>
                    <th>Location</th>
                    <th>Instances</th>
                    <th>Total Quantity</th>
                  </tr>
                </thead>
                <tbody>
                  {summary.by_location.map((loc) => (
                    <tr key={loc.location_id}>
                      <td>
                        <Link to={`/locations/${loc.location_id}`} className={styles.locationLink}>
                          {loc.location_name}
                        </Link>
                      </td>
                      <td>{loc.instance_count}</td>
                      <td>{loc.total_quantity}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}

          {summary.by_parent_instance.length > 0 && (
            <>
              <h3 className={styles.sectionHeader}>By Parent Instance</h3>
              <table className={styles.locationTable}>
                <thead>
                  <tr>
                    <th>Parent Instance</th>
                    <th>Location</th>
                    <th>Instances</th>
                    <th>Total Quantity</th>
                  </tr>
                </thead>
                <tbody>
                  {summary.by_parent_instance.map((pi) => (
                    <tr key={pi.parent_instance_id}>
                      <td>
                        <Link to={`/instances/${pi.parent_instance_id}`} className={styles.locationLink}>
                          {pi.parent_instance_name}
                        </Link>
                      </td>
                      <td>
                        <Link to={`/locations/${pi.location_id}`} className={styles.locationLink}>
                          {pi.location_name}
                        </Link>
                      </td>
                      <td>{pi.instance_count}</td>
                      <td>{pi.total_quantity}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
        </div>
      )}
    </div>
  )
}


