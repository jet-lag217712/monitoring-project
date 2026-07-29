import { describe, expect, it } from 'vitest'
import { isNoAuthProvider } from './authMode.js'

describe('NoAuth provider', () => {
  it('recognizes only the explicit disabled provider', () => {
    expect(isNoAuthProvider('disabled')).toBe(true)
    expect(isNoAuthProvider('google_session')).toBe(false)
    expect(isNoAuthProvider(undefined)).toBe(false)
  })
})
