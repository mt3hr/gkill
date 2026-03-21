import { describe, test, expect } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'

describe('KFTLStatement', () => {
  test('can be instantiated with text', () => {
    const stmt = new KFTLStatement('テストメモ')
    expect(stmt.get_statement_text()).toBe('テストメモ')
  })

  test('empty text creates valid statement', () => {
    const stmt = new KFTLStatement('')
    expect(stmt.get_statement_text()).toBe('')
  })

  test('lookahead_line_count is defined', () => {
    expect(KFTLStatement.lookahead_line_count).toBeGreaterThan(0)
  })
})
