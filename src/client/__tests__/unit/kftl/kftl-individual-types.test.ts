import { describe, test, expect, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { KFTLKmemoStatementLine } from '@/classes/kftl/kftl_kmemo/kftl-kmemo-statement-line'
import { KFTLSplitStatementLine } from '@/classes/kftl/kftl_split/kftl-split-statement-line'
import { KFTLSplitAndNextSecondStatementLine } from '@/classes/kftl/kftl_split/kftl-split-and-next-second-statement-line'
import { KFTLPrototypeRequest } from '@/classes/kftl/kftl_prototype/kftl-prototype-request'
import { KFTLTagStatementLine } from '@/classes/kftl/kftl_tag/kftl-tag-statement-line'
import { KFTLRelatedTimeStatementLine } from '@/classes/kftl/kftl_related_time/kftl-related-time-statement-line'
import { KFTLStartTextStatementLine } from '@/classes/kftl/kftl_text/kftl-start-text-statement-line'
import { KFTLStartTimeIsStatementLine } from '@/classes/kftl/kftl_timeis/kftl-start-time-is-statement-line'
import { KFTLStartKCStatementLine } from '@/classes/kftl/kftl_kc/kftl-start-kc-statement-line'
import { KFTLStartMiStatementLine } from '@/classes/kftl/kftl_mi/kftl-start-mi-statement-line'
import { KFTLStartLantanaStatementLine } from '@/classes/kftl/kftl_lantana/kftl-start-lantana-statement-line'
import { KFTLStartNlogStatementLine } from '@/classes/kftl/kftl_nlog/kftl-start-nlog-statement-line'
import { KFTLStartURLogStatementLine } from '@/classes/kftl/kftl_urlog/kftl-start-ur-log-statement-line'

/**
 * Supplementary individual KFTL type tests.
 *
 * The comprehensive type detection (17 types, 28 tests) and request generation
 * (kmemo, tag, split, related-time, 15 tests) are covered in:
 * - kftl-type-detection.test.ts
 * - kftl-request-generation.test.ts
 *
 * This file tests additional behaviors not covered there:
 * - Split vs SplitAndNextSecond distinction (mutual exclusivity)
 * - KFTLPrototypeRequest.is_prototype_request static method
 * - Prefix uniqueness across all type detectors
 * - Kmemo catch-all with various special characters
 */
describe('KFTL Individual Type Supplementary Tests', () => {

  // ─── Split vs SplitAndNextSecond distinction ─────────────────────────────────

  describe('Split vs SplitAndNextSecond mutual exclusivity', () => {
    test('Split does not match SplitAndNextSecond prefix', () => {
      // Split is "、" (single), SplitAndNextSecond is "、、" (double)
      expect(KFTLSplitStatementLine.is_this_type('、、')).toBe(false)
    })

    test('SplitAndNextSecond does not match Split prefix', () => {
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type('、')).toBe(false)
    })

    test('Split matches only the exact single separator', () => {
      expect(KFTLSplitStatementLine.is_this_type('、')).toBe(true)
      expect(KFTLSplitStatementLine.is_this_type('、、')).toBe(false)
      expect(KFTLSplitStatementLine.is_this_type('、test')).toBe(false)
    })

    test('SplitAndNextSecond matches only the exact double separator', () => {
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type('、、')).toBe(true)
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type('、')).toBe(false)
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type('、、、')).toBe(false)
    })
  })

  // ─── KFTLPrototypeRequest ────────────────────────────────────────────────────

  describe('KFTLPrototypeRequest.is_prototype_request', () => {
    test('is_prototype_request is a static method', () => {
      expect(typeof KFTLPrototypeRequest.is_prototype_request).toBe('function')
    })
  })

  // ─── Kmemo catch-all behavior with edge cases ───────────────────────────────

  describe('Kmemo catch-all with special inputs', () => {
    test('returns true for strings starting with tag prefix', () => {
      // Kmemo is the catch-all — it always returns true regardless of content
      expect(KFTLKmemoStatementLine.is_this_type('。タグ')).toBe(true)
    })

    test('returns true for strings matching split prefix', () => {
      expect(KFTLKmemoStatementLine.is_this_type('、')).toBe(true)
    })

    test('returns true for strings matching type prefixes', () => {
      expect(KFTLKmemoStatementLine.is_this_type('ーー')).toBe(true)
      expect(KFTLKmemoStatementLine.is_this_type('ーち')).toBe(true)
      expect(KFTLKmemoStatementLine.is_this_type('ーか')).toBe(true)
    })

    test('returns true for numeric strings', () => {
      expect(KFTLKmemoStatementLine.is_this_type('12345')).toBe(true)
    })

    test('returns true for multi-line strings', () => {
      expect(KFTLKmemoStatementLine.is_this_type('line1\nline2')).toBe(true)
    })
  })

  // ─── Prefix uniqueness ──────────────────────────────────────────────────────

  describe('type prefix uniqueness (no two exact-match types share a prefix)', () => {
    // Collect all exact-match prefixes from i18n (using the actual key names from ja.json)
    const exactPrefixes: Array<{ name: string; prefix: string }> = [
      { name: 'Split', prefix: i18n.global.t('KFTL_SPLIT_PREFIX') },
      { name: 'SplitAndNextSecond', prefix: i18n.global.t('KFTL_SPLIT_APPEND_TIME_PREFIX') },
      { name: 'Text', prefix: i18n.global.t('KFTL_TEXT_SPLITTER_TITLE') },
      { name: 'TimeIs', prefix: i18n.global.t('KFTL_TIMEIS_SPLITTER_TITLE') },
      { name: 'KC', prefix: i18n.global.t('KFTL_KC_SPLITTER_TITLE') },
      { name: 'Mi', prefix: i18n.global.t('KFTL_MI_SPLITTER_TITLE') },
      { name: 'Lantana', prefix: i18n.global.t('KFTL_LANTANA_SPLITTER_TITLE') },
      { name: 'Nlog', prefix: i18n.global.t('KFTL_NLOG_SPLITTER_TITLE') },
      { name: 'URLog', prefix: i18n.global.t('KFTL_URLOG_SPLITTER_TITLE') },
    ]

    test('all exact-match prefixes are distinct', () => {
      const values = exactPrefixes.map((p) => p.prefix)
      const uniqueValues = new Set(values)
      expect(uniqueValues.size).toBe(values.length)
    })

    test('no exact-match prefix is empty', () => {
      for (const entry of exactPrefixes) {
        expect(entry.prefix.length).toBeGreaterThan(0)
      }
    })
  })

  // ─── startsWith vs exact match behavior ──────────────────────────────────────

  describe('startsWith types accept extended content, exact types do not', () => {
    test('Tag (startsWith) accepts content after prefix', () => {
      expect(KFTLTagStatementLine.is_this_type('。追加テキスト')).toBe(true)
    })

    test('RelatedTime (startsWith) accepts content after prefix', () => {
      expect(KFTLRelatedTimeStatementLine.is_this_type('？2025-06-15 12:00:00')).toBe(true)
    })

    test('Text (exact) rejects content after prefix', () => {
      expect(KFTLStartTextStatementLine.is_this_type('ーー追加')).toBe(false)
    })

    test('TimeIs (exact) rejects content after prefix', () => {
      expect(KFTLStartTimeIsStatementLine.is_this_type('ーちextra')).toBe(false)
    })

    test('KC (exact) rejects content after prefix', () => {
      expect(KFTLStartKCStatementLine.is_this_type('ーかextra')).toBe(false)
    })

    test('Mi (exact) rejects content after prefix', () => {
      expect(KFTLStartMiStatementLine.is_this_type('ーみextra')).toBe(false)
    })

    test('Lantana (exact) rejects content after prefix', () => {
      expect(KFTLStartLantanaStatementLine.is_this_type('ーらextra')).toBe(false)
    })

    test('Nlog (exact) rejects content after prefix', () => {
      expect(KFTLStartNlogStatementLine.is_this_type('ーんextra')).toBe(false)
    })

    test('URLog (exact) rejects content after prefix', () => {
      expect(KFTLStartURLogStatementLine.is_this_type('ーうextra')).toBe(false)
    })
  })
})
