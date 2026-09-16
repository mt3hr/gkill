// リリース工程の「書く側」のテスト。
//
// attestation.test.mjs は記録の評価とゲート（読む側）を固定している。ここは
//   - run_test_suite.mjs が「どのときだけ記録するか」（失敗・CI・絞り込み引数では記録しない）
//   - put_version_info.mjs が version.json に何を書くか（tree_hash・git が無いときの unknown）
//   - verify_release_artifacts.mjs の成果物一覧・7za 一覧の読み方・debug 署名判定・apksigner 探索
// を固定する。どれも壊れてもビルドは通り、ゲートが黙って緩む種類の変更なので機械検査する。
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { SUITES, decideRecording, parseServerTreeHash } from '../run_test_suite.mjs'
import { buildVersionInfo, formatBuildTime } from '../put_version_info.mjs'
import {
  REQUIRED_SAMPLE_ENTRIES,
  certificateFingerprintLine,
  expectedArtifacts,
  findApksigner,
  isDebugSigned,
  missingSampleEntries,
  parse7zaListingPaths,
} from '../verify_release_artifacts.mjs'

describe('run_test_suite: 記録の条件', () => {
  const allowed = SUITES.test_server.allowed

  test('成功・非CI・許可された引数だけなら記録する', () => {
    expect(decideRecording({ status: 0, signal: null, ci: false, args: [], allowed })).toEqual({ record: true })
    expect(decideRecording({ status: 0, signal: null, ci: false, args: ['-p', '2'], allowed })).toEqual({ record: true })
  })

  test('失敗したスイートは記録しない（exit 非0・シグナル中断）', () => {
    expect(decideRecording({ status: 1, signal: null, ci: false, args: [], allowed }).record).toBe(false)
    expect(decideRecording({ status: null, signal: 'SIGTERM', ci: false, args: [], allowed }).record).toBe(false)
  })

  test('CI 環境では記録しない（CI の記録は誰も読まない）', () => {
    const decision = decideRecording({ status: 0, signal: null, ci: true, args: [], allowed })
    expect(decision.record).toBe(false)
    expect(decision.reason).toContain('CI')
  })

  test('絞り込み引数（-run / spec 名）は全件通ったと言えないので記録しない', () => {
    const decision = decideRecording({ status: 0, signal: null, ci: false, args: ['-run', 'TestFoo'], allowed })
    expect(decision.record).toBe(false)
    expect(decision.reason).toContain('-run')
    expect(decideRecording({ status: 0, signal: null, ci: false, args: ['foo.spec.ts'], allowed: SUITES.test_client_e2e.allowed }).record).toBe(false)
  })

  test('引数が検査内容を置き換えるスイート（verify_docs 等）は引数が1つでもあれば記録しない', () => {
    for (const suite of ['verify_docs', 'test_plugins', 'test_android', 'test_wear_os']) {
      expect(decideRecording({ status: 0, signal: null, ci: false, args: ['--list'], allowed: SUITES[suite].allowed }).record, suite).toBe(false)
    }
  })

  test('gkill_server version の出力から tree の行だけを読む', () => {
    const stdout = ['gkill_server:\t1.1.9-dev', 'build_time:\t2026-09-16 07:00:00 +0900 JST', 'hash:\t\tabcdef0', 'tree:\t\t0123456789abcdef', ''].join('\n')
    expect(parseServerTreeHash(stdout)).toBe('0123456789abcdef')
    expect(parseServerTreeHash('gkill_server:\t1.1.9-dev\n')).toBeNull()
    expect(parseServerTreeHash('')).toBeNull()
    expect(parseServerTreeHash(undefined)).toBeNull()
  })
})

describe('put_version_info: version.json の中身', () => {
  test('build_time はローカル時刻＋タイムゾーンの ISO 形式', () => {
    const formatted = formatBuildTime(new Date(2026, 2, 19, 10, 30, 0))
    expect(formatted).toMatch(/^2026-03-19T10:30:00[+-]\d{2}:\d{2}$/)
  })

  test('git から HEAD と作業ツリーの tree hash を写す', () => {
    const info = buildVersionInfo({
      version: '1.2.3',
      now: new Date(2026, 0, 2, 3, 4, 5),
      readHead: () => 'a'.repeat(40),
      readWorkingTree: () => 'b'.repeat(40),
    })
    expect(info).toEqual({
      commit_hash: 'a'.repeat(40),
      build_time: formatBuildTime(new Date(2026, 0, 2, 3, 4, 5)),
      version: '1.2.3',
      tree_hash: 'b'.repeat(40),
    })
  })

  test('git が読めなければ環境変数の SHA か unknown で埋めてビルドを止めない', () => {
    const warnings = []
    const info = buildVersionInfo({
      version: undefined,
      env: { GITHUB_SHA: 'c'.repeat(40) },
      readHead: () => { throw new Error('not a git repository') },
      readWorkingTree: () => { throw new Error('not a git repository') },
      warn: (message) => warnings.push(message),
    })
    expect(info.commit_hash).toBe('c'.repeat(40))
    expect(info.tree_hash).toBe('unknown')
    expect(info.version).toBe('unknown')
    expect(warnings).toHaveLength(1)
    expect(warnings[0]).toContain('tree_hash')

    const noEnv = buildVersionInfo({ version: '1.0.0', env: {}, readHead: () => { throw new Error('x') }, readWorkingTree: () => 'd'.repeat(40) })
    expect(noEnv.commit_hash).toBe('unknown')
    expect(noEnv.tree_hash).toBe('d'.repeat(40))
  })
})

