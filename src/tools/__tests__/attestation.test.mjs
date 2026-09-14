// テスト済み attestation（記録・評価・リリースゲート）のテスト。
//
// git を伴うものは os.tmpdir() に使い捨てのリポジトリを作って回す。
// リポジトリ本体の index・作業ツリーには一切触らない。
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import {
  ATTESTATION_FILE,
  REQUIRED_SUITES,
  NIGHTLY_MAX_AGE_HOURS,
  head,
  headTree,
  isClean,
  workingTree,
  assertAttestationIgnored,
  isAncestor,
  pathsChangedBetween,
  parseGithubRepo,
  readAttestation,
  recordSuite,
  releaseAttestationName,
  evaluateSuites,
  evaluateCiRuns,
  evaluateNightly,
  githubApi,
  workflowRunsUrl,
} from '../attestation.mjs'
import { SUITES, firstDisallowedArg } from '../run_test_suite.mjs'
import { runGate } from '../verify_release_gate.mjs'

// ---------------------------------------------------------------------------
// 使い捨て git リポジトリ
// ---------------------------------------------------------------------------
const tmpRoots = []

function sh(cwd, args) {
  const res = spawnSync('git', args, { cwd, encoding: 'utf8' })
  if (res.status !== 0) throw new Error(`git ${args.join(' ')}: ${res.stderr}`)
  return res.stdout.trim()
}

// 署名や利用者設定に依存しないよう、全部 -c で固定する
function makeRepo({ ignoreAttestation = true } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'gkill-attest-test-'))
  tmpRoots.push(dir)
  sh(dir, ['init', '-q', '-b', 'main'])
  sh(dir, ['config', 'user.email', 't@example.com'])
  sh(dir, ['config', 'user.name', 't'])
  sh(dir, ['config', 'commit.gpgsign', 'false'])
  sh(dir, ['config', 'core.autocrlf', 'false'])
  sh(dir, ['remote', 'add', 'origin', 'https://github.com/o/r.git'])
  fs.writeFileSync(path.join(dir, '.gitignore'), ignoreAttestation ? `${ATTESTATION_FILE}\n` : 'nothing\n')
  fs.writeFileSync(path.join(dir, 'a.txt'), 'hello\n')
  fs.writeFileSync(path.join(dir, 'package.json'), '{ "name": "x", "version": "0.0.1" }\n')
  sh(dir, ['add', '-A'])
  sh(dir, ['commit', '-q', '-m', 'init'])
  return dir
}

function commitAll(dir, message) {
  sh(dir, ['add', '-A'])
  sh(dir, ['commit', '-q', '-m', message])
  return sh(dir, ['rev-parse', 'HEAD'])
}

afterAll(() => {
  for (const d of tmpRoots) fs.rmSync(d, { recursive: true, force: true })
})

// ---------------------------------------------------------------------------
// parseGithubRepo
// ---------------------------------------------------------------------------
describe('parseGithubRepo', () => {
  test.each([
    ['https://github.com/o/r.git', { owner: 'o', repo: 'r' }],
    ['https://github.com/o/r', { owner: 'o', repo: 'r' }],
    ['https://github.com/o/r/', { owner: 'o', repo: 'r' }],
    ['git@example.com:o/r.git', { owner: 'o', repo: 'r' }],
    ['ssh://git@example.com/o/r.git', { owner: 'o', repo: 'r' }],
    ['my-org/my.repo', null],
    ['', null],
    [undefined, null],
  ])('%s', (url, expected) => {
    expect(parseGithubRepo(url)).toEqual(expected)
  })
})

