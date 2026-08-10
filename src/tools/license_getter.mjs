#!/usr/bin/env node
// 依存パッケージのライセンス一覧 LICENSES_DEPENDENCE をリポジトリルートに生成する。
//
// カバー範囲（4セクション）:
//   1. Go Modules       — src/server + src/plugins 配下の全モジュール（推移依存含む、本文全文）
//   2. Node.js Modules  — ルート package-lock.json の dependencies（devDependencies は配布物に
//                         含まれないため対象外。本文全文）
//   3. Android (Gradle) — src/android の releaseRuntimeClasspath（推移依存、ライセンス名+URL）
//   4. Wear OS (Gradle) — src/wear_os の phone_companion + watch_app（同上、GAV で重複排除）
//
// Gradle はライセンス本文をローカルに持たないため、名前+URL のみ（POM から抽出、
// 無ければ親 POM を追跡）。テスト専用依存（junit / mockk / espresso 等）は配布物に
// 入らないので対象外。
//
// 使い方: npm run license_getter
//   --skip-gradle : Gradle 環境（JDK + Android SDK）が無い環境用。Android / Wear OS
//                   セクションを未収集の注記だけにして続行する。
//
// 前提: go が PATH にあること、`npm ci`（または npm i）実行済みであること、
//       Gradle 部は JDK + Android SDK（無ければ --skip-gradle）。
// リポジトリへの副作用なし（go mod tidy は実行しない。go.mod / go.sum は書き換わらない）。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..') // src/tools/ → リポジトリルート
const OUTPUT_FILE = path.join(ROOT, 'LICENSES_DEPENDENCE')
const MAX_BUFFER = 64 * 1024 * 1024 // go list -m -json all / gradle 出力対策

const skipGradle = process.argv.includes('--skip-gradle')
const warnings = []

function warn(msg) {
  warnings.push(msg)
  console.warn(`⚠️  ${msg}`)
}

function fail(msg) {
  console.error(`❌ ${msg}`)
  process.exit(1)
}

// 改行コードを LF に正規化（ライセンス本文ファイルは CRLF 混在のため。出力の決定性を保つ）
function normalizeNewlines(text) {
  return text.replace(/\r\n?/g, '\n')
}

// ─────────────────────────────────────────────────────────────
// 1. Go Modules
// ─────────────────────────────────────────────────────────────

// go.mod を持つディレクトリ = 独立モジュール（test_plugins.mjs の findModules と同形）
function findGoModuleDirs(dir) {
  const out = []
  if (!fs.existsSync(dir)) return out
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    const child = path.join(dir, entry.name)
    if (fs.existsSync(path.join(child, 'go.mod'))) {
      out.push(child)
      continue
    }
    out.push(...findGoModuleDirs(child))
  }
  return out
}

const GO_BIN = process.platform === 'win32' ? 'go.exe' : 'go'

function runGo(cwd, args) {
  return spawnSync(GO_BIN, args, { cwd, encoding: 'utf8', maxBuffer: MAX_BUFFER })
}

// `go list -m -json all` は JSON オブジェクトの連結を返す。
// トップレベルオブジェクトの閉じ括弧は必ず行頭の "}" 単独行なので、それで分割する。
function splitGoListJson(text) {
  const objects = []
  let buf = []
  for (const line of normalizeNewlines(text).split('\n')) {
    buf.push(line)
    if (line === '}') {
      objects.push(JSON.parse(buf.join('\n')))
      buf = []
    }
  }
  return objects
}

// モジュールディレクトリからライセンス/告知ファイルを収集（大文字小文字は無視）
const GO_LICENSE_FILE_RE = /^(license|licence|license\.txt|license\.md|licence\.txt|licence\.md|unlicense|copying|notice)$/i

function readGoLicenseFiles(dir) {
  const found = []
  for (const name of fs.readdirSync(dir)) {
    if (!GO_LICENSE_FILE_RE.test(name)) continue
    const p = path.join(dir, name)
    if (!fs.statSync(p).isFile()) continue
    found.push({ name, text: normalizeNewlines(fs.readFileSync(p, 'utf8')) })
  }
  found.sort((a, b) => a.name.localeCompare(b.name))
  return found
}

