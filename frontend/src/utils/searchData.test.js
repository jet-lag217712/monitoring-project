import { describe, expect, it } from 'vitest'
import { filterDevicesFromSiteDetails, mergeDeviceSearchHits } from './searchData.js'

describe('searchData', () => {
  it('matches hostname and IP from site details', () => {
    const hits = filterDevicesFromSiteDetails(
      {
        'school-a': {
          location: 'School A',
          latest: {
            devices: {
              '10.20.0.1': { hostname: 'sch-a-core-01', role: 'Core Switch', status: 1 },
            },
          },
        },
      },
      '10.20',
    )
    expect(hits).toHaveLength(1)
    expect(hits[0].hostname).toBe('sch-a-core-01')
    expect(hits[0].map_key).toBe('10.20.0.1')
  })

  it('dedupes merged device hits', () => {
    const merged = mergeDeviceSearchHits(
      [{ site_id: 'a', map_key: '10.0.0.1', hostname: 'core' }],
      [{ site_id: 'a', map_key: '10.0.0.1', hostname: 'core' }, { site_id: 'b', map_key: '10.0.0.2' }],
    )
    expect(merged).toHaveLength(2)
  })
})
