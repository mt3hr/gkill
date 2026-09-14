// 編集前に読む: .claude/skills/gkill-build-test/SKILL.md「リリースゲート（テスト済み attestation）」
//
// テスト済み attestation の共有ライブラリ。
//   - run_test_suite.mjs      … 各テストスイートが成功したときに記録する側
//   - verify_release_gate.mjs … npm run release の先頭で記録を検査する側
//   - put_version_info.mjs    … version.json に tree hash を焼き込む側
//
// attestation は「commit SHA」ではなく「作業ツリーの tree hash」に束縛する。
// dirty なツリーでテストを通し、そのまま全部コミットすれば HEAD^{tree} が一致するので
// 再テスト無しでリリースできる。release 側は「作業ツリーがクリーン」を別に要求するので、
// 結局 HEAD^{tree} == attested tree ＝「テスト済みコミット」と同値になる。
//
// git の呼び出しはすべて `-c core.autocrlf=true` を付ける。WSL 側（release が走る側）は
// autocrlf 未設定で、Windows が CRLF で checkout した追跡ファイル 184 件が偽 dirty に見える
// （2026-09-14 実測。`-c core.autocrlf=true` を付けると両方 0 件）。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
export const ROOT = path.resolve(__dirname, '..', '..') // src/tools/ → リポジトリルート

// 記録先。`.gitignore` の明示行で ignore されている（`*.local` は「.local で終わる名前」にしか
// 当たらない）。ignore されていないと自分自身が作業ツリーを汚し、tree hash も clean 判定も壊れる。
export const ATTESTATION_FILE = 'test_attestation.local.json'
export const ATTESTATION_SCHEMA = 1

// release が要求するスイート。`npm test` の並び（install_server を除く）と同じ集合。
export const REQUIRED_SUITES = [
  'verify_docs',
  'test_server',
  'test_client_unit',
  'test_client_e2e',
  'test_mcp',
  'test_tools',
  'test_plugins',
  'test_android',
  'test_wear_os',
]

// Nightly（govulncheck / npm audit）の結果が「この HEAD にも当てはまる」と言える条件:
// Nightly が見た SHA から HEAD までの間に依存の宣言が変わっていないこと。
export const DEPENDENCY_MANIFESTS = [
  'package.json',
  'package-lock.json',
  'src/server/go.mod',
  'src/server/go.sum',
  'src/plugins/*/go.mod',
  'src/plugins/*/go.sum',
]

export const NIGHTLY_MAX_AGE_HOURS = 48

// ─────────────────────────────────────────────────────────────
// git
// ─────────────────────────────────────────────────────────────

// GIT_INDEX_FILE は pre-commit フックの中で export されている（commit -a なら index.lock を指す）。
// このライブラリは実 index を触らないので、環境から外して git の既定へ戻す。
function gitEnv(extra = {}) {
  const env = { ...process.env, ...extra }
  if (!('GIT_INDEX_FILE' in extra)) delete env.GIT_INDEX_FILE
  return env
}

// 成功時は stdout（trim 済み）を返し、失敗時は投げる。stderr は捕捉する
// （autocrlf の「LF will be replaced by CRLF」警告を垂れ流さないため）。
export function git(args, { cwd = ROOT, env = {}, allowStatus = [0] } = {}) {
  const res = spawnSync('git', ['-c', 'core.autocrlf=true', ...args], {
    cwd,
    env: gitEnv(env),
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  })
  if (res.error) throw new Error(`git ${args[0]} を実行できない: ${res.error.message}`)
  if (!allowStatus.includes(res.status)) {
    throw new Error(`git ${args.join(' ')} が失敗 (exit ${res.status}): ${(res.stderr || '').trim()}`)
  }
  return { status: res.status, stdout: (res.stdout || '').trim(), stderr: (res.stderr || '').trim() }
}

export function head(cwd = ROOT) {
  return git(['rev-parse', 'HEAD'], { cwd }).stdout
}

export function headTree(cwd = ROOT) {
  return git(['rev-parse', 'HEAD^{tree}'], { cwd }).stdout
}

// 作業ツリーの変更一覧。空ならクリーン。untracked も含む。
//
// `git status` は使わない。status は index の stat cache と「サイズが違う」だけで内容を
// 読まずに modified と言う（ie_modified の DATA_CHANGED 短絡）ので、CRLF/LF の差だけで
// サイズが変わったファイルが autocrlf=true でも偽 dirty になる。`git diff HEAD` は
// 内容を改行変換のうえで比べる（diffcore_skip_stat_unmatch）ので、こちらを使う。
export function dirtyEntries(cwd = ROOT) {
  const modified = git(['diff', '--name-only', 'HEAD', '--'], { cwd }).stdout
  const untracked = git(['ls-files', '--others', '--exclude-standard'], { cwd }).stdout
  return [
    ...(modified === '' ? [] : modified.split('\n').map((p) => ` M ${p}`)),
    ...(untracked === '' ? [] : untracked.split('\n').map((p) => `?? ${p}`)),
  ]
}