function collectGoLicenses() {
  const serverDir = path.join(ROOT, 'src', 'server')
  const moduleDirs = [serverDir, ...findGoModuleDirs(path.join(ROOT, 'src', 'plugins')).sort()]
  // key = `${Path}@${Version}` → { modPath, version, dir }
  const deps = new Map()

  for (const modDir of moduleDirs) {
    const rel = path.relative(ROOT, modDir).split(path.sep).join('/')
    console.log(`  go list -m -json all  (${rel})`)
    const res = runGo(modDir, ['list', '-m', '-json', 'all'])
    if (res.error || res.status !== 0) {
      const detail = res.error ? res.error.message : (res.stderr || '').trim()
      if (modDir === serverDir) {
        fail(`src/server の go list に失敗: ${detail}\n   go.mod / go.sum の不整合なら先に npm run go_mod を実行すること`)
      }
      // プラグイン（特に go.sum の無い gkill_example）の失敗は警告スキップ。
      // 依存は src/server の部分集合なので一覧の網羅性には影響しない。
      warn(`${rel} の go list に失敗（スキップ）: ${detail.split('\n')[0]}`)
      continue
    }
    for (const mod of splitGoListJson(res.stdout)) {
      const eff = mod.Replace ?? mod // replace は実効モジュール側を採用
      if (mod.Main) continue
      const effDir = eff.Dir ?? null
      // リポジトリ内 replace（プラグイン → src/server）は自プロジェクトなので対象外
      if (effDir && path.resolve(effDir).startsWith(ROOT + path.sep)) continue
      const key = `${eff.Path}@${eff.Version ?? ''}`
      if (deps.has(key)) continue
      deps.set(key, { modPath: eff.Path, version: eff.Version ?? '', dir: effDir, fromModule: modDir })
    }
  }

  const lines = ['', '=== [Go Modules] ===']
  const keys = [...deps.keys()].sort()
  let count = 0
  for (const key of keys) {
    const dep = deps.get(key)
    // モジュールキャッシュ未展開なら go mod download で取得（GOMODCACHE への書き込みのみ）
    if (!dep.dir || !fs.existsSync(dep.dir)) {
      const res = runGo(dep.fromModule, ['mod', 'download', '-json', `${dep.modPath}@${dep.version}`])
      if (res.status === 0) {
        try {
          dep.dir = JSON.parse(res.stdout).Dir ?? null
        } catch {
          dep.dir = null
        }
      }
      if (!dep.dir || !fs.existsSync(dep.dir)) {
        warn(`Go モジュールのキャッシュを取得できない: ${key}`)
        continue
      }
    }
    const header = dep.version ? `${dep.modPath} ${dep.version}` : dep.modPath
    lines.push('', `=== [${header}] ===`)
    const files = readGoLicenseFiles(dep.dir)
    if (files.length === 0) {
      lines.push('No license or notice file found.')
    } else {
      for (const f of files) {
        lines.push(`License file: ${f.name}`, '', f.text.trimEnd())
      }
    }
    count++
  }
  return { text: lines.join('\n'), count }
}

// ─────────────────────────────────────────────────────────────
// 2. Node.js Modules
// ─────────────────────────────────────────────────────────────

const NODE_LICENSE_FILE_RE = /^(licen[cs]e|copying|unlicense)/i

function normalizeRepoUrl(repo) {
  const url = typeof repo === 'string' ? repo : repo?.url
  if (!url) return null
  return url.replace(/^git\+/, '').replace(/\.git$/, '')
}

// lock の dev / devOptional フラグは入れ子の optional 依存に正しく伝播しない（npm の既知の癖。
// 例: lightningcss は devOptional だがそのプラットフォーム別バイナリは optional のみ）。
// そのためフラグではなく、ルート package.json の dependencies から node_modules 解決を
// 辿った到達可能集合を「本番依存」とする。
function collectProductionKeys(lock) {
  const byKey = lock.packages ?? {}
  const parentScope = (key) => {
    const idx = key.lastIndexOf('/node_modules/')
    if (idx !== -1) return key.slice(0, idx)
    return key.startsWith('node_modules/') ? '' : null
  }
  const resolveKey = (fromKey, name) => {
    let base = fromKey
    while (base !== null) {
      const cand = base ? `${base}/node_modules/${name}` : `node_modules/${name}`
      if (byKey[cand]) return cand
      base = parentScope(base)
    }
    return null
  }
  // peer は optional 指定（peerDependenciesMeta.optional）を除いて辿る。
  // vuetify → vite-plugin-vuetify / typescript のような「ビルド時にだけ意味を持つ optional peer」
  // まで辿ると、本番到達集合にビルドツールチェーン一式が混入するため。
  const depNamesOf = (e) => [
    ...Object.keys(e.dependencies ?? {}),
    ...Object.keys(e.optionalDependencies ?? {}),
    ...Object.keys(e.peerDependencies ?? {}).filter((n) => !(e.peerDependenciesMeta?.[n]?.optional)),
  ]
  const root = byKey[''] ?? {}
  const visited = new Set()
  const queue = [['', [...Object.keys(root.dependencies ?? {}), ...Object.keys(root.optionalDependencies ?? {})]]]
  while (queue.length > 0) {
    const [fromKey, depNames] = queue.pop()
    for (const name of depNames) {
      const key = resolveKey(fromKey, name)
      if (key === null) continue // optional / peer の未インストールは lock に無いことがある
      if (visited.has(key)) continue
      visited.add(key)
      queue.push([key, depNamesOf(byKey[key])])
    }
  }
  return visited
}

