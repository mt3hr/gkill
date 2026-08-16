import { describe, test, expect } from 'vitest'
import '../../helpers/setup-i18n'

// Import the statement line classes to test their is_this_type static methods
import { KFTLTagStatementLine } from '@/classes/kftl/kftl_tag/kftl-tag-statement-line'
import { KFTLSplitStatementLine } from '@/classes/kftl/kftl_split/kftl-split-statement-line'
import { KFTLSplitAndNextSecondStatementLine } from '@/classes/kftl/kftl_split/kftl-split-and-next-second-statement-line'
import { KFTLRelatedTimeStatementLine } from '@/classes/kftl/kftl_related_time/kftl-related-time-statement-line'
import { KFTLStartTextStatementLine } from '@/classes/kftl/kftl_text/kftl-start-text-statement-line'
import { KFTLStartTimeIsStatementLine } from '@/classes/kftl/kftl_timeis/kftl-start-time-is-statement-line'
import { KFTLStartTimeIsStartStatementLine } from '@/classes/kftl/kftl_timeis/kftl_timeis_start/kftl-start-time-is-start-statement-line'
import { KFTLStartTimeIsEndStatementLine } from '@/classes/kftl/kftl_timeis/kftl_timeis_end/kftl-start-time-is-end-statement-line'
import { KFTLStartTimeIsEndIfExistStatementLine } from '@/classes/kftl/kftl_timeis/kftl_timeis_end/kftl_timeis_end_exist/kftl-start-time-is-end-if-exist-statement-line'
import { KFTLStartTimeIsEndByTagStatementLine } from '@/classes/kftl/kftl_timeis/kftl_timeis_end/kftl_timeis_end_tag/kftl-start-time-is-end-by-tag-statement-line'
import { KFTLStartTimeIsEndByTagIfExistStatementLine } from '@/classes/kftl/kftl_timeis/kftl_timeis_end/kftl_timeis_end_tag_exist/kftl-start-time-is-end-by-tag-if-exist-statement-line'
import { KFTLStartKCStatementLine } from '@/classes/kftl/kftl_kc/kftl-start-kc-statement-line'
import { KFTLStartMiStatementLine } from '@/classes/kftl/kftl_mi/kftl-start-mi-statement-line'
import { KFTLStartLantanaStatementLine } from '@/classes/kftl/kftl_lantana/kftl-start-lantana-statement-line'
import { KFTLStartNlogStatementLine } from '@/classes/kftl/kftl_nlog/kftl-start-nlog-statement-line'
import { KFTLStartURLogStatementLine } from '@/classes/kftl/kftl_urlog/kftl-start-ur-log-statement-line'
import { KFTLKmemoStatementLine } from '@/classes/kftl/kftl_kmemo/kftl-kmemo-statement-line'
import { KFTLStartMiReKyouStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-start-mi-re-kyou-statement-line'
import { KFTLEndMiReKyouStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-end-mi-re-kyou-statement-line'

describe('KFTL Statement Line Type Detection', () => {
  // All is_this_type methods use either startsWith or == against i18n translated prefixes.
  // Tag and RelatedTime use startsWith, all others use == (exact match).

  describe('Tag (startsWith "。")', () => {
    test('matches tag with content', () => {
      expect(KFTLTagStatementLine.is_this_type('。日記')).toBe(true)
    })
    test('matches bare prefix', () => {
      expect(KFTLTagStatementLine.is_this_type('。')).toBe(true)
    })
    test('rejects plain text', () => {
      expect(KFTLTagStatementLine.is_this_type('普通のテキスト')).toBe(false)
    })
  })

  describe('Split (exact "、")', () => {
    test('matches exact', () => {
      expect(KFTLSplitStatementLine.is_this_type('、')).toBe(true)
    })
    test('rejects plain text', () => {
      expect(KFTLSplitStatementLine.is_this_type('テスト')).toBe(false)
    })
  })

  describe('SplitAndNextSecond (exact "、、")', () => {
    test('matches exact', () => {
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type('、、')).toBe(true)
    })
  })

  describe('RelatedTime (startsWith "？")', () => {
    test('matches with date', () => {
      expect(KFTLRelatedTimeStatementLine.is_this_type('？2025-01-15')).toBe(true)
    })
    test('matches bare prefix', () => {
      expect(KFTLRelatedTimeStatementLine.is_this_type('？')).toBe(true)
    })
    test('rejects plain text', () => {
      expect(KFTLRelatedTimeStatementLine.is_this_type('テスト')).toBe(false)
    })
  })

  describe('Text (exact "ーー")', () => {
    test('matches exact', () => {
      expect(KFTLStartTextStatementLine.is_this_type('ーー')).toBe(true)
    })
  })

  describe('TimeIs (exact "ーち")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsStatementLine.is_this_type('ーち')).toBe(true)
    })
  })

  describe('TimeIsStart (exact "ーた")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsStartStatementLine.is_this_type('ーた')).toBe(true)
    })
  })

  describe('TimeIsEnd (exact "ーえ")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsEndStatementLine.is_this_type('ーえ')).toBe(true)
    })
  })

  describe('TimeIsEndIfExist (exact "ーいえ")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsEndIfExistStatementLine.is_this_type('ーいえ')).toBe(true)
    })
  })

  describe('TimeIsEndByTag (exact "ーたえ")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsEndByTagStatementLine.is_this_type('ーたえ')).toBe(true)
    })
  })

  describe('TimeIsEndByTagIfExist (exact "ーいたえ")', () => {
    test('matches exact', () => {
      expect(KFTLStartTimeIsEndByTagIfExistStatementLine.is_this_type('ーいたえ')).toBe(true)
    })
  })

  describe('KC (exact "ーか")', () => {
    test('matches exact', () => {
      expect(KFTLStartKCStatementLine.is_this_type('ーか')).toBe(true)
    })
  })

  describe('Mi (exact "ーみ")', () => {
    test('matches exact', () => {
      expect(KFTLStartMiStatementLine.is_this_type('ーみ')).toBe(true)
    })
  })

  describe('MiReKyou (exact "～～")', () => {
    test('matches exact', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('～～')).toBe(true)
    })
    test('rejects with extra text', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('～～タスク')).toBe(false)
    })
    // 「～」はWindowsのIMEがU+FF5E、macOS/iOSのIMEがU+301Cを出す。
    // 正規化を落とすとiOSからだけ記法が効かなくなる
    test('matches wave dash (U+301C) written by the macOS/iOS IME', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('〜〜')).toBe(true)
      expect(KFTLEndMiReKyouStatementLine.is_this_type('〜〜')).toBe(true)
    })
    test('matches mixed wave dash and fullwidth tilde', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('〜～')).toBe(true)
    })
    test('end line matches the same splitter', () => {
      expect(KFTLEndMiReKyouStatementLine.is_this_type('～～')).toBe(true)
    })
  })

  describe('Lantana (exact "ーら")', () => {
    test('matches exact', () => {
      expect(KFTLStartLantanaStatementLine.is_this_type('ーら')).toBe(true)
    })
  })

  describe('Nlog (exact "ーん")', () => {
    test('matches exact', () => {
      expect(KFTLStartNlogStatementLine.is_this_type('ーん')).toBe(true)
    })
    test('rejects with extra text', () => {
      expect(KFTLStartNlogStatementLine.is_this_type('ーん支出')).toBe(false)
    })
  })

  describe('URLog (exact "ーう")', () => {
    test('matches exact', () => {
      expect(KFTLStartURLogStatementLine.is_this_type('ーう')).toBe(true)
    })
    test('rejects with extra text', () => {
      expect(KFTLStartURLogStatementLine.is_this_type('ーうURL')).toBe(false)
    })
  })

  describe('Kmemo (catch-all)', () => {
    test('always returns true', () => {
      expect(KFTLKmemoStatementLine.is_this_type('普通のメモ')).toBe(true)
    })
    test('empty string returns true', () => {
      expect(KFTLKmemoStatementLine.is_this_type('')).toBe(true)
    })
  })

  describe('Cross-type rejection', () => {
    test('plain text is not Split', () => {
      expect(KFTLSplitStatementLine.is_this_type('メモ')).toBe(false)
    })
    test('KC splitter is not Mi', () => {
      expect(KFTLStartMiStatementLine.is_this_type('ーか')).toBe(false)
    })
    test('Mi splitter is not KC', () => {
      expect(KFTLStartKCStatementLine.is_this_type('ーみ')).toBe(false)
    })
    // リポストタスクの「～～」とテキストの「ーー」は形が似ているので取り違えを見張る
    test('MiReKyou splitter is not Text', () => {
      expect(KFTLStartTextStatementLine.is_this_type('～～')).toBe(false)
    })
    test('Text splitter is not MiReKyou', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('ーー')).toBe(false)
    })
    test('Mi splitter is not MiReKyou', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('ーみ')).toBe(false)
    })
  })

  // MCP(Go側パーサー)と同じASCIIプレフィックスを受け付けること
  describe('ASCII prefixes', () => {
    test('Tag matches "#" with content', () => {
      expect(KFTLTagStatementLine.is_this_type('#diary')).toBe(true)
    })
    test('RelatedTime matches "?" with date', () => {
      expect(KFTLRelatedTimeStatementLine.is_this_type('?2025-01-15')).toBe(true)
    })
    test('Text matches exact "--"', () => {
      expect(KFTLStartTextStatementLine.is_this_type('--')).toBe(true)
    })
    test('MiReKyou matches exact "~~"', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('~~')).toBe(true)
      expect(KFTLEndMiReKyouStatementLine.is_this_type('~~')).toBe(true)
    })
    test('Split matches exact ","', () => {
      expect(KFTLSplitStatementLine.is_this_type(',')).toBe(true)
    })
    test('SplitAndNextSecond matches exact ",,"', () => {
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type(',,')).toBe(true)
    })
    test('KC matches exact "/num"', () => {
      expect(KFTLStartKCStatementLine.is_this_type('/num')).toBe(true)
    })
    test('Mi matches exact "/mi"', () => {
      expect(KFTLStartMiStatementLine.is_this_type('/mi')).toBe(true)
    })
    test('Lantana matches exact "/mood"', () => {
      expect(KFTLStartLantanaStatementLine.is_this_type('/mood')).toBe(true)
    })
    test('Nlog matches exact "/expense"', () => {
      expect(KFTLStartNlogStatementLine.is_this_type('/expense')).toBe(true)
    })
    test('URLog matches exact "/url"', () => {
      expect(KFTLStartURLogStatementLine.is_this_type('/url')).toBe(true)
    })
    test('TimeIs matches exact "/timeis"', () => {
      expect(KFTLStartTimeIsStatementLine.is_this_type('/timeis')).toBe(true)
    })
    test('TimeIsStart matches exact "/start"', () => {
      expect(KFTLStartTimeIsStartStatementLine.is_this_type('/start')).toBe(true)
    })
    test('TimeIsEnd matches exact "/end"', () => {
      expect(KFTLStartTimeIsEndStatementLine.is_this_type('/end')).toBe(true)
    })
    test('TimeIsEndIfExist matches exact "/end?"', () => {
      expect(KFTLStartTimeIsEndIfExistStatementLine.is_this_type('/end?')).toBe(true)
    })
    test('TimeIsEndByTag matches exact "/endt"', () => {
      expect(KFTLStartTimeIsEndByTagStatementLine.is_this_type('/endt')).toBe(true)
    })
    test('TimeIsEndByTagIfExist matches exact "/endt?"', () => {
      expect(KFTLStartTimeIsEndByTagIfExistStatementLine.is_this_type('/endt?')).toBe(true)
    })
  })

  describe('ASCII cross-type rejection', () => {
    test('"," is not SplitAndNextSecond', () => {
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type(',')).toBe(false)
    })
    test('",," is not Split', () => {
      expect(KFTLSplitStatementLine.is_this_type(',,')).toBe(false)
    })
    test('",,," matches neither Split nor SplitAndNextSecond', () => {
      expect(KFTLSplitStatementLine.is_this_type(',,,')).toBe(false)
      expect(KFTLSplitAndNextSecondStatementLine.is_this_type(',,,')).toBe(false)
    })
    test('"/end" is not TimeIsEndIfExist nor TimeIsEndByTag', () => {
      expect(KFTLStartTimeIsEndIfExistStatementLine.is_this_type('/end')).toBe(false)
      expect(KFTLStartTimeIsEndByTagStatementLine.is_this_type('/end')).toBe(false)
    })
    test('"/endt?" is not TimeIsEndByTag', () => {
      expect(KFTLStartTimeIsEndByTagStatementLine.is_this_type('/endt?')).toBe(false)
    })
    test('unknown command "/endx" matches no TimeIs type', () => {
      expect(KFTLStartTimeIsEndStatementLine.is_this_type('/endx')).toBe(false)
      expect(KFTLStartTimeIsEndIfExistStatementLine.is_this_type('/endx')).toBe(false)
      expect(KFTLStartTimeIsEndByTagStatementLine.is_this_type('/endx')).toBe(false)
      expect(KFTLStartTimeIsEndByTagIfExistStatementLine.is_this_type('/endx')).toBe(false)
    })
    test('"/mi" with trailing text is rejected', () => {
      expect(KFTLStartMiStatementLine.is_this_type('/mi task')).toBe(false)
    })
    test('"~~" is not Text and "--" is not MiReKyou', () => {
      expect(KFTLStartTextStatementLine.is_this_type('~~')).toBe(false)
      expect(KFTLStartMiReKyouStatementLine.is_this_type('--')).toBe(false)
    })
    test('"~~" with trailing text is rejected', () => {
      expect(KFTLStartMiReKyouStatementLine.is_this_type('~~ task')).toBe(false)
    })
  })
})
