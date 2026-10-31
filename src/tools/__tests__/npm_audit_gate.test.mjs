// Nightly の npm audit の判定（npm_audit_gate.mjs）のテスト。
//
// 例外リストは「修正版の無い開発用依存の advisory だけを、条件付きで通す」ためのもの。条件が1つ緩むと
// 例外もエラーも出ないまま本番依存の脆弱性や修正版の出た advisory が Nightly を素通りし、リリースゲートまで
// 緑になる（ゲートは Nightly の run 単位の success しか見ない）。どの条件で落ちるかをここで固定する。
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  ALLOWLIST_PATH,
  advisoryId,
  evaluateAudit,
  isAtLeast,
  npmInvocation,
  parseAllowlist,
  parseAuditReport,
} from '../npm_audit_gate.mjs'

// npm audit --json（auditReportVersion 2）の形。braces の advisory 1件が micromatch → fast-glob と連鎖する
function report(vulnerabilities) {
  return JSON.stringify({ auditReportVersion: 2, vulnerabilities, metadata: {} })
}

const BRACES_VIA = {
  source: 1000001,
  name: 'braces',
  dependency: 'braces',
  title: 'braces stack exhaustion',
  url: 'https://github.com/advisories/GHSA-aaaa-bbbb-cccc',
  severity: 'high',
  range: '<=3.0.3',
}

const CHAINED = {
  braces: { name: 'braces', severity: 'high', isDirect: false, via: [BRACES_VIA], range: '*' },
  micromatch: { name: 'micromatch', severity: 'high', isDirect: false, via: ['braces'], range: '>=0.2.0' },
  'fast-glob': { name: 'fast-glob', severity: 'high', isDirect: false, via: ['micromatch'], range: '*' },
}

const BRACES_EXCEPTION = {
  id: 'GHSA-aaaa-bbbb-cccc',
  package: 'braces',
  range: '<=3.0.3',
  reason: '修正版が無い開発用依存',
  review_by: '2024-03-01',
}

const NOW = new Date('2024-01-01T00:00:00Z')

describe('npm_audit_gate: npm audit の出力を読む', () => {
  test('via のオブジェクトだけを advisory として拾い、連鎖（文字列の via）は数えない', () => {
    const advisories = parseAuditReport(report(CHAINED))
    expect(advisories).toEqual([
      {
        id: 'GHSA-aaaa-bbbb-cccc',
        package: 'braces',
        severity: 'high',
        range: '<=3.0.3',
        title: 'braces stack exhaustion',
        url: 'https://github.com/advisories/GHSA-aaaa-bbbb-cccc',
      },
    ])
  })

  test('同じ advisory が複数のエントリに載っていても1件にまとめ、ID 順に並べる', () => {
    const other = { ...BRACES_VIA, name: 'aaa', url: 'https://github.com/advisories/GHSA-0000-0000-0000', range: '<1.0.0' }
    const advisories = parseAuditReport(report({
      braces: { name: 'braces', severity: 'high', via: [BRACES_VIA, BRACES_VIA] },
      other: { name: 'braces', severity: 'high', via: [BRACES_VIA] },
      aaa: { name: 'aaa', severity: 'high', via: [other] },
    }))
    expect(advisories.map((a) => a.id)).toEqual(['GHSA-0000-0000-0000', 'GHSA-aaaa-bbbb-cccc'])
  })

  test('脆弱性が無ければ空', () => {
    expect(parseAuditReport(report({}))).toEqual([])
  })

  test('JSON でない・エラーの JSON・想定外の形は投げる（fail-closed）', () => {
    expect(() => parseAuditReport('')).toThrow('JSON として読めない')
    expect(() => parseAuditReport(JSON.stringify({ error: { code: 'ENOAUDIT', summary: 'audit endpoint returned an error' } })))
      .toThrow('audit endpoint returned an error')
    expect(() => parseAuditReport(JSON.stringify({ auditReportVersion: 1, advisories: {} }))).toThrow('形が想定と違う')
    expect(() => parseAuditReport(JSON.stringify({ auditReportVersion: 2 }))).toThrow('形が想定と違う')
  })

  test('high 以上のエントリがあるのに advisory を拾えなければ投げる（via の形が変わっても黙って 0 件にしない）', () => {
    const onlyChains = { micromatch: CHAINED.micromatch, 'fast-glob': CHAINED['fast-glob'] }
    expect(() => parseAuditReport(report(onlyChains))).toThrow('advisory を拾えない')
    const asString = { braces: { ...CHAINED.braces, via: ['GHSA-aaaa-bbbb-cccc'] } }
    expect(() => parseAuditReport(report(asString))).toThrow('advisory を拾えない')
  })

  test('advisory の ID は URL の GHSA、無ければ npm の source 番号', () => {
    expect(advisoryId({ url: 'https://github.com/advisories/GHSA-vfj7-8cjw-p6xm', source: 1 })).toBe('GHSA-vfj7-8cjw-p6xm')
    expect(advisoryId({ url: '', source: 1000002 })).toBe('npm:1000002')
  })

  test('high 以上を落とす。知らない severity も落とす側に倒す', () => {
    expect(['info', 'low', 'moderate'].map((s) => isAtLeast(s))).toEqual([false, false, false])
    expect(['high', 'critical', 'unknown'].map((s) => isAtLeast(s))).toEqual([true, true, true])
  })
})