function collectNpmLicenses() {
  const lockPath = path.join(ROOT, 'package-lock.json')
  if (!fs.existsSync(lockPath)) fail('package-lock.json が見つからない')
  const lock = JSON.parse(fs.readFileSync(lockPath, 'utf8'))
  if (lock.lockfileVersion !== 3) {
    warn(`package-lock.json の lockfileVersion が想定外 (${lock.lockfileVersion}) — v3 前提で処理する`)
  }
  const productionKeys = collectProductionKeys(lock)

  const entries = new Map() // name@version → 出力ブロック
  for (const key of productionKeys) {
    const info = lock.packages[key]
    if (info.link === true) continue
    const name = key.slice(key.lastIndexOf('node_modules/') + 'node_modules/'.length)
    const id = `${name}@${info.version}`
    if (entries.has(id)) continue

    const dir = path.join(ROOT, ...key.split('/'))
    const block = ['', `=== [${id}] ===`]
    let licenseName = info.license ?? null
    let repository = null
    let dirExists = fs.existsSync(dir)

    if (!dirExists) {
      if (info.optional === true) {
        // 他プラットフォーム向け optional 依存は npm ci でも入らない。名前だけ出す。
        warn(`未インストールの optional 依存（本文なし）: ${id}`)
      } else {
        fail(`node_modules に ${id} が無い。npm ci を実行してから再実行すること`)
      }
    }

    let pkgJson = null
    if (dirExists) {
      try {
        pkgJson = JSON.parse(fs.readFileSync(path.join(dir, 'package.json'), 'utf8'))
      } catch {
        pkgJson = null
      }
    }
    if (!licenseName) {
      const lic = pkgJson?.license
      licenseName = typeof lic === 'string' ? lic : lic?.type ?? 'UNKNOWN'
    }
    repository = normalizeRepoUrl(pkgJson?.repository)

    block.push(`License: ${licenseName}`)
    if (repository) block.push(`Repository: ${repository}`)

    let bodyFound = false
    if (dirExists) {
      const names = fs.readdirSync(dir).filter((n) => NODE_LICENSE_FILE_RE.test(n)).sort()
      for (const n of names) {
        const p = path.join(dir, n)
        if (!fs.statSync(p).isFile()) continue
        block.push('', normalizeNewlines(fs.readFileSync(p, 'utf8')).trimEnd())
        bodyFound = true
      }
      const noticePath = path.join(dir, 'NOTICE')
      if (fs.existsSync(noticePath) && fs.statSync(noticePath).isFile()) {
        block.push('', `[NOTICE] (from ${id}):`, normalizeNewlines(fs.readFileSync(noticePath, 'utf8')).trimEnd())
      }
    }
    if (!bodyFound) block.push('', '(ライセンス本文が見つかりませんでした)')

    entries.set(id, block.join('\n'))
  }

  const ids = [...entries.keys()].sort()
  const lines = ['', '=== [Node.js Modules] ===', ...ids.map((id) => entries.get(id))]
  return { text: lines.join('\n'), count: ids.length }
}

// ─────────────────────────────────────────────────────────────
// 3. / 4. Gradle (Android / Wear OS)
// ─────────────────────────────────────────────────────────────

