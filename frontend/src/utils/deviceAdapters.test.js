import { describe, expect, it } from 'vitest'
import { DEVICE_STATUS_LABELS } from '../common/statusLabels.js'
import { adaptDeviceDetail, adaptInterface } from '../utils/deviceAdapters.js'
import { buildAlerts, getSiteStats, normalizeSites } from '../utils/siteData.js'

describe('DEVICE_STATUS_LABELS', () => {
  it('maps 0 to unknown visual treatment', () => {
    expect(DEVICE_STATUS_LABELS[0]).toEqual(['unknown', 'Unknown'])
  })

  it('maps healthy warning and critical distinctly', () => {
    expect(DEVICE_STATUS_LABELS[1]).toEqual(['ok', 'Healthy'])
    expect(DEVICE_STATUS_LABELS[2]).toEqual(['caution', 'Warning'])
    expect(DEVICE_STATUS_LABELS[3]).toEqual(['alert', 'Critical'])
  })
})

describe('adaptDeviceDetail', () => {
  it('preserves Unknown status and dependency evidence', () => {
    const adapted = adaptDeviceDetail(
      {
        hostname: 'access-01',
        status: 0,
        status_reason: 'upstream_unreachable',
        failure_count: 2,
        upstream_device_ids: ['dist-01'],
        unavailable_upstream_device_ids: ['dist-01'],
        root_cause_device_ids: ['core-01'],
        cpu_pct: null,
        memory_pct: null,
        temperature_c: null,
        snmp: {
          sysName: 'access-01',
          sysObjectID: '1.3.6.1.4.1.9.1.9999',
          sysDescr: 'Sanitized',
        },
        power_components: [{ component_id: 'power-1', name: 'PSU 1', status: 'ok', unit: 'state' }],
        history: {
          cpu: [],
          memory: [],
          temperature: [{ ts: '2026-07-16T18:00:00Z', value: 52.5 }],
          uptime: [],
        },
      },
      [],
    )

    expect(adapted.status).toBe(0)
    expect(adapted.status_reason).toBe('upstream_unreachable')
    expect(adapted.root_cause_device_ids).toEqual(['core-01'])
    expect(adapted.cpu_pct).toBeNull()
    expect(adapted.memory_pct).toBeNull()
    expect(adapted.snmp.sysObjectID).toBe('1.3.6.1.4.1.9.1.9999')
    expect(adapted.power_components).toHaveLength(1)
    expect(adapted.history.temperature).toHaveLength(1)
    expect(adapted.admin_status).toBe('down')
  })

  it('does not coerce missing metrics to zero', () => {
    const adapted = adaptDeviceDetail({ hostname: 'core-01', status: 1 }, [])
    expect(adapted.cpu_pct).toBeNull()
    expect(adapted.memory_pct).toBeNull()
    expect(adapted.temperature_c).toBeNull()
  })
})

describe('adaptInterface', () => {
  it('keeps counters when present', () => {
    const adapted = adaptInterface({
      if_index: 1,
      name: 'Gi1/0/1',
      admin_status: 'up',
      oper_status: 'up',
      speed_bps: 1_000_000_000,
      in_octets: 100,
      out_octets: 50,
      in_errors: 1,
      out_errors: 0,
      traffic_history: [{ ts: '2026-07-16T18:00:00Z', value: 100 }],
    })
    expect(adapted.bytes_in).toBe(100)
    expect(adapted.errors_in).toBe(1)
    expect(adapted.traffic_history).toHaveLength(1)
    expect(adapted.speed).toBe('1 Gbps')
  })
})

describe('site aggregates', () => {
  it('does not treat Unknown as Critical in alerts', () => {
    const sites = normalizeSites({
      'district-office': {
        location: 'District Office',
        status: 'caution',
        latest: {
          summary: {
            device_count: 2,
            critical_count: 0,
            unknown_count: 1,
            dependency_impacted_count: 1,
            active_alerts: 0,
          },
        },
      },
      'school-b': {
        location: 'School B',
        status: 'alert',
        latest: {
          summary: {
            device_count: 2,
            critical_count: 1,
            unknown_count: 0,
            dependency_impacted_count: 0,
            active_alerts: 1,
          },
        },
      },
    })

    const alerts = buildAlerts(sites)
    expect(alerts).toHaveLength(1)
    expect(alerts[0].site).toBe('School B')

    const stats = getSiteStats(sites)
    expect(stats.alertCount).toBe(1)
    expect(stats.unknownCount).toBe(1)
    expect(stats.dependencyImpactedCount).toBe(1)
  })
})
