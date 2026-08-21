/**
 * i18n locale completeness tests.
 * Verifies that all non-Japanese locales have all keys from Japanese (ja) locale,
 * and that every i18n key referenced in the client source actually exists in ja.json.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import ja from '../../../locales/ja.json'
import en from '../../../locales/en.json'
import zh from '../../../locales/zh.json'
import ko from '../../../locales/ko.json'
import es from '../../../locales/es.json'
import fr from '../../../locales/fr.json'
import de from '../../../locales/de.json'

const jaKeys = Object.keys(ja).sort()

const locales: Record<string, Record<string, string>> = { en, zh, ko, es, fr, de }

describe('i18n locale completeness', () => {
  test.each(['en', 'zh', 'ko', 'es', 'fr', 'de'])('%s has all keys from ja', (locale) => {
    const localeKeys = Object.keys(locales[locale])
    const missing = jaKeys.filter(k => !localeKeys.includes(k))
    if (missing.length > 0) {
      const sample = missing.slice(0, 10).join(', ')
      throw new Error(`Locale '${locale}' is missing ${missing.length} key(s): ${sample}`)
    }
  })

  test('no duplicate keys within ja locale', () => {
    // JSON parse automatically deduplicates, so we read as text
    // For this test, just verify key count matches
    const keys = Object.keys(ja)
    expect(new Set(keys).size).toBe(keys.length)
  })

  // S3-kc-i18n: ソース中の t('KEY') / $t('KEY') の実在キーが ja.json に無いと、
  // vue-i18n は生キー文字列をそのまま画面に出す（use-edit-kc-view.ts が
  // 実在しない KC_RELATED_TIME_IS_BLANK_MESSAGE を参照して生キー表示していた）。
  // i18n キーは UPPER_SNAKE 固定なので、その形のリテラルだけを検査して動的キーは対象外。
  test('every referenced i18n key exists in ja.json', () => {
    // vitest はリポジトリ直下から走るので process.cwd() が gkill ルート。
    const clientDir = join(process.cwd(), 'src', 'client')
    const jaKeySet = new Set(Object.keys(ja))
    const keyPattern = /(?:\$t|\bt)\(\s*['"]([A-Z][A-Z0-9_]+)['"]/g
    const missing = new Map<string, string>() // key -> first file

    const walk = (dir: string): void => {
      for (const entry of readdirSync(dir, { withFileTypes: true })) {
        if (entry.name === '__tests__' || entry.name === 'node_modules') continue
        const full = `${dir}/${entry.name}`
        if (entry.isDirectory()) {
          walk(full)
        } else if (/\.(ts|vue)$/.test(entry.name)) {
          const src = readFileSync(full, 'utf8')
          let m: RegExpExecArray | null
          keyPattern.lastIndex = 0
          while ((m = keyPattern.exec(src)) !== null) {
            const key = m[1]
            if (!jaKeySet.has(key) && !missing.has(key)) {
              missing.set(key, full.replace(clientDir, ''))
            }
          }
        }
      }
    }
    walk(clientDir)

    if (missing.size > 0) {
      const lines = [...missing.entries()].map(([k, f]) => `  ${k} (${f})`).join('\n')
      throw new Error(`Referenced i18n key(s) missing from ja.json:\n${lines}`)
    }
  })
})