// resolutionResult から外部モジュールの GAV を列挙し、ArtifactResolutionQuery で POM を
// 明示ダウンロードしてローカルパスごと JSON に書き出す init script。
// （Gradle Module Metadata (.module) 運用では POM が自然にはキャッシュされないため、
//   クエリでの明示取得が必要。）
const GRADLE_INIT_SCRIPT = `
allprojects { p ->
  p.tasks.register("gkillLicenseReport") {
    doLast {
      def confName = p.findProperty("gkillConf") ?: "releaseRuntimeClasspath"
      def conf = p.configurations.findByName(confName)
      if (conf == null) return
      def ids = conf.incoming.resolutionResult.allComponents*.id
        .findAll { it instanceof org.gradle.api.artifacts.component.ModuleComponentIdentifier }
      def poms = [:]
      def q = p.dependencies.createArtifactResolutionQuery()
        .forComponents(ids)
        .withArtifacts(org.gradle.maven.MavenModule, org.gradle.maven.MavenPomArtifact)
        .execute()
      q.resolvedComponents.each { comp ->
        comp.getArtifacts(org.gradle.maven.MavenPomArtifact).each { a ->
          if (a instanceof org.gradle.api.artifacts.result.ResolvedArtifactResult) {
            poms["\${comp.id.group}:\${comp.id.module}:\${comp.id.version}"] = a.file.absolutePath
          }
        }
      }
      def out = new File(p.property("gkillOut"), p.name + ".json")
      out.text = groovy.json.JsonOutput.toJson(ids.collect {
        [group: it.group, name: it.module, version: it.version,
         pom: poms["\${it.group}:\${it.module}:\${it.version}"]]
      })
    }
  }
}
`

function decodeXmlEntities(s) {
  return s
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&amp;/g, '&')
}

function xmlTagText(xml, tag) {
  const m = xml.match(new RegExp(`<${tag}>([\\s\\S]*?)</${tag}>`))
  return m ? decodeXmlEntities(m[1].trim()) : null
}

// POM の <licenses> からライセンス名+URL を抽出する。
function parsePomLicenses(pomText) {
  const licensesBlock = pomText.match(/<licenses>([\s\S]*?)<\/licenses>/)
  if (!licensesBlock) return []
  const out = []
  for (const m of licensesBlock[1].matchAll(/<license>([\s\S]*?)<\/license>/g)) {
    out.push({ name: xmlTagText(m[1], 'name'), url: xmlTagText(m[1], 'url') })
  }
  return out.filter((l) => l.name || l.url)
}

function parsePomParent(pomText) {
  const parent = pomText.match(/<parent>([\s\S]*?)<\/parent>/)
  if (!parent) return null
  const g = xmlTagText(parent[1], 'groupId')
  const a = xmlTagText(parent[1], 'artifactId')
  const v = xmlTagText(parent[1], 'version')
  return g && a && v ? { group: g, artifact: a, version: v } : null
}

// Gradle キャッシュ (~/.gradle/caches/modules-2/files-2.1/) から POM を探す（親 POM 追跡用）
function findPomInCache(group, artifact, version) {
  const base = path.join(os.homedir(), '.gradle', 'caches', 'modules-2', 'files-2.1', group, artifact, version)
  if (!fs.existsSync(base)) return null
  for (const hashDir of fs.readdirSync(base)) {
    const p = path.join(base, hashDir, `${artifact}-${version}.pom`)
    if (fs.existsSync(p)) return p
  }
  return null
}

// POM からライセンスを解決。無ければ親 POM を辿る（深さ上限5）。
function resolveLicenses(pomPath, depth = 0) {
  if (!pomPath || depth > 5) return { licenses: [], fallbackUrl: null }
  const text = fs.readFileSync(pomPath, 'utf8')
  const licenses = parsePomLicenses(text)
  if (licenses.length > 0) return { licenses, fallbackUrl: null }
  const parent = parsePomParent(text)
  if (parent) {
    const parentPom = findPomInCache(parent.group, parent.artifact, parent.version)
    const fromParent = resolveLicenses(parentPom, depth + 1)
    if (fromParent.licenses.length > 0) return fromParent
  }
  const fallbackUrl = xmlTagText(text, 'url') ?? (text.match(/<scm>([\s\S]*?)<\/scm>/) ? xmlTagText(text.match(/<scm>([\s\S]*?)<\/scm>/)[1], 'url') : null)
  return { licenses: [], fallbackUrl }
}

