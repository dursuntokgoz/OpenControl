import { describe, expect, it } from 'vitest'
import { en } from '../i18n/en'
import { tr } from '../i18n/tr'
import { translate, type Lang } from '../i18n'

describe('i18n catalogs', () => {
  it('have identical key sets', () => {
    const enKeys = Object.keys(en).sort()
    const trKeys = Object.keys(tr).sort()
    expect(trKeys).toEqual(enKeys)
  })

  it('contain no empty strings', () => {
    for (const [key, value] of Object.entries(tr)) {
      expect(value.trim().length, `tr.${key} is empty`).toBeGreaterThan(0)
    }
  })
})

describe('translate', () => {
  const cases: Array<[Lang, keyof typeof en]> = [
    ['en', 'app.title'],
    ['tr', 'app.title'],
    ['tr', 'health.offline'],
  ]
  it.each(cases)('returns catalog value for %s/%s', (lang, key) => {
    expect(translate(lang, key)).not.toBe(key)
  })

  it('falls back to the raw key when missing', () => {
    // Simulate a missing entry by casting an unknown key.
    const bogusKey = 'missing.key' as keyof typeof en
    expect(translate('en', bogusKey)).toBe('missing.key')
  })
})