describe('verify_release_artifacts: 判定', () => {
  test('成果物はゲートの記録 + Go 7種 + APK 3本 + サンプルデータの 12 件', () => {
    const names = expectedArtifacts('1.2.3')
    expect(names).toHaveLength(12)
    expect(names[0]).toBe('RELEASE_ATTESTATION_1.2.3.json')
    expect(names.filter((name) => name.endsWith('.apk'))).toEqual(['gkill_1.2.3.apk', 'gkill_wear_companion_1.2.3.apk', 'gkill_wear_watch_1.2.3.apk'])
    expect(names).toContain('gkill_sample_data_1.2.3.zip')
    expect(new Set(names).size).toBe(names.length)
  })

  test('7za -slt の一覧から Path 行だけを拾い、区切りを / に揃える', () => {
    const listing = [
      '7-Zip (a) 23.01',
      '',
      'Path = gkill_sample_data\\README.txt',
      'Size = 10',
      'Path = gkill_sample_data/configs/account.db',
      'Attributes = A',
      '',
    ].join('\n')
    const paths = parse7zaListingPaths(listing)
    expect([...paths]).toEqual(['gkill_sample_data/README.txt', 'gkill_sample_data/configs/account.db'])
  })

  test('必須エントリの欠けを列挙する', () => {
    const complete = new Set(REQUIRED_SAMPLE_ENTRIES)
    expect(missingSampleEntries(complete)).toEqual([])
    complete.delete('gkill_sample_data/gkill_server.exe')
    expect(missingSampleEntries(complete)).toEqual(['gkill_sample_data/gkill_server.exe'])
    expect(REQUIRED_SAMPLE_ENTRIES).toContain('gkill_sample_data/LAUNCH_GKILL_SAMPLE_DATA.bat')
  })

  test('debug 証明書で署名された APK を見分け、指紋の行を拾う', () => {
    const debugOut = 'Signer #1 certificate DN: CN=Android Debug, O=Android, C=US\nSigner #1 certificate SHA-256 digest: 00aa\n'
    const releaseOut = 'Signer #1 certificate DN: CN=gkill\nSigner #1 certificate SHA-256 digest: 11bb\n'
    expect(isDebugSigned(debugOut)).toBe(true)
    expect(isDebugSigned(releaseOut)).toBe(false)
    expect(certificateFingerprintLine(releaseOut)).toBe('Signer #1 certificate SHA-256 digest: 11bb')
    expect(certificateFingerprintLine('no certs')).toBeNull()
  })

  test('apksigner は SDK の build-tools の最新版から探し、無ければ null', () => {
    const sdk = fs.mkdtempSync(path.join(os.tmpdir(), 'gkill-sdk-test-'))
    try {
      expect(findApksigner({})).toBeNull()
      expect(findApksigner({ ANDROID_HOME: sdk })).toBeNull()
      fs.mkdirSync(path.join(sdk, 'build-tools', '34.0.0'), { recursive: true })
      fs.mkdirSync(path.join(sdk, 'build-tools', '35.0.1'), { recursive: true })
      fs.writeFileSync(path.join(sdk, 'build-tools', '34.0.0', 'apksigner.bat'), '')
      fs.writeFileSync(path.join(sdk, 'build-tools', '35.0.1', 'apksigner'), '')
      expect(findApksigner({ ANDROID_HOME: sdk })).toBe(path.join(sdk, 'build-tools', '35.0.1', 'apksigner'))
      expect(findApksigner({ ANDROID_SDK_ROOT: sdk })).toBe(path.join(sdk, 'build-tools', '35.0.1', 'apksigner'))
    } finally {
      fs.rmSync(sdk, { recursive: true, force: true })
    }
  })
})