export function isClean(cwd = ROOT) {
  return dirtyEntries(cwd).length === 0
}

// attestation ファイルが ignore されていることの確認。されていないと自己参照で壊れるので、
// 記録・検査の前に必ず呼ぶ。
export function assertAttestationIgnored(cwd = ROOT) {
  const res = git(['check-ignore', '-q', ATTESTATION_FILE], { cwd, allowStatus: [0, 1] })
  if (res.status !== 0) {
    throw new Error(
      `${ATTESTATION_FILE} が .gitignore で無視されていない。` +
      '明示行（*.local では当たらない）を .gitignore に足すこと')
  }
}

// 作業ツリー全体（追跡ファイルの変更 + untracked の非 ignore ファイル）の tree hash。
//
// クリーンなら HEAD^{tree} をそのまま返す。WSL の /mnt/c では Windows が書いた index の
// stat 情報が合わず全ファイルを再ハッシュするため、dirty のときだけ一時 index を使う:
//   実 index を tmp へコピー（stat cache を引き継ぐ）→ GIT_INDEX_FILE=tmp で `git add -A`
//   → `git write-tree`。実 index には触らない。
export function workingTree(cwd = ROOT) {
  assertAttestationIgnored(cwd)
  if (isClean(cwd)) return headTree(cwd)

  const gitDirRaw = git(['rev-parse', '--git-dir'], { cwd }).stdout
  const gitDir = path.isAbsolute(gitDirRaw) ? gitDirRaw : path.resolve(cwd, gitDirRaw)
  const realIndex = path.join(gitDir, 'index')
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gkill-attest-'))
  const tmpIndex = path.join(tmpDir, 'index')
  try {
    const env = { GIT_INDEX_FILE: tmpIndex }
    if (fs.existsSync(realIndex)) {
      fs.copyFileSync(realIndex, tmpIndex)
    } else {
      git(['read-tree', 'HEAD'], { cwd, env })
    }
    git(['add', '-A'], { cwd, env })
    return git(['write-tree'], { cwd, env }).stdout
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true })
  }
}

export function isAncestor(sha, ofSha = 'HEAD', cwd = ROOT) {
  // exit 0 = 祖先、1 = 祖先でない、それ以外 = SHA が無い等のエラー
  const res = git(['merge-base', '--is-ancestor', sha, ofSha], { cwd, allowStatus: [0, 1] })
  return res.status === 0
}

// `git diff --quiet` は 0 = 差分なし、1 = 差分あり、2 以上 = エラー（SHA 未取得など）。
// execFileSync は 1 でも投げてしまうので、ここで区別する。
export function pathsChangedBetween(fromSha, toSha, pathspecs, cwd = ROOT) {
  const res = git(['diff', '--quiet', fromSha, toSha, '--', ...pathspecs], { cwd, allowStatus: [0, 1] })
  return res.status === 1
}

// remote URL → { owner, repo }。https:// と git@host:owner/repo.git の両形式。
export function parseGithubRepo(remoteUrl) {
  const s = String(remoteUrl || '').trim()
  const m = s.match(/^(?:https?:\/\/[^/]+\/|ssh:\/\/[^/]+\/|[^@]+@[^:]+:)([^/]+)\/([^/]+?)(?:\.git)?\/?$/)
  if (!m) return null
  return { owner: m[1], repo: m[2] }
}

export function remoteGithubRepo(cwd = ROOT) {
  const url = git(['config', '--get', 'remote.origin.url'], { cwd, allowStatus: [0, 1] }).stdout
  return parseGithubRepo(url)
}

// ─────────────────────────────────────────────────────────────
// attestation ファイル
// ─────────────────────────────────────────────────────────────

// release/ に残す成果物名。verify_release_artifacts.mjs が SHA256SUMS に載せる
export function releaseAttestationName(version) {
  return `RELEASE_ATTESTATION_${version}.json`
}

export function attestationPath(root = ROOT) {
  return path.join(root, ATTESTATION_FILE)
}

export function readAttestation(root = ROOT) {
  const p = attestationPath(root)
  if (!fs.existsSync(p)) return { schema: ATTESTATION_SCHEMA, suites: {} }
  let parsed
  try {
    parsed = JSON.parse(fs.readFileSync(p, 'utf8'))
  } catch {
    // 壊れたファイルは「記録なし」として扱う（ゲートが不足として列挙する）
    return { schema: ATTESTATION_SCHEMA, suites: {} }
  }
  if (!parsed || typeof parsed !== 'object' || parsed.schema !== ATTESTATION_SCHEMA ||
      !parsed.suites || typeof parsed.suites !== 'object') {
    return { schema: ATTESTATION_SCHEMA, suites: {} }
  }
  return parsed
}

// 1 スイート分を追記する。並走する別のターミナルの記録を消さないよう、
// rename 直前に読み直してからマージし、temp+rename で原子的に置き換える。
export function recordSuite(suite, entry, root = ROOT) {
  const p = attestationPath(root)
  const current = readAttestation(root)
  current.suites[suite] = entry
  const tmp = `${p}.${process.pid}.tmp`
  fs.writeFileSync(tmp, JSON.stringify(current, null, 2) + '\n')
  fs.renameSync(tmp, p)
  return current
}