// ---------------------------------------------------------------------------
// evaluateSuites
// ---------------------------------------------------------------------------
describe('evaluateSuites', () => {
  const T = 'a'.repeat(40)
  const entry = (tree, extra = {}) => ({ tree, head: 'h', finished_at: '2026-09-14T00:00:00Z', ...extra })

  test('全スイートが同じ tree で記録済みなら全部 ok', () => {
    const att = { schema: 1, suites: {} }
    for (const s of REQUIRED_SUITES) att.suites[s] = entry(T, s === 'test_client_e2e' ? { server_tree: T } : {})
    const r = evaluateSuites(att, T)
    expect(r.every((x) => x.ok)).toBe(true)
    expect(r.map((x) => x.suite)).toEqual(REQUIRED_SUITES)
  })

  test('記録が無いスイートは「記録なし」', () => {
    const r = evaluateSuites({ schema: 1, suites: {} }, T, ['test_mcp'])
    expect(r).toEqual([{ suite: 'test_mcp', ok: false, reason: '記録なし', entry: null }])
  })

  test('tree が違う記録は不一致（記録側の先頭7桁と時刻を出す）', () => {
    const r = evaluateSuites({ schema: 1, suites: { test_mcp: entry('b'.repeat(40)) } }, T, ['test_mcp'])
    expect(r[0].ok).toBe(false)
    expect(r[0].reason).toContain('不一致')
    expect(r[0].reason).toContain('bbbbbbb')
    expect(r[0].reason).toContain('2026-09-14T00:00:00Z')
  })

  test('E2E は gkill_server の tree も要る（未記録・不一致は ok にしない）', () => {
    const missing = evaluateSuites({ schema: 1, suites: { test_client_e2e: entry(T, { server_tree: null }) } }, T, ['test_client_e2e'])
    expect(missing[0].ok).toBe(false)
    expect(missing[0].reason).toContain('gkill_server')
    const stale = evaluateSuites({ schema: 1, suites: { test_client_e2e: entry(T, { server_tree: 'c'.repeat(40) }) } }, T, ['test_client_e2e'])
    expect(stale[0].ok).toBe(false)
    expect(stale[0].reason).toContain('ccccccc')
    const fine = evaluateSuites({ schema: 1, suites: { test_client_e2e: entry(T, { server_tree: T }) } }, T, ['test_client_e2e'])
    expect(fine[0].ok).toBe(true)
  })

  test('attestation が null でも落ちない', () => {
    expect(evaluateSuites(null, T, ['test_mcp'])[0].ok).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// evaluateCiRuns / evaluateNightly
// ---------------------------------------------------------------------------
describe('evaluateCiRuns', () => {
  test('completed + success があれば ok（並んでいる順に関係なく）', () => {
    const r = evaluateCiRuns([
      { id: 2, status: 'in_progress', conclusion: null },
      { id: 1, status: 'completed', conclusion: 'success' },
    ])
    expect(r.ok).toBe(true)
    expect(r.run.id).toBe(1)
  })

  test('未完了だけなら待つよう言う', () => {
    const r = evaluateCiRuns([{ id: 2, status: 'queued', conclusion: null }])
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('queued')
  })

  test('run が 1 つも無ければ push されていない', () => {
    const r = evaluateCiRuns([])
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('push')
    expect(evaluateCiRuns(undefined).ok).toBe(false)
  })

  test('失敗した run しか無ければその結論を出す', () => {
    const r = evaluateCiRuns([{ id: 3, status: 'completed', conclusion: 'failure' }])
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('failure')
  })
})

describe('evaluateNightly', () => {
  const now = new Date('2026-09-14T12:00:00Z')
  const fresh = { id: 9, conclusion: 'success', head_sha: 'd'.repeat(40), created_at: '2026-09-13T15:17:00Z', html_url: 'u' }
  const yes = () => true
  const no = () => false

  test('success・48h 以内・祖先・依存未変更なら ok', () => {
    expect(evaluateNightly(fresh, { isAncestor: yes, depsChangedSince: no, now }).ok).toBe(true)
  })

  test('run が無ければ ok にしない', () => {
    expect(evaluateNightly(null, { isAncestor: yes, depsChangedSince: no, now }).ok).toBe(false)
  })

  test('conclusion が success 以外', () => {
    const r = evaluateNightly({ ...fresh, conclusion: 'failure' }, { isAncestor: yes, depsChangedSince: no, now })
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('failure')
  })

  test(`${NIGHTLY_MAX_AGE_HOURS} 時間より古い`, () => {
    const old = { ...fresh, created_at: '2026-09-10T15:17:00Z' }
    const r = evaluateNightly(old, { isAncestor: yes, depsChangedSince: no, now })
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('workflow_dispatch')
  })

  test('HEAD の祖先でない', () => {
    const r = evaluateNightly(fresh, { isAncestor: no, depsChangedSince: no, now })
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('祖先')
  })

  test('以後に依存の宣言が変わっている', () => {
    const r = evaluateNightly(fresh, { isAncestor: yes, depsChangedSince: yes, now })
    expect(r.ok).toBe(false)
    expect(r.reason).toContain('依存')
  })
})

// ---------------------------------------------------------------------------
// ランナーの引数許可リスト
// ---------------------------------------------------------------------------
describe('firstDisallowedArg', () => {
  const e2e = SUITES.test_client_e2e.allowed
  const go = SUITES.test_server.allowed
  const vitest = SUITES.test_client_unit.allowed

  test('引数なしは許可', () => {
    expect(firstDisallowedArg([], e2e)).toBeNull()
    expect(firstDisallowedArg([], [])).toBeNull()
  })

  test('= 区切りも空白区切りも同じ', () => {
    expect(firstDisallowedArg(['--workers=2'], e2e)).toBeNull()
    expect(firstDisallowedArg(['--workers', '2'], e2e)).toBeNull()
    expect(firstDisallowedArg(['-p', '2', '-count=1', '-v'], go)).toBeNull()
    expect(firstDisallowedArg(['--pool=threads', '--maxWorkers', '2', '--no-file-parallelism'], vitest)).toBeNull()
  })

  test('値が要るのに無い・形が違う', () => {
    expect(firstDisallowedArg(['--workers'], e2e)).toBe('--workers')
    expect(firstDisallowedArg(['--workers=many'], e2e)).toBe('--workers=many')
    expect(firstDisallowedArg(['--workers', 'x'], e2e)).toBe('--workers=x')
  })

  test('値を取らないフラグに値が付いている', () => {
    expect(firstDisallowedArg(['-v=1'], go)).toBe('-v=1')
  })

  test('絞り込みや位置引数は最初に外れたものを返す', () => {
    expect(firstDisallowedArg(['--workers=2', 'login.spec.ts'], e2e)).toBe('login.spec.ts')
    expect(firstDisallowedArg(['-run', 'TestX'], go)).toBe('-run')
    expect(firstDisallowedArg(['--grep=foo'], e2e)).toBe('--grep=foo')
  })

  test('許可リストが空のスイートは何を渡しても記録しない', () => {
    expect(firstDisallowedArg(['--list'], SUITES.verify_docs.allowed)).toBe('--list')
    expect(firstDisallowedArg(['vet'], SUITES.test_plugins.allowed)).toBe('vet')
    expect(firstDisallowedArg(['--offline'], SUITES.test_android.allowed)).toBe('--offline')
  })
})

describe('SUITES', () => {
  test('必須スイートと 1:1 で対応している', () => {
    expect(Object.keys(SUITES).sort()).toEqual([...REQUIRED_SUITES].sort())
  })

  test('引数が検査内容を置き換えるスイートは許可リストが空', () => {
    for (const s of ['verify_docs', 'test_plugins', 'test_android', 'test_wear_os']) {
      expect(SUITES[s].allowed).toEqual([])
    }
  })

  test('E2E だけが gkill_server の tree を記録する', () => {
    const rec = Object.entries(SUITES).filter(([, v]) => v.recordsServerTree).map(([k]) => k)
    expect(rec).toEqual(['test_client_e2e'])
  })
})

// ---------------------------------------------------------------------------
// git（使い捨てリポジトリ）
// ---------------------------------------------------------------------------
describe('workingTree / isClean（使い捨て git リポジトリ）', () => {
  test('クリーンなら HEAD^{tree} と一致し、編集・untracked で変わる', () => {
    const dir = makeRepo()
    expect(isClean(dir)).toBe(true)
    expect(workingTree(dir)).toBe(headTree(dir))

    fs.writeFileSync(path.join(dir, 'a.txt'), 'changed\n')
    expect(isClean(dir)).toBe(false)
    const dirtyTree = workingTree(dir)
    expect(dirtyTree).not.toBe(headTree(dir))
    // 実 index には何も stage されない
    expect(sh(dir, ['diff', '--cached', '--name-only'])).toBe('')

    // そのままコミットすれば HEAD^{tree} と一致する（dirty でテスト→コミット→再テスト不要）
    commitAll(dir, 'change')
    expect(headTree(dir)).toBe(dirtyTree)

    fs.writeFileSync(path.join(dir, 'new.txt'), 'x\n')
    expect(workingTree(dir)).not.toBe(headTree(dir))
  })

  test('CRLF で書き換えても tree は変わらない（WSL の偽 dirty を autocrlf=true で吸収する）', () => {
    const dir = makeRepo()
    fs.writeFileSync(path.join(dir, 'a.txt'), 'hello\r\n')
    // このリポジトリは core.autocrlf=false なので素の git は dirty と言う
    expect(sh(dir, ['status', '--porcelain'])).not.toBe('')
    // ライブラリは -c core.autocrlf=true の diff で内容を比べるのでクリーン
    // （status はサイズ差だけで modified と言うので、autocrlf=true を付けても偽 dirty のまま）
    expect(isClean(dir)).toBe(true)
    expect(workingTree(dir)).toBe(headTree(dir))
  })

  test('attestation ファイルが ignore されていないと止まる', () => {
    const dir = makeRepo({ ignoreAttestation: false })
    expect(() => assertAttestationIgnored(dir)).toThrow(/gitignore/)
    expect(() => workingTree(dir)).toThrow(/gitignore/)
  })

  test('attestation ファイルを書いてもクリーンのまま', () => {
    const dir = makeRepo()
    recordSuite('test_mcp', { tree: 't' }, dir)
    expect(isClean(dir)).toBe(true)
  })

  test('isAncestor / pathsChangedBetween', () => {
    const dir = makeRepo()
    const first = head(dir)
    fs.writeFileSync(path.join(dir, 'a.txt'), 'v2\n')
    const second = commitAll(dir, 'no deps change')
    expect(isAncestor(first, 'HEAD', dir)).toBe(true)
    expect(isAncestor('HEAD', first, dir)).toBe(false)
    expect(pathsChangedBetween(first, second, ['package.json', 'src/plugins/*/go.mod'], dir)).toBe(false)

    fs.writeFileSync(path.join(dir, 'package.json'), '{ "name": "x", "version": "0.0.2" }\n')
    const third = commitAll(dir, 'deps change')
    expect(pathsChangedBetween(first, third, ['package.json'], dir)).toBe(true)
    // 無い SHA はエラー（1 ではなく 128）なので投げる
    expect(() => pathsChangedBetween('0'.repeat(40), third, ['package.json'], dir)).toThrow()
  })
})

// ---------------------------------------------------------------------------
// attestation ファイルの読み書き
// ---------------------------------------------------------------------------
describe('readAttestation / recordSuite', () => {
  test('無ければ空、書けば読める、別スイートは残る', () => {
    const dir = makeRepo()
    expect(readAttestation(dir)).toEqual({ schema: 1, suites: {} })
    recordSuite('test_mcp', { tree: 't1' }, dir)
    recordSuite('test_server', { tree: 't2' }, dir)
    recordSuite('test_mcp', { tree: 't3' }, dir)
    expect(readAttestation(dir).suites).toEqual({ test_mcp: { tree: 't3' }, test_server: { tree: 't2' } })
    // temp ファイルが残らない
    expect(fs.readdirSync(dir).filter((f) => f.endsWith('.tmp'))).toEqual([])
  })

  test('壊れた JSON・違うスキーマは「記録なし」扱い', () => {
    const dir = makeRepo()
    fs.writeFileSync(path.join(dir, ATTESTATION_FILE), '{ broken')
    expect(readAttestation(dir)).toEqual({ schema: 1, suites: {} })
    fs.writeFileSync(path.join(dir, ATTESTATION_FILE), JSON.stringify({ schema: 99, suites: { a: {} } }))
    expect(readAttestation(dir)).toEqual({ schema: 1, suites: {} })
  })
})

// ---------------------------------------------------------------------------
// GitHub API
// ---------------------------------------------------------------------------
describe('githubApi / workflowRunsUrl', () => {
  test('URL の形', () => {
    expect(workflowRunsUrl({ owner: 'o', repo: 'r' }, 'ci.yml', { head_sha: 'abc', per_page: '10' }))
      .toBe('https://api.github.com/repos/o/r/actions/workflows/ci.yml/runs?head_sha=abc&per_page=10')
  })

  test('非 2xx は投げる（fail-closed）、トークンがあれば Authorization を付ける', async () => {
    const calls = []
    const fetchImpl = async (url, init) => {
      calls.push({ url, init })
      return { ok: false, status: 403, text: async () => 'rate limited' }
    }
    await expect(githubApi('https://api.github.com/x', { fetchImpl, token: 'tkn' })).rejects.toThrow(/403/)
    expect(calls[0].init.headers.Authorization).toBe('Bearer tkn')
    const noToken = async () => ({ ok: true, status: 200, json: async () => ({ a: 1 }) })
    expect(await githubApi('https://api.github.com/y', { fetchImpl: noToken, token: '' })).toEqual({ a: 1 })
  })

  test('fetch が無い環境は投げる', async () => {
    await expect(githubApi('https://api.github.com/x', { fetchImpl: null })).rejects.toThrow(/fetch/)
  })
})

// ---------------------------------------------------------------------------
// ゲート end-to-end（使い捨てリポジトリ + fetch スタブ）
// ---------------------------------------------------------------------------
describe('runGate', () => {
  const now = new Date('2026-09-14T12:00:00Z')

  function stubFetch({ ciRuns, nightlyRuns }) {
    return async (url) => {
      const body = url.includes('/ci.yml/') ? { workflow_runs: ciRuns } : { workflow_runs: nightlyRuns }
      return { ok: true, status: 200, json: async () => body }
    }
  }

  function attestAll(dir) {
    const tree = headTree(dir)
    for (const s of REQUIRED_SUITES) {
      recordSuite(s, {
        tree, head: head(dir), finished_at: now.toISOString(), platform: 'test', args: [],
        ...(s === 'test_client_e2e' ? { server_tree: tree } : {}),
      }, dir)
    }
  }

  const quiet = { log: () => {}, error: () => {} }

  test('全部揃えば OK で RELEASE_ATTESTATION を書く', async () => {
    const dir = makeRepo()
    attestAll(dir)
    const sha = head(dir)
    const fetchImpl = stubFetch({
      ciRuns: [{ id: 1, status: 'completed', conclusion: 'success', head_sha: sha, html_url: 'ci' }],
      nightlyRuns: [{ id: 2, status: 'completed', conclusion: 'success', head_sha: sha, created_at: '2026-09-13T15:17:00Z', html_url: 'n' }],
    })
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, ...quiet })
    expect(r.ok).toBe(true)
    expect(path.basename(r.outFile)).toBe(releaseAttestationName('9.9.9'))
    const record = JSON.parse(fs.readFileSync(r.outFile, 'utf8'))
    expect(record.head).toBe(sha)
    expect(record.tree).toBe(headTree(dir))
    expect(Object.keys(record.suites).sort()).toEqual([...REQUIRED_SUITES].sort())
    expect(record.ci_run.id).toBe(1)
    expect(record.nightly_run.id).toBe(2)
  })

  test('作業ツリーが dirty なら NG（成果物は書かない）', async () => {
    const dir = makeRepo()
    attestAll(dir)
    fs.writeFileSync(path.join(dir, 'a.txt'), 'dirty\n')
    const sha = head(dir)
    const fetchImpl = stubFetch({
      ciRuns: [{ id: 1, status: 'completed', conclusion: 'success', head_sha: sha }],
      nightlyRuns: [{ id: 2, status: 'completed', conclusion: 'success', head_sha: sha, created_at: '2026-09-13T15:17:00Z' }],
    })
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, ...quiet })
    expect(r.ok).toBe(false)
    expect(r.failed).toBe(1)
    expect(fs.existsSync(path.join(dir, 'release', releaseAttestationName('9.9.9')))).toBe(false)
  })

  test('attestation 不足は 1 スイート 1 件で数える', async () => {
    const dir = makeRepo()
    const sha = head(dir)
    const fetchImpl = stubFetch({
      ciRuns: [{ id: 1, status: 'completed', conclusion: 'success', head_sha: sha }],
      nightlyRuns: [{ id: 2, status: 'completed', conclusion: 'success', head_sha: sha, created_at: '2026-09-13T15:17:00Z' }],
    })
    const errors = []
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, log: () => {}, error: (m) => errors.push(m) })
    expect(r.ok).toBe(false)
    expect(r.failed).toBe(REQUIRED_SUITES.length)
    expect(errors.some((m) => m.includes('npm run test_client_e2e'))).toBe(true)
  })

  test('GitHub が読めなければ NG（ネットワーク不通も fail-closed）', async () => {
    const dir = makeRepo()
    attestAll(dir)
    const fetchImpl = async () => { throw new Error('ENOTFOUND api.github.com') }
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, ...quiet })
    expect(r.ok).toBe(false)
    expect(r.failed).toBe(2) // CI と Nightly
  })

  test('CI が失敗している・Nightly が古い', async () => {
    const dir = makeRepo()
    attestAll(dir)
    const sha = head(dir)
    const fetchImpl = stubFetch({
      ciRuns: [{ id: 1, status: 'completed', conclusion: 'failure', head_sha: sha }],
      nightlyRuns: [{ id: 2, status: 'completed', conclusion: 'success', head_sha: sha, created_at: '2026-09-01T15:17:00Z' }],
    })
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, ...quiet })
    expect(r.ok).toBe(false)
    expect(r.failed).toBe(2)
  })

  test('Nightly 以後に依存の宣言が変わっていれば NG', async () => {
    const dir = makeRepo()
    const nightlySha = head(dir)
    fs.writeFileSync(path.join(dir, 'package.json'), '{ "name": "x", "version": "0.0.2" }\n')
    const sha = commitAll(dir, 'bump dep')
    attestAll(dir)
    const fetchImpl = stubFetch({
      ciRuns: [{ id: 1, status: 'completed', conclusion: 'success', head_sha: sha }],
      nightlyRuns: [{ id: 2, status: 'completed', conclusion: 'success', head_sha: nightlySha, created_at: '2026-09-13T15:17:00Z' }],
    })
    const r = await runGate({ cwd: dir, version: '9.9.9', fetchImpl, now, ...quiet })
    expect(r.ok).toBe(false)
    expect(r.failed).toBe(1)
  })
})