function runGradleReport(projectDir, taskPaths, outDir, initScript) {
  const gradlewPath = path.join(projectDir, process.platform === 'win32' ? 'gradlew.bat' : 'gradlew')
  const rel = path.relative(ROOT, projectDir).split(path.sep).join('/')
  if (!fs.existsSync(gradlewPath)) {
    fail(`${rel}/gradlew が無い。npm run setup_wear_os_gradle で入れ直すこと（または --skip-gradle）`)
  }
  const gradlew = process.platform === 'win32' ? `"${gradlewPath}"` : gradlewPath
  const args = [
    ...taskPaths,
    '-I', `"${initScript}"`,
    `"-PgkillOut=${outDir}"`,
    '-PgkillConf=releaseRuntimeClasspath',
    '--no-configuration-cache', // 両プロジェクトとも configuration-cache=true のため必須
    '--quiet',
  ]
  console.log(`  gradlew ${taskPaths.join(' ')}  (${rel})`)
  // Windows の .bat は shell 経由でしか起動できない。args 配列 + shell だと DEP0190 が出るため、
  // 自前で引用符を付けたコマンド文字列1本にして渡す（パスに空白があっても壊れない）。
  const res = process.platform === 'win32'
    ? spawnSync(`${gradlew} ${args.join(' ')}`, { cwd: projectDir, shell: true, encoding: 'utf8', maxBuffer: MAX_BUFFER })
    : spawnSync(gradlew, args.map((a) => a.replace(/^"|"$/g, '')), { cwd: projectDir, encoding: 'utf8', maxBuffer: MAX_BUFFER })
  if (res.error || res.status !== 0) {
    const detail = res.error ? res.error.message : (res.stderr || res.stdout || '').trim().split('\n').slice(-15).join('\n')
    fail(`${rel} の Gradle 依存解決に失敗（JDK / Android SDK が無い環境では --skip-gradle を使う）:\n${detail}`)
  }
}

function collectGradleLicenses(sectionName, projectDir, taskPaths) {
  const lines = ['', `=== [${sectionName}] ===`]
  if (skipGradle) {
    lines.push('', '[!] Gradle 依存は未収集（--skip-gradle 指定）')
    return { text: lines.join('\n'), count: 0 }
  }

  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gkill-license-'))
  try {
    const initScript = path.join(tmpDir, 'license-report.init.gradle')
    fs.writeFileSync(initScript, GRADLE_INIT_SCRIPT)
    const outDir = path.join(tmpDir, 'out')
    fs.mkdirSync(outDir)
    runGradleReport(projectDir, taskPaths, outDir, initScript)

    const deps = new Map() // GAV → { pom }（複数プロジェクト間の重複排除）
    for (const jsonFile of fs.readdirSync(outDir).sort()) {
      for (const dep of JSON.parse(fs.readFileSync(path.join(outDir, jsonFile), 'utf8'))) {
        const gav = `${dep.group}:${dep.name}:${dep.version}`
        if (!deps.has(gav)) deps.set(gav, dep)
      }
    }
    if (deps.size === 0) {
      fail(`${sectionName}: 依存が1件も取得できなかった（init script の想定と Gradle の挙動がずれている可能性）`)
    }

    let count = 0
    for (const gav of [...deps.keys()].sort()) {
      const dep = deps.get(gav)
      lines.push('', `=== [${gav}] ===`)
      const pomPath = dep.pom ?? findPomInCache(dep.group, dep.name, dep.version)
      const { licenses, fallbackUrl } = pomPath ? resolveLicenses(pomPath) : { licenses: [], fallbackUrl: null }
      if (licenses.length > 0) {
        for (const l of licenses) {
          lines.push(`License: ${l.name ?? '不明'}`)
          if (l.url) lines.push(`License URL: ${l.url}`)
        }
      } else {
        lines.push('License: 不明（リポジトリURL参照）')
        if (fallbackUrl) lines.push(`Repository: ${fallbackUrl}`)
        warn(`${sectionName}: POM からライセンスを特定できない: ${gav}`)
      }
      count++
    }
    return { text: lines.join('\n'), count }
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true })
  }
}

// ─────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────

console.log('Go 依存ライセンスを取得中...')
const go = collectGoLicenses()

console.log('Node.js 依存ライセンスを取得中...')
const npm = collectNpmLicenses()

console.log('Android (Gradle) 依存ライセンスを取得中...')
const android = collectGradleLicenses('Android (Gradle) Modules', path.join(ROOT, 'src', 'android'), [':app:gkillLicenseReport'])

console.log('Wear OS (Gradle) 依存ライセンスを取得中...')
const wearOs = collectGradleLicenses('Wear OS (Gradle) Modules', path.join(ROOT, 'src', 'wear_os'), [':phone_companion:gkillLicenseReport', ':watch_app:gkillLicenseReport'])

// 全セクション成功後に一括書き出し（途中失敗時に既存ファイルを壊さない）
const output = ['=== 依存ライセンス一覧 ===', go.text, npm.text, android.text, wearOs.text].join('\n') + '\n'
fs.writeFileSync(OUTPUT_FILE, output) // UTF-8 / BOM なし / LF

if (warnings.length > 0) {
  console.warn(`\n⚠️  警告 ${warnings.length}件（上記参照）`)
}
console.log(`\n✅ ${path.relative(ROOT, OUTPUT_FILE)} を出力（Go ${go.count}件 / Node.js ${npm.count}件 / Android ${android.count}件 / Wear OS ${wearOs.count}件）`)
