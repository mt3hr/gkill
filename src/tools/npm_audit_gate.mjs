#!/usr/bin/env node
// 編集前に読む: .claude/skills/gkill-build-test/SKILL.md「Nightly の npm audit と例外リスト」
//
// Nightly の npm audit の判定。`npm audit --audit-level=high` と同じく high 以上の advisory があれば落とすが、
// src/tools/npm_audit_allowlist.json で名指しした advisory だけは、次をすべて満たす間に限って通す。
//   - 本番依存（--omit=dev）の監査に出てこない。本番依存の high 以上は例外リストに関係なく落とす
//   - 例外に書いたパッケージ名・対象範囲が advisory の今の内容と一致する
//     （対象範囲が変わった＝修正版が出た可能性があるので、人が見直すまで落とす）
//   - 見直し期限（review_by、UTC の日付）を過ぎていない
// npm audit の出力が読めない・例外リストの形が違うときも落とす（fail-closed）。
// どの advisory にも当たらなかった例外は「使われていない例外」として警告だけ出す。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
export const ALLOWLIST_PATH = path.join(ROOT, 'src', 'tools', 'npm_audit_allowlist.json')

// npm audit の severity の並び。--audit-level=high と同じく high 以上を落とす
const SEVERITY_ORDER = ['info', 'low', 'moderate', 'high', 'critical']
export const FAIL_LEVEL = 'high'

// 知らない severity は重いものとして扱う（fail-closed）
export function isAtLeast(severity, level = FAIL_LEVEL) {
  const s = SEVERITY_ORDER.indexOf(severity)
  return s === -1 || s >= SEVERITY_ORDER.indexOf(level)
}

// advisory の URL から GHSA ID を取る。取れなければ npm の source 番号で代用する
export function advisoryId(via) {
  const m = String(via.url || '').match(/GHSA(?:-[0-9a-z]{4}){3}/i)
  return m ? m[0] : `npm:${via.source}`
}

// `npm audit --json`（auditReportVersion 2）から advisory の一覧を取り出す。
// vulnerabilities の各エントリの via のうち、オブジェクトのものが advisory 本体。文字列のものは
// 「この依存を経由して影響を受ける」という連鎖で、advisory は経由先のエントリに載っている
// （braces の 1 件が micromatch・fast-glob・… と連鎖して high 6 件と数えられる）。
export function parseAuditReport(text) {
  let report
  try {
    report = JSON.parse(text)
  } catch (e) {
    throw new Error(`npm audit の出力が JSON として読めない: ${e.message}`)
  }
  if (report?.error) {
    const err = report.error
    throw new Error(`npm audit がエラーを返した: ${err.summary || err.code || JSON.stringify(err)}`)
  }
  if (report?.auditReportVersion !== 2 || typeof report.vulnerabilities !== 'object' || report.vulnerabilities === null) {
    throw new Error('npm audit の出力の形が想定と違う（auditReportVersion 2 の vulnerabilities が無い）')
  }
  const seen = new Map()
  for (const entry of Object.values(report.vulnerabilities)) {
    for (const via of entry?.via || []) {
      if (typeof via !== 'object' || via === null) continue
      const advisory = {
        id: advisoryId(via),
        package: via.name,
        severity: via.severity,
        range: via.range,
        title: via.title,
        url: via.url,
      }
      const key = `${advisory.id}\0${advisory.package}\0${advisory.range}`
      if (!seen.has(key)) seen.set(key, advisory)
    }
  }
  const advisories = [...seen.values()].sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))
  // エントリの severity は連鎖の最大値なので、high 以上のエントリがあれば high 以上の advisory も必ずある。
  // 無いなら via の形が変わって advisory を拾えていない。黙って 0 件で通さずに止める
  const highEntry = Object.values(report.vulnerabilities).some((e) => isAtLeast(e?.severity))
  if (highEntry && !advisories.some((a) => isAtLeast(a.severity))) {
    throw new Error('npm audit の出力に high 以上のエントリがあるのに advisory を拾えない（via の形が変わった？）')
  }
  return advisories
}

const EXCEPTION_KEYS = ['id', 'package', 'range', 'reason', 'review_by']

// 例外リストを読む。欠けた項目・日付でない review_by・同じ ID の重複は投げる（黙って無視すると例外が効かなくなるか、
// 逆に効きすぎる）
export function parseAllowlist(text) {
  let data
  try {
    data = JSON.parse(text)
  } catch (e) {
    throw new Error(`例外リストが JSON として読めない: ${e.message}`)
  }
  if (!Array.isArray(data?.exceptions)) throw new Error('例外リストに exceptions の配列が無い')
  const ids = new Set()
  return data.exceptions.map((ex, i) => {
    for (const key of EXCEPTION_KEYS) {
      if (typeof ex?.[key] !== 'string' || ex[key].trim() === '') {
        throw new Error(`例外リストの exceptions[${i}] に ${key} が無い`)
      }
    }
    if (!isCalendarDate(ex.review_by)) {
      throw new Error(`例外リストの exceptions[${i}] の review_by が YYYY-MM-DD の日付でない: ${ex.review_by}`)
    }
    if (ids.has(ex.id)) throw new Error(`例外リストで ${ex.id} が重複している`)
    ids.add(ex.id)
    return Object.fromEntries(EXCEPTION_KEYS.map((k) => [k, ex[k]]))
  })
}

function isCalendarDate(s) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return false
  const d = new Date(`${s}T00:00:00Z`)
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === s
}

