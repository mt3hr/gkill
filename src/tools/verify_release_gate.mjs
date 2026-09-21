#!/usr/bin/env node
// 編集前に読む: .claude/skills/gkill-build-test/SKILL.md「リリースゲート（テスト済み attestation）」
//
// リリースゲート。npm run release の先頭で走り、「テスト済みのコミットしかリリースしない」を
// 人の記憶ではなく機械で強制する（指摘 F-006 は
// リリース工程に検証ギャップがあった実例）。1 つでも欠ければ非 0 で止まり、抜け道フラグは無い。
//
// 検査（すべて fail-closed。ネットワーク不通・API エラーも NG）:
//   1. 作業ツリーがクリーン（untracked も不可）。dirty なら何がビルドに入るか分からない
//   2. 必須スイート全部が HEAD^{tree} に対して記録済み（run_test_suite.mjs が書く）
//   3. E2E は PATH 上の gkill_server も同じツリーから作られている
//   4. GitHub Actions: CI ワークフローがこの HEAD で success。直近の Nightly が success で、
//      HEAD の祖先・48 時間以内・以後に依存の宣言が変わっていない
//      （Nightly の run 単位 success は govulncheck / npm audit の保証。E2E / Android は
//      continue-on-error なので、その実証は 2 のローカル attestation が担う）
// 通ったら release/RELEASE_ATTESTATION_<version>.json を書き、verify_release_artifacts.mjs が
// それを成果物として SHA256SUMS に載せ、ビルド後にも HEAD が動いていないことを検査する。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import {
  ROOT,
  REQUIRED_SUITES,
  DEPENDENCY_MANIFESTS,
  head,
  headTree,
  dirtyEntries,
  assertAttestationIgnored,
  isAncestor,
  pathsChangedBetween,
  remoteGithubRepo,
  readAttestation,
  releaseAttestationName,
  evaluateSuites,
  evaluateCiRuns,
  evaluateNightly,
  githubApi,
  workflowRunsUrl,
} from './attestation.mjs'

const require = createRequire(import.meta.url)

// テストから差し替えられるよう、入出力はすべて引数で受ける。
// 戻り値: { ok, failed, outFile }
export async function runGate({
  cwd = ROOT,
  version = require(path.join(ROOT, 'package.json')).version,
  fetchImpl = globalThis.fetch,
  now = new Date(),
  releaseDir = path.join(cwd, 'release'),
  log = console.log,
  error = console.error,
} = {}) {
  let failed = 0
  const ok = (label, detail = '') => log(`  OK   ${label}${detail ? '  ' + detail : ''}`)
  const fail = (label, detail = '') => { error(`  NG   ${label}${detail ? '  ' + detail : ''}`); failed++ }

  assertAttestationIgnored(cwd)
  const sha = head(cwd)
  const tree = headTree(cwd)
  log(`リリースゲート: HEAD ${sha.slice(0, 7)} (tree ${tree.slice(0, 7)}) version ${version}`)

  // 1. クリーン
  const dirty = dirtyEntries(cwd)
  if (dirty.length === 0) {
    ok('作業ツリーはクリーン')
  } else {
    fail('作業ツリーに未コミットの変更がある', `${dirty.length} 件: ${dirty.slice(0, 5).join(' / ')}${dirty.length > 5 ? ' …' : ''}`)
  }

  // 2 + 3. attestation
  const attestation = readAttestation(cwd)
  const results = evaluateSuites(attestation, tree, REQUIRED_SUITES)
  for (const r of results) {
    if (r.ok) ok(r.suite.padEnd(18), r.reason)
    else fail(r.suite.padEnd(18), r.reason)
  }
  const missing = results.filter((r) => !r.ok).map((r) => r.suite)
  if (missing.length > 0) {
    error(`       → このツリーで ${missing.map((s) => `npm run ${s}`).join(' / ')} を通して記録すること`)
    if (missing.includes('test_client_e2e')) {
      error('       → E2E は PATH の gkill_server が今のツリーのビルドであることも要る（npm run put_version_info_embed → ビルド → E2E）')
    }
  }

  // 4. GitHub
  let ciRun = null
  let nightlyRun = null
  const repo = remoteGithubRepo(cwd)
  if (!repo) {
    fail('GitHub リポジトリ', 'remote.origin.url から owner/repo を読めない')
  } else {
    try {
      const ci = await githubApi(workflowRunsUrl(repo, 'ci.yml', { head_sha: sha, per_page: '10' }), { fetchImpl })
      const ev = evaluateCiRuns(ci.workflow_runs)
      ciRun = ev.run
      const link = ev.run ? ` ${ev.run.html_url}` : ''
      if (ev.ok) ok('CI (ci.yml)'.padEnd(18), `${ev.reason}${link}`)
      else fail('CI (ci.yml)'.padEnd(18), `${ev.reason}${link}`)
    } catch (e) {
      fail('CI (ci.yml)'.padEnd(18), e.message)
    }
    try {
      const nightly = await githubApi(
        workflowRunsUrl(repo, 'nightly.yml', { branch: 'main', status: 'completed', per_page: '1' }), { fetchImpl })
      const run = (nightly.workflow_runs || [])[0] || null
      const ev = evaluateNightly(run, {
        isAncestor: (s) => isAncestor(s, 'HEAD', cwd),
        depsChangedSince: (s) => pathsChangedBetween(s, 'HEAD', DEPENDENCY_MANIFESTS, cwd),
        now,
      })
      nightlyRun = run
      const link = run ? ` ${run.html_url}` : ''
      if (ev.ok) ok('Nightly'.padEnd(18), `${ev.reason}${link}`)
      else fail('Nightly'.padEnd(18), `${ev.reason}${link}`)
    } catch (e) {
      fail('Nightly'.padEnd(18), e.message)
    }
  }

  if (failed > 0) {
    error(`\nリリースゲート NG ${failed} 件。release を中止しました（抜け道はありません）。`)
    return { ok: false, failed, outFile: null }
  }

  // 5. 成果物として残す
  fs.mkdirSync(releaseDir, { recursive: true })
  const outFile = path.join(releaseDir, releaseAttestationName(version))
  const record = {
    version,
    head: sha,
    tree,
    gate_time: now.toISOString(),
    suites: Object.fromEntries(results.map((r) => [r.suite, r.entry])),
    ci_run: ciRun ? { id: ciRun.id, url: ciRun.html_url, head_sha: ciRun.head_sha } : null,
    nightly_run: nightlyRun
      ? { id: nightlyRun.id, url: nightlyRun.html_url, head_sha: nightlyRun.head_sha, created_at: nightlyRun.created_at }
      : null,
  }
  fs.writeFileSync(outFile, JSON.stringify(record, null, 2) + '\n')
  log(`\nリリースゲート OK。${path.relative(cwd, outFile)} を書き出しました`)
  return { ok: true, failed: 0, outFile }
}

// テストから import できるよう、直接実行されたときだけ走らせる（Windows はドライブ文字の大小が揺れる）
function isDirectRun() {
  if (!process.argv[1]) return false
  const a = path.resolve(process.argv[1])
  const b = fileURLToPath(import.meta.url)
  return process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b
}

if (isDirectRun()) {
  runGate()
    .then((r) => process.exit(r.ok ? 0 : 1))
    .catch((e) => {
      console.error(`リリースゲートが実行できない: ${e.message}`)
      process.exit(1)
    })
}
