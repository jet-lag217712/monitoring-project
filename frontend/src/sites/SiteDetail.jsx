import { useEffect, useState } from 'react'
import LoadingSkeleton from '../common/LoadingSkeleton.jsx'
import PageNavStack from '../common/PageNavStack.jsx'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import DevicesTable from '../tables/DevicesTable.jsx'

export default function SiteDetail({ data, siteId, onBack, onDeviceClick, onRenameLocation }) {
  const siteLabel = data?.location ?? siteId ?? 'Site'
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(siteLabel)
  const [saving, setSaving] = useState(false)
  const [renameError, setRenameError] = useState(null)

  useEffect(() => {
    if (!editing) {
      setDraft(data?.location ?? siteId ?? '')
      setRenameError(null)
    }
  }, [data?.location, siteId, editing])

  const beginEdit = () => {
    setDraft(data?.location ?? siteId ?? '')
    setRenameError(null)
    setEditing(true)
  }

  const cancelEdit = () => {
    setEditing(false)
    setDraft(data?.location ?? siteId ?? '')
    setRenameError(null)
  }

  const saveEdit = async () => {
    if (!onRenameLocation || !siteId) return
    setSaving(true)
    setRenameError(null)
    try {
      await onRenameLocation(siteId, draft)
      setEditing(false)
    } catch (err) {
      setRenameError(err?.message ?? 'Failed to rename site')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <PageNavStack
        breadcrumbItems={[
          { label: 'All Sites', onClick: onBack },
          { label: siteLabel },
        ]}
        onBack={onBack}
      />

      {!data ? (
        <LoadingSkeleton />
      ) : (
        <>
          <div className="site-detail-header">
            <div>
              <div className="page-eyebrow">
                <span className="eyebrow-dot" />
                Site Detail
              </div>
              {editing ? (
                <div className="site-rename-row">
                  <input
                    className="site-rename-input"
                    value={draft}
                    onChange={e => setDraft(e.target.value)}
                    onKeyDown={e => {
                      if (e.key === 'Enter') void saveEdit()
                      if (e.key === 'Escape') cancelEdit()
                    }}
                    maxLength={255}
                    disabled={saving}
                    aria-label="Site display name"
                    autoFocus
                  />
                  <button type="button" className="site-rename-btn" onClick={() => void saveEdit()} disabled={saving}>
                    {saving ? 'Saving…' : 'Save'}
                  </button>
                  <button type="button" className="site-rename-btn site-rename-btn-muted" onClick={cancelEdit} disabled={saving}>
                    Cancel
                  </button>
                </div>
              ) : (
                <div className="site-title-row">
                  <h1 className="page-title">{data.location}</h1>
                  {onRenameLocation ? (
                    <button type="button" className="site-rename-edit" onClick={beginEdit}>
                      Rename
                    </button>
                  ) : null}
                </div>
              )}
              <p className="site-id-meta">{siteId}</p>
              {renameError ? <p className="site-rename-error">{renameError}</p> : null}
              <p className="page-sub">
                {data.summary?.total_devices ?? '—'} devices · {data.summary?.online_count ?? '—'} online
                {(data.summary?.warning_count ?? 0) > 0 && ` · ${data.summary.warning_count} warning`}
                {(data.summary?.critical_count ?? 0) > 0 && ` · ${data.summary.critical_count} critical`}
                {(data.summary?.unknown_count ?? 0) > 0 && ` · ${data.summary.unknown_count} unknown`}
                {(data.summary?.dependency_impacted_count ?? 0) > 0 &&
                  ` · ${data.summary.dependency_impacted_count} dependency-impacted`}
              </p>
            </div>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', height: 'fit-content' }}>
              {((data.summary?.critical_count ?? 0) > 0 || data.summary?.active_alerts > 0) && (
                <span className="status-badge alert">
                  <span className="badge-dot" /> Critical Alerts Active
                </span>
              )}
              {(data.summary?.unknown_count ?? 0) > 0 && (
                <span className="status-badge unknown">
                  <span className="badge-dot" /> Dependency Impact
                </span>
              )}
              {data.site_dependency_impacted && (data.root_cause_site_ids?.length ?? 0) > 0 && (
                <span className="status-badge unknown">
                  <span className="badge-dot" /> Upstream: {data.root_cause_site_ids.join(', ')}
                </span>
              )}
            </div>
          </div>

          {data.site_dependency_impacted && (data.unavailable_upstream_site_ids?.length ?? 0) > 0 && (
            <div className="site-dependency-banner" style={{ marginBottom: '1rem', color: 'var(--status-unknown)', fontSize: '0.82rem' }}>
              Site dependency impact: upstream {data.unavailable_upstream_site_ids.join(', ')} unavailable
              {(data.root_cause_site_ids?.length ?? 0) > 0 && ` · root cause ${data.root_cause_site_ids.join(', ')}`}
            </div>
          )}

          <DevicesTable
            devices={data.latest?.devices ?? {}}
            onDeviceClick={onDeviceClick}
            renderStatus={status => <DeviceStatusBadge status={status} />}
          />
        </>
      )}
    </div>
  )
}