// 判定。full は全依存、prod は本番依存だけの advisory 一覧（parseAuditReport の戻り値）。
// 戻り値: { ok, failures: [{ advisory, reason }], accepted: [{ advisory, exception }], warnings: [string] }
export function evaluateAudit({ full, prod, allowlist, now = new Date() }) {
  const today = now.toISOString().slice(0, 10)
  const failures = []
  const accepted = []
  const used = new Set()

  // 例外は開発用依存にしか効かない。本番依存の high 以上は無条件で落とす
  const prodHigh = prod.filter((a) => isAtLeast(a.severity))
  const prodHighIds = new Set(prodHigh.map((a) => a.id))
  for (const a of prodHigh) {
    if (allowlist.some((e) => e.id === a.id)) used.add(a.id)
    failures.push({ advisory: a, reason: '本番依存に入っている（例外リストは開発用依存にしか効かない）' })
  }

  for (const a of full.filter((x) => isAtLeast(x.severity))) {
    if (prodHighIds.has(a.id)) continue
    const ex = allowlist.find((e) => e.id === a.id)
    if (!ex) {
      failures.push({ advisory: a, reason: '例外リストに無い' })
      continue
    }
    used.add(ex.id)
    if (ex.package !== a.package) {
      failures.push({ advisory: a, reason: `例外リストのパッケージ名（${ex.package}）と違う` })
    } else if (ex.range !== a.range) {
      failures.push({
        advisory: a,
        reason: `advisory の対象範囲が例外リストの ${ex.range} から変わった。修正版が出ていないか確かめ、例外を見直すこと`,
      })
    } else if (today > ex.review_by) {
      failures.push({
        advisory: a,
        reason: `例外の見直し期限 ${ex.review_by} を過ぎた。修正版の有無を確かめ、例外を消すか期限を延ばすこと`,
      })
    } else {
      accepted.push({ advisory: a, exception: ex })
    }
  }

  const warnings = allowlist
    .filter((ex) => !used.has(ex.id))
    .map((ex) => `例外 ${ex.id}（${ex.package}）に当たる advisory が無い。不要になったなら例外リストから消すこと`)

  return { ok: failures.length === 0, failures, accepted, warnings }
}

// npm を子プロセスで呼ぶ形。npm run 経由なら npm_execpath に npm-cli.js が入っているので node で直接叩く。
// Windows の素の npm は npm.cmd なので cmd 経由で呼ぶ（shell: true は使わない。gradle_test.mjs と同じ理由）
export function npmInvocation({ env = process.env, platform = process.platform, execPath = process.execPath } = {}) {
  const npmCli = env.npm_execpath
  if (npmCli && /\.c?js$/i.test(npmCli)) return { command: execPath, prefix: [npmCli] }
  if (platform === 'win32') return { command: env.ComSpec || env.comspec || 'cmd.exe', prefix: ['/d', '/s', '/c', 'npm'] }
  return { command: 'npm', prefix: [] }
}

function runNpmAudit(extraArgs, cwd) {
  const { command, prefix } = npmInvocation()
  const res = spawnSync(command, [...prefix, 'audit', '--json', ...extraArgs], {
    cwd,
    encoding: 'utf8',
    maxBuffer: 256 * 1024 * 1024,
  })
  if (res.error) throw new Error(`npm audit を実行できない: ${res.error.message}`)
  // 脆弱性があると npm audit は exit 1 で終わる。判定は JSON で行うので終了コードは見ない
  // （出力が空・エラーの JSON なら parseAuditReport が投げる）
  return res.stdout
}

function formatAdvisory(a) {
  return `${a.id} ${a.package} ${a.range}（${a.severity}）${a.title ? ' ' + a.title : ''}`
}

export function runGate({ cwd = ROOT, allowlistPath = ALLOWLIST_PATH, now = new Date(), log = console.log, error = console.error } = {}) {
  const allowlist = parseAllowlist(fs.readFileSync(allowlistPath, 'utf8'))
  const full = parseAuditReport(runNpmAudit([], cwd))
  const prod = parseAuditReport(runNpmAudit(['--omit=dev'], cwd))
  const result = evaluateAudit({ full, prod, allowlist, now })

  log(`npm audit（${FAIL_LEVEL} 以上）: 全依存 ${full.filter((a) => isAtLeast(a.severity)).length} 件 / ` +
    `本番依存 ${prod.filter((a) => isAtLeast(a.severity)).length} 件の advisory、例外 ${allowlist.length} 件`)
  for (const { advisory, exception } of result.accepted) {
    log(`  OK   ${formatAdvisory(advisory)}  例外（見直し期限 ${exception.review_by}）`)
  }
  for (const { advisory, reason } of result.failures) {
    error(`  NG   ${formatAdvisory(advisory)}  ${advisory.url || ''}`)
    error(`       → ${reason}`)
  }
  for (const w of result.warnings) log(`  警告 ${w}`)
  if (!result.ok) {
    error(`\nnpm audit NG ${result.failures.length} 件。例外の条件は src/tools/npm_audit_allowlist.json と src/tools/README.md を参照`)
  }
  return result
}

// テストから import できるよう、直接実行されたときだけ走らせる（Windows はドライブ文字の大小が揺れる）
function isDirectRun() {
  if (!process.argv[1]) return false
  const a = path.resolve(process.argv[1])
  const b = fileURLToPath(import.meta.url)
  return process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b
}

if (isDirectRun()) {
  try {
    process.exit(runGate().ok ? 0 : 1)
  } catch (e) {
    console.error(`npm audit の判定ができない: ${e.message}`)
    process.exit(1)
  }
}