// ─────────────────────────────────────────────────────────────
// 評価（純粋関数。テストはここを直接叩く）
// ─────────────────────────────────────────────────────────────

// 各必須スイートが tree に対して記録済みかを判定する。
// 戻り値: [{ suite, ok, reason, entry }]
export function evaluateSuites(attestation, tree, requiredSuites = REQUIRED_SUITES) {
  const suites = (attestation && attestation.suites) || {}
  return requiredSuites.map((suite) => {
    const entry = suites[suite]
    if (!entry) return { suite, ok: false, reason: '記録なし', entry: null }
    if (entry.tree !== tree) {
      return {
        suite,
        ok: false,
        reason: `tree 不一致（記録 ${String(entry.tree).slice(0, 7)} ${entry.finished_at || ''}）`.trim(),
        entry,
      }
    }
    if (suite === 'test_client_e2e') {
      if (!entry.server_tree) {
        return { suite, ok: false, reason: 'gkill_server の tree hash が未記録（version.json 無しのビルド）', entry }
      }
      if (entry.server_tree !== tree) {
        return {
          suite,
          ok: false,
          reason: `PATH の gkill_server が別のツリー（${String(entry.server_tree).slice(0, 7)}）から作られている`,
          entry,
        }
      }
    }
    return { suite, ok: true, reason: `記録済み ${entry.finished_at || ''}`.trim(), entry }
  })
}

// GitHub Actions の CI ワークフロー run 一覧（head_sha で絞ったもの）から合否を出す。
export function evaluateCiRuns(runs) {
  const list = Array.isArray(runs) ? runs : []
  const success = list.find((r) => r.status === 'completed' && r.conclusion === 'success')
  if (success) return { ok: true, run: success, reason: 'success' }
  const pending = list.find((r) => r.status !== 'completed')
  if (pending) return { ok: false, run: pending, reason: `CI が ${pending.status}。完了を待つこと` }
  if (list.length === 0) {
    return { ok: false, run: null, reason: 'この HEAD の CI run が無い（push されていない。リリースは push 済みコミットからのみ）' }
  }
  const latest = list[0]
  return { ok: false, run: latest, reason: `CI が ${latest.conclusion}。直してから push し直すこと` }
}

// 直近の完了済み Nightly run が「この HEAD に対しても有効」かを判定する。
//   run            … GitHub API の run オブジェクト（無ければ null）
//   isAncestor     … (sha) => boolean。run.head_sha が HEAD の祖先か
//   depsChangedSince … (sha) => boolean。run.head_sha 以後に依存宣言が変わったか
//   now            … Date
export function evaluateNightly(run, { isAncestor, depsChangedSince, now = new Date(), maxAgeHours = NIGHTLY_MAX_AGE_HOURS }) {
  if (!run) return { ok: false, reason: 'main の完了済み Nightly run が無い' }
  if (run.conclusion !== 'success') {
    return { ok: false, reason: `直近の Nightly が ${run.conclusion}（${run.head_sha?.slice(0, 7)}）` }
  }
  const created = new Date(run.created_at)
  const ageHours = (now.getTime() - created.getTime()) / 3600000
  if (!(ageHours <= maxAgeHours)) {
    return { ok: false, reason: `直近の Nightly が ${Math.floor(ageHours)} 時間前（${maxAgeHours} 時間以内が必要。workflow_dispatch で回して待つこと）` }
  }
  if (!isAncestor(run.head_sha)) {
    return { ok: false, reason: `Nightly が見た ${run.head_sha.slice(0, 7)} が HEAD の祖先でない（git fetch origin main が要るか、別ブランチ）` }
  }
  if (depsChangedSince(run.head_sha)) {
    return { ok: false, reason: `Nightly（${run.head_sha.slice(0, 7)}）以後に依存の宣言が変わっている。Nightly を workflow_dispatch で回して待つこと` }
  }
  return { ok: true, reason: `success ${run.created_at}（${run.head_sha.slice(0, 7)}）` }
}

// ─────────────────────────────────────────────────────────────
// GitHub API
// ─────────────────────────────────────────────────────────────

export async function githubApi(url, { fetchImpl = globalThis.fetch, token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN } = {}) {
  if (typeof fetchImpl !== 'function') throw new Error('fetch が使えない Node。Node 20 以降が必要')
  const headers = {
    Accept: 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
    'User-Agent': 'gkill-verify-release-gate',
  }
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetchImpl(url, { headers })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`GitHub API ${res.status} ${url}: ${body.slice(0, 200)}`)
  }
  return res.json()
}

export function workflowRunsUrl({ owner, repo }, workflowFile, params) {
  const q = new URLSearchParams(params).toString()
  return `https://api.github.com/repos/${owner}/${repo}/actions/workflows/${workflowFile}/runs?${q}`
}