describe('npm_audit_gate: 例外リストを読む', () => {
  const text = (exceptions) => JSON.stringify({ exceptions })

  test('5項目そろった例外を読む（余計な項目は落とす）', () => {
    expect(parseAllowlist(text([{ ...BRACES_EXCEPTION, extra: 'x' }]))).toEqual([BRACES_EXCEPTION])
  })

  test('項目の欠け・空文字は投げる', () => {
    for (const key of ['id', 'package', 'range', 'reason', 'review_by']) {
      const ex = { ...BRACES_EXCEPTION }
      delete ex[key]
      expect(() => parseAllowlist(text([ex]))).toThrow(key)
      expect(() => parseAllowlist(text([{ ...BRACES_EXCEPTION, [key]: ' ' }]))).toThrow(key)
    }
  })

  test('review_by は実在する YYYY-MM-DD だけ', () => {
    for (const bad of ['2024/03/01', '2024-3-1', '2024-02-30', 'someday']) {
      expect(() => parseAllowlist(text([{ ...BRACES_EXCEPTION, review_by: bad }]))).toThrow('review_by')
    }
  })

  test('同じ ID の重複・exceptions の欠け・JSON でないものは投げる', () => {
    expect(() => parseAllowlist(text([BRACES_EXCEPTION, BRACES_EXCEPTION]))).toThrow('重複')
    expect(() => parseAllowlist(JSON.stringify({}))).toThrow('exceptions')
    expect(() => parseAllowlist('{')).toThrow('JSON として読めない')
  })

  test('リポジトリの例外リストそのものが読める', () => {
    // 例外は無いのが平常（空の exceptions も正しい形）。形が崩れていれば parseAllowlist が投げる
    const allowlist = parseAllowlist(fs.readFileSync(ALLOWLIST_PATH, 'utf8'))
    expect(Array.isArray(allowlist)).toBe(true)
    const here = path.dirname(fileURLToPath(import.meta.url))
    expect(path.resolve(ALLOWLIST_PATH)).toBe(path.resolve(here, '..', 'npm_audit_allowlist.json'))
  })
})

describe('npm_audit_gate: 判定', () => {
  const full = parseAuditReport(report(CHAINED))

  test('例外の条件をすべて満たせば通す', () => {
    const r = evaluateAudit({ full, prod: [], allowlist: [BRACES_EXCEPTION], now: NOW })
    expect(r.ok).toBe(true)
    expect(r.accepted.map((x) => x.advisory.id)).toEqual(['GHSA-aaaa-bbbb-cccc'])
    expect(r.warnings).toEqual([])
  })

  test('例外リストに無い high 以上は落とす', () => {
    const r = evaluateAudit({ full, prod: [], allowlist: [], now: NOW })
    expect(r.ok).toBe(false)
    expect(r.failures.map((f) => f.reason)).toEqual(['例外リストに無い'])
  })

  test('moderate 以下は例外リストに無くても落とさない', () => {
    const moderate = full.map((a) => ({ ...a, severity: 'moderate' }))
    expect(evaluateAudit({ full: moderate, prod: moderate, allowlist: [], now: NOW }).ok).toBe(true)
  })

  test('本番依存に出たら例外リストにあっても落とす（例外は開発用依存にしか効かない）', () => {
    const r = evaluateAudit({ full, prod: full, allowlist: [BRACES_EXCEPTION], now: NOW })
    expect(r.ok).toBe(false)
    expect(r.failures).toHaveLength(1)
    expect(r.failures[0].reason).toContain('本番依存')
    // 例外は「使われていない」とは言わない
    expect(r.warnings).toEqual([])
  })

  test('advisory の対象範囲が変わったら落とす（修正版が出た可能性）', () => {
    const r = evaluateAudit({ full, prod: [], allowlist: [{ ...BRACES_EXCEPTION, range: '<=3.0.2' }], now: NOW })
    expect(r.ok).toBe(false)
    expect(r.failures[0].reason).toContain('対象範囲')
  })

  test('パッケージ名が違えば落とす', () => {
    const r = evaluateAudit({ full, prod: [], allowlist: [{ ...BRACES_EXCEPTION, package: 'micromatch' }], now: NOW })
    expect(r.ok).toBe(false)
    expect(r.failures[0].reason).toContain('パッケージ名')
  })

  test('見直し期限の当日までは通し、翌日（UTC）から落とす', () => {
    const allowlist = [BRACES_EXCEPTION]
    expect(evaluateAudit({ full, prod: [], allowlist, now: new Date('2024-03-01T23:00:00Z') }).ok).toBe(true)
    const r = evaluateAudit({ full, prod: [], allowlist, now: new Date('2024-03-02T00:00:00Z') })
    expect(r.ok).toBe(false)
    expect(r.failures[0].reason).toContain('見直し期限')
  })

  test('当たる advisory が無い例外は警告だけ出し、落とさない', () => {
    const r = evaluateAudit({ full: [], prod: [], allowlist: [BRACES_EXCEPTION], now: NOW })
    expect(r.ok).toBe(true)
    expect(r.warnings).toHaveLength(1)
    expect(r.warnings[0]).toContain('GHSA-aaaa-bbbb-cccc')
  })
})

describe('npm_audit_gate: npm の呼び方', () => {
  test('npm run 経由なら npm-cli.js を node で直接叩く', () => {
    const inv = npmInvocation({ env: { npm_execpath: '/x/npm-cli.js' }, platform: 'win32', execPath: '/node' })
    expect(inv).toEqual({ command: '/node', prefix: ['/x/npm-cli.js'] })
  })

  test('直接実行の Windows は cmd 経由（npm.cmd を shell: true 無しで解決するため）', () => {
    const inv = npmInvocation({ env: { ComSpec: 'C:\\cmd.exe' }, platform: 'win32', execPath: '/node' })
    expect(inv).toEqual({ command: 'C:\\cmd.exe', prefix: ['/d', '/s', '/c', 'npm'] })
  })

  test('それ以外は npm をそのまま呼ぶ', () => {
    expect(npmInvocation({ env: {}, platform: 'linux', execPath: '/node' })).toEqual({ command: 'npm', prefix: [] })
  })
})
