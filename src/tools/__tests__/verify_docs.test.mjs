// verify_docs.mjs の解析部の検証。
//
// 件数の突き合わせは実ツリーを読むので、ここでは「正規表現と composeTools の読み方」だけを
// 合成のソースで固定する。ここが空振りしても verify_docs は「件数 0 が資料と合わない」でしか
// 落ちず、資料側を 0 に直されると気付けない。
import { describe, expect, test } from 'vitest'
import {
  BOUNDARY_TABLE_IDS,
  MCP_TEST_RE,
  cellKeys,
  checkBoundaryDoc,
  checkContractRows,
  checkReverseDocIndex,
  checkWebRouteRows,
  compareKeySets,
  composedToolNames,
  countComposedToolNames,
  expectedWebRouteCells,
  extractBoundaryBlock,
  parseGkillApi,
  parseGoHandlers,
  parseRouteTable,
} from '../verify_docs.mjs'

describe('MCP_TEST_RE', () => {
  test('t.Run(...) の行だけを数える（コメント・別名・ネストの深さは問わない）', () => {
    const src = [
      'func TestX(t *testing.T) {',
      '\tt.Run("a", func(t *testing.T) {})',
      '\t\tt.Run("nested", func(t *testing.T) {})',
      '\t// t.Run("commented", nil)',
      '\tother.Run("not a subtest")',
      '\tt.Run(',
      '}',
    ].join('\n')
    expect(src.match(MCP_TEST_RE)).toHaveLength(3)
    expect('// t.Run("x")'.match(MCP_TEST_RE)).toBeNull()
  })
})

describe('countComposedToolNames', () => {
  const modules = {
    ReadTools: ['gkill_get_kyous', 'gkill_get_status', 'gkill_get_idf_file'],
    WriteTools: ['gkill_add_kmemo', 'gkill_delete_kyou'],
    PluginTools: ['gkill_get_plugin_list'],
  }
  const namesInModule = (name) => modules[name] ?? null

  test('モジュールの連結をそのまま数える（重複は1つ）', () => {
    const src = [
      'func newServer() {',
      '\treturn composeTools(ReadTools, PluginTools, ReadTools)',
      '}',
    ].join('\n')
    expect(countComposedToolNames(src, namesInModule)).toBe(4)
  })

  test('filterTools(Mod, set) は同じソースの newNameSet(...) の名前集合で絞る', () => {
    const src = [
      'var writeSideReadTools = newNameSet(',
      '\t"gkill_get_kyous",',
      '\t"gkill_get_status",',
      ')',
      'func newServer() {',
      '\treturn composeTools(filterTools(ReadTools, writeSideReadTools), WriteTools, PluginTools)',
      '}',
    ].join('\n')
    expect(countComposedToolNames(src, namesInModule)).toBe(5)
  })

  test('知らないモジュール名は無視し、composeTools の定義行は読まない', () => {
    const definition = 'func composeTools(lists ...[]*jsonobj.Object) []*jsonobj.Object {\n\treturn nil\n}\n'
    expect(countComposedToolNames(definition, namesInModule)).toBe(0)
    expect(countComposedToolNames('\treturn composeTools(Unknown, WriteTools)\n', namesInModule)).toBe(2)
  })

  test('composedToolNames は同じ読み方で名前の集合を返す', () => {
    const names = composedToolNames('\treturn composeTools(WriteTools, PluginTools)\n', namesInModule)
    expect([...names].sort()).toEqual(['gkill_add_kmemo', 'gkill_delete_kyou', 'gkill_get_plugin_list'])
  })
})

// ─── 境界対応表（cross-boundary-map.md） ───────────────────────────────
// 合成のコードと資料で固定する。実ツリーとの突き合わせは verify_docs 本体が走らせる。

const ROUTE_SRC = [
  'func (g *GkillServerAPI) apiRoutes() []apiRoute {',
  '\treturn []apiRoute{',
  '\t\t{Path: "/api/add_kmemo", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleAddKmemo},',
  '\t\t{Path: "/api/add_user", Method: "POST", Auth: authSession, Body: bodyNone, Handler: g.HandleAddAccount},',
  '\t\t{Path: "/api/get_kyous_mcp", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetKyousMCP},',
  '\t\t{Path: "/api/urlog_bookmarklet", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleURLogBookmarklet},',
  '\t}',
  '}',
].join('\r\n')

const HANDLER_FILES = [
  {
    name: 'handle_add_kmemo.go',
    src: [
      'func (g *GkillServerAPI) HandleAddKmemo(w http.ResponseWriter, r *http.Request) {',
      '\trequest := &req_res.AddKmemoRequest{}',
      '\tresponse := &req_res.AddKmemoResponse{}',
      '\tg.UsecaseCtx.AddKmemo(r.Context(), request.Kmemo)',
      '}',
      'func helper() {',
      '\tg.UsecaseCtx.NotMine(nil)',
      '}',
    ].join('\r\n'),
  },
  {
    name: 'handle_add_account.go',
    src: [
      'func (g *GkillServerAPI) HandleAddAccount(w http.ResponseWriter, r *http.Request) {',
      '\trequest := &req_res.AddAccountRequest{}',
      '\tresponse := &req_res.AddAccountResponse{}',
      '\tg.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(ctx, a)',
      '}',
    ].join('\n'),
  },
  {
    name: 'handle_get_kyous_mcp.go',
    src: [
      'func (g *GkillServerAPI) HandleGetKyousMCP(w http.ResponseWriter, r *http.Request) {',
      '\trequest := &req_res.GetKyousMCPRequest{}',
      '\tresponse := &req_res.GetKyousMCPResponse{}',
      '\tg.FindFilter.FindKyous(ctx)',
      '}',
    ].join('\n'),
  },
  {
    name: 'handle_urlog_bookmarklet.go',
    src: [
      'func (g *GkillServerAPI) HandleURLogBookmarklet(w http.ResponseWriter, r *http.Request) {',
      '\trequest := &req_res.URLogBookmarkletRequest{}',
      '\tresponse := &req_res.URLogBookmarkletResponse{}',
      '}',
    ].join('\n'),
  },
]

const GKILL_API_SRC = [
  'export class GkillAPI {',
  '        add_kmemo_address: string',
  '        protected constructor() {',
  '                this.add_kmemo_address = "/api/add_kmemo"',
  '                this.add_account_address = "/api/add_user"',
  '                this.urlog_bookmarklet_address = "/api/urlog_bookmarklet"',
  '        }',
  '        async add_kmemo(req: AddKmemoRequest): Promise<AddKmemoResponse> {',
  '                const res = await this.gkill_fetch(this.add_kmemo_address, {',
  '                })',
  '        }',
  '        async add_user_legacy(req: AddAccountRequest): Promise<AddAccountResponse> {',
  '                const res = await this.gkill_fetch(this.add_account_address, {',
  '                })',
  '        }',
  '}',
  'export class GkillAPIForSharedKyou extends GkillAPI {',
  '        async add_kmemo(_req: AddKmemoRequest): Promise<AddKmemoResponse> {',
  '                const res = await this.gkill_fetch(this.shared_only_address, {',
  '        }',
  '}',
].join('\r\n')

function boundaryFacts(overrides = {}) {
  return {
    routes: parseRouteTable(ROUTE_SRC),
    handlers: parseGoHandlers(HANDLER_FILES),
    api: parseGkillApi(GKILL_API_SRC),
    otherCallers: {
      SW: new Set(['/api/add_kmemo']),
      MCP: new Set(['/api/add_kmemo', '/api/get_kyous_mcp']),
      CLI: new Set(),
      Wear: new Set(),
      配信HTML: new Set(['/api/urlog_bookmarklet']),
    },
    servePrefixes: new Set(['/files/', '/rykv', '/']),
    routerPaths: new Set(['/', '/rykv']),
    mcpTools: new Map([['gkill_add_kmemo', new Set(['write', 'readwrite'])]]),
    // 合成のルート表に載っているパスで組む（MCP が叩くパスはルート表にあることも検査されるので）
    mcpEntities: [{ dataType: 'kmemo', get: '/api/get_kyous_mcp', update: '/api/add_kmemo' }],
    mcpHttpRoutes: new Set(['/mcp']),
    wearWatchPaths: new Set(['/gkill/submit']),
    wearCompanionPaths: new Set(['/gkill/submit']),
    wearApiFunctions: new Set(['submitKFTLText']),
    pluginHostCommands: new Set(['ping']),
    pluginSdkCommands: new Set(['ping']),
    pluginHostFlags: new Set(['gkill-plugin-dir']),
    pluginSdkFlags: new Set(['gkill-plugin-dir']),
    postMessageKeys: new Set(['gkill_theme']),
    reqResUnpaired: new Set(['account.go']),
    resolveSuffix: (s) => (s === 'a/close.go' ? ['src/a/close.go'] : s === 'b/Main.kt' ? ['src/b/Main.kt'] : []),
    readFile: (rel) => (rel === 'src/a/close.go' ? 'Access your record space at : %s' : 'Access your record space at : '),
    ...overrides,
  }
}

const GOOD_WEB_ROWS = [
  '| `/api/add_kmemo` | POST | authSessionRepos | `add_kmemo()` | SW MCP | `HandleAddKmemo` | handle_add_kmemo.go | `UsecaseCtx.AddKmemo` | AddKmemoRequest / AddKmemoResponse | = |',
  '| `/api/add_user` | POST | authSession | `add_user_legacy()` ※ `add_account_address` | — | `HandleAddAccount` | handle_add_account.go ※ | 直接: `ConfigDAOs.AccountDAO` | AddAccountRequest / AddAccountResponse | = |',
  '| `/api/get_kyous_mcp` | POST | authNone | — 非Web | MCP | `HandleGetKyousMCP` | handle_get_kyous_mcp.go | 直接: `FindFilter` | GetKyousMCPRequest / GetKyousMCPResponse | — |',
  '| `/api/urlog_bookmarklet` | POST | authNone | — アドレスのみ（Go が配る HTML が叩く） | 配信HTML | `HandleURLogBookmarklet` | handle_urlog_bookmarklet.go | 直接: `request` | URLogBookmarkletRequest / URLogBookmarkletResponse | — |',
]

function table(id, header, rows) {
  return [
    `<!-- BOUNDARY-TABLE:${id}:BEGIN -->`,
    header,
    '|' + header.split('|').slice(1, -1).map(() => '---').join('|') + '|',
    ...rows,
    `<!-- BOUNDARY-TABLE:${id}:END -->`,
  ].join('\n')
}

function webRows(lines) {
  return extractBoundaryBlock(table('web-routes', '| パス | M | 認証 | TS | 他経路 | Go | ファイル | 委譲先 | Go 型 | TS 型 |', lines), 'web-routes').rows
}

describe('parseRouteTable / parseGoHandlers / parseGkillApi', () => {
  test('ルート表の1行を読み、CRLF でも同じ結果になる', () => {
    const routes = parseRouteTable(ROUTE_SRC)
    expect(routes).toHaveLength(4)
    expect(routes[1]).toEqual({ path: '/api/add_user', method: 'POST', auth: 'authSession', body: 'bodyNone', handler: 'HandleAddAccount' })
    expect(parseRouteTable(ROUTE_SRC.replace(/\r\n/g, '\n'))).toEqual(routes)
  })

  test('ハンドラの本体は次の行頭 func の手前まで。後ろの関数の UsecaseCtx 呼び出しを混ぜない', () => {
    const handlers = parseGoHandlers(HANDLER_FILES)
    expect(handlers.get('HandleAddKmemo')).toMatchObject({ file: 'handle_add_kmemo.go', usecases: ['AddKmemo'], req: 'AddKmemoRequest', res: 'AddKmemoResponse' })
    expect(handlers.get('HandleAddAccount').usecases).toEqual([])
  })

  test('要求/応答の型が無いハンドラは null', () => {
    const handlers = parseGoHandlers([{ name: 'handle_x.go', src: 'func (g *GkillServerAPI) HandleX(w http.ResponseWriter, r *http.Request) {\n}\n' }])
    expect(handlers.get('HandleX')).toMatchObject({ req: null, res: null })
  })

  test('GkillAPI だけを読み、共有ページ用の子クラスの上書きを混ぜない', () => {
    const api = parseGkillApi(GKILL_API_SRC)
    expect(api.addressByPath.get('/api/add_kmemo')).toBe('add_kmemo')
    expect(api.methodByField.get('add_kmemo')).toEqual({ name: 'add_kmemo', req: 'AddKmemoRequest', res: 'AddKmemoResponse' })
    expect(api.methodByField.has('shared_only')).toBe(false)
  })

  test('フィールド名とメソッド名がずれていても gkill_fetch で対にし、アドレスだけのものはメソッド無し', () => {
    const api = parseGkillApi(GKILL_API_SRC)
    expect(api.addressByPath.get('/api/add_user')).toBe('add_account')
    expect(api.methodByField.get('add_account').name).toBe('add_user_legacy')
    expect(api.addressByPath.get('/api/urlog_bookmarklet')).toBe('urlog_bookmarklet')
    expect(api.methodByField.has('urlog_bookmarklet')).toBe(false)
  })
})

describe('extractBoundaryBlock / cellKeys', () => {
  test('見出しと区切りを飛ばしてデータ行を読み、CRLF でも資料の行番号が合う', () => {
    const doc = ['# t', '', table('cli-api', '| パス | 置き場所 |', ['| `/api/add_tag` | `add_tag.go` |', '注記', '| `/api/get_kyous` | x |'])].join('\r\n')
    const block = extractBoundaryBlock(doc, 'cli-api')
    expect(block.rows.map((r) => [r.cells[0], r.line])).toEqual([['`/api/add_tag`', 6], ['`/api/get_kyous`', 8]])
    expect(block.text).toContain('注記')
  })

  test('目印が無ければ null。1列目のバッククォートは全部キーになる', () => {
    expect(extractBoundaryBlock('| a | b |', 'cli-api')).toBeNull()
    expect(cellKeys('`/oauth/token` `/token`')).toEqual(['/oauth/token', '/token'])
    expect(cellKeys('—')).toEqual([])
  })
})

describe('checkWebRouteRows', () => {
  test('コードと一致すれば0件', () => {
    expect(checkWebRouteRows(webRows(GOOD_WEB_ROWS), boundaryFacts())).toEqual([])
  })

  test('欠けた行は貼れる期待行付きで報告し、余分な行と重複も報告する', () => {
    const rows = webRows([
      GOOD_WEB_ROWS[0],
      GOOD_WEB_ROWS[0],
      GOOD_WEB_ROWS[2],
      GOOD_WEB_ROWS[3],
      '| `/api/gone` | POST | authNone | — 非Web | — | `HandleGone` | handle_gone.go | 直接: `x` | — / — | — |',
    ])
    const errors = checkWebRouteRows(rows, boundaryFacts())
    expect(errors.some((e) => e.includes('/api/add_kmemo が重複'))).toBe(true)
    expect(errors.some((e) => e.includes('/api/gone はルート表に無い'))).toBe(true)
    const missing = errors.find((e) => e.includes('/api/add_user の行が無い'))
    expect(missing).toContain('| `/api/add_user` | POST | authSession | `add_user_legacy()` ※ `add_account_address` |')
    expect(missing).toContain('直接: （委譲先を書く）')
  })

  test('並びがルート表と違えば最初のずれを報告する', () => {
    const rows = webRows([GOOD_WEB_ROWS[1], GOOD_WEB_ROWS[0], GOOD_WEB_ROWS[2], GOOD_WEB_ROWS[3]])
    expect(checkWebRouteRows(rows, boundaryFacts())).toEqual([
      expect.stringContaining('並びがルート表と違う（最初のずれ: 資料 /api/add_user／ルート表 /api/add_kmemo）'),
    ])
  })

  test('※ の付け忘れも付けすぎも落とす', () => {
    const forgot = GOOD_WEB_ROWS[1].replace('handle_add_account.go ※', 'handle_add_account.go')
    const extra = GOOD_WEB_ROWS[0].replace('handle_add_kmemo.go', 'handle_add_kmemo.go ※')
    const errors = checkWebRouteRows(webRows([extra, forgot, GOOD_WEB_ROWS[2], GOOD_WEB_ROWS[3]]), boundaryFacts())
    expect(errors).toHaveLength(2)
    expect(errors[0]).toContain('/api/add_kmemo の実装ファイル列 → 資料「handle_add_kmemo.go ※」／期待「handle_add_kmemo.go」')
    expect(errors[1]).toContain('期待「handle_add_account.go ※」')
  })

  test('TS 列: 非Web・アドレスのみ（後ろは自由記述）・メソッド名の違いを見る', () => {
    const facts = boundaryFacts()
    const routes = facts.routes
    expect(expectedWebRouteCells(routes[2], facts)[3]).toBe('— 非Web')
    expect(expectedWebRouteCells(routes[3], facts)[3]).toBe('— アドレスのみ')
    const wrong = GOOD_WEB_ROWS[3].replace('— アドレスのみ（Go が配る HTML が叩く）', '— 非Web')
    expect(checkWebRouteRows(webRows([GOOD_WEB_ROWS[0], GOOD_WEB_ROWS[1], GOOD_WEB_ROWS[2], wrong]), facts))
      .toEqual([expect.stringContaining('/api/urlog_bookmarklet のTS（GkillAPI）列')])
  })

  test('委譲先: usecase は正規形と一致、直接はバッククォートの字面がハンドラ本体にあること', () => {
    const wrongUsecase = GOOD_WEB_ROWS[0].replace('`UsecaseCtx.AddKmemo`', '`UsecaseCtx.AddKmemo2`')
    const ghost = GOOD_WEB_ROWS[1].replace('`ConfigDAOs.AccountDAO`', '`ConfigDAOs.GhostDAO`')
    const noKey = GOOD_WEB_ROWS[2].replace('直接: `FindFilter`', '直接: 検索')
    const errors = checkWebRouteRows(webRows([wrongUsecase, ghost, noKey, GOOD_WEB_ROWS[3]]), boundaryFacts())
    expect(errors).toHaveLength(3)
    expect(errors[0]).toContain('/api/add_kmemo の委譲先列')
    expect(errors[1]).toContain('字面「ConfigDAOs.GhostDAO」が HandleAddAccount の本体に無い')
    expect(errors[2]).toContain('バッククォートの字面が無い')
  })

  test('他経路は SW MCP CLI Wear 配信HTML の順で、無ければ —', () => {
    const facts = boundaryFacts()
    expect(expectedWebRouteCells(facts.routes[0], facts)[4]).toBe('SW MCP')
    expect(expectedWebRouteCells(facts.routes[1], facts)[4]).toBe('—')
  })
})

describe('compareKeySets / checkContractRows / checkReverseDocIndex', () => {
  test('資料にだけある・コードにだけある・資料側の重複を報告する', () => {
    const errors = compareKeySets('plugin-commands', ['ping', 'ping', 'gone'], new Set(['ping', 'close']), 'sdk.go')
    expect(errors).toEqual([
      '境界対応表: plugin-commands: ping が重複',
      '境界対応表: plugin-commands: 資料にだけある gone（sdk.go に無い。消えたか改名された）',
      '境界対応表: plugin-commands: 資料に無い close（sdk.go にある。表へ1行足すこと）',
    ])
  })

  test('契約の字面が欠けたファイルを名指しし、字面の無い行（—）は飛ばす', () => {
    const doc = table('contracts', '| 字面 | 契約 | 置き場所 | テスト |', [
      '| `Access your record space at :` | 起動行 | `a/close.go` `b/Main.kt` | x |',
      '| `ERR000002` | 認証切れ | `a/close.go` `b/Main.kt` | x |',
      '| — | i18n | 何か | x |',
    ])
    const rows = extractBoundaryBlock(doc, 'contracts').rows
    const facts = boundaryFacts()
    const errors = checkContractRows(rows, facts.resolveSuffix, facts.readFile)
    expect(errors).toHaveLength(2)
    expect(errors[0]).toContain('「ERR000002」が src/a/close.go に無い')
    expect(errors[1]).toContain('「ERR000002」が src/b/Main.kt に無い')
  })

  test('契約の置き場所が0件・複数件に当たる、または片側しか書いていなければ落とす', () => {
    const doc = table('contracts', '| 字面 | 契約 | 置き場所 | テスト |', ['| `x` | y | `nowhere.go` | z |'])
    const rows = extractBoundaryBlock(doc, 'contracts').rows
    const errors = checkContractRows(rows, () => [], () => '')
    expect(errors[0]).toContain('置き場所が1件')
    expect(errors[1]).toContain('「nowhere.go」が0件のファイルに当たる')
  })

  test('reverse 資料は README の2リンク・依存図のノード・folder-structure のツリーに載っていること', () => {
    const readme = '1. [a.md](a.md)\n| [a.md](a.md) |\n  A[a.md<br/>x]\n  B[b.md<br/>y]\n| [b.md](b.md) |'
    const tree = '├── a.md\n'
    expect(checkReverseDocIndex(['README.md', 'a.md', 'b.md'], readme, tree)).toEqual([
      'reverse 資料の索引: documents/reverse/README.md の読む順と要約表に b.md へのリンクが1件（2件要る）',
      'reverse 資料の索引: documents/reverse/folder-structure.md のツリーに b.md が無い',
    ])
  })
})

describe('checkBoundaryDoc', () => {
  const tables = {
    'web-routes': ['| パス | M | 認証 | TS | 他経路 | Go | ファイル | 委譲先 | Go 型 | TS 型 |', GOOD_WEB_ROWS],
    'service-worker': ['| パス | 扱い |', ['| `/api/add_kmemo` | share-target |']],
    'server-prefix': ['| 接頭辞 | 登録 |', ['| `/files/` | a |', '| `/rykv` | b |', '| `/` | c |']],
    'mcp-tools': ['| ツール | サーバ | /api |', ['| `gkill_add_kmemo` | write readwrite | `/api/add_kmemo` |']],
    'mcp-entities': ['| data_type | get | update |', ['| `kmemo` | `/api/get_kyous_mcp` | `/api/add_kmemo` |']],
    'mcp-http': ['| 経路 | 役割 |', ['| `/mcp` | JSON-RPC。`/api/get_kyous_mcp` |']],
    'wear-datalayer': ['| パス | 向き |', ['| `/gkill/submit` | 時計→スマホ |']],
    'wear-api': ['| パス | 関数 |', []],
    'cli-api': ['| パス | サブコマンド |', []],
    'plugin-commands': ['| コマンド | 向き |', ['| `ping` | 本体→プラグイン |']],
    'plugin-flags': ['| フラグ | 意味 |', ['| `--gkill-plugin-dir` | 置き場所 |']],
    'post-message': ['| キー | 向き |', ['| `gkill_theme` | 親→iframe |']],
    contracts: ['| 字面 | 契約 | 置き場所 | テスト |', ['| — | i18n | x | y |']],
    'req-res-files': ['| ファイル | 側 |', ['| `account.go` | Go だけ |']],
  }
  const buildDoc = (skip) => BOUNDARY_TABLE_IDS.filter((id) => id !== skip)
    .map((id) => table(id, tables[id][0], tables[id][1])).join('\n\n')

  test('小さな資料と事実で全体が通る', () => {
    expect(checkBoundaryDoc(buildDoc(), boundaryFacts())).toEqual([])
  })

  test('目印の欠けた表・ルート表に無い呼び出し先・vue-router の path の載せ忘れを報告する', () => {
    const facts = boundaryFacts({
      routerPaths: new Set(['/', '/rykv', '/kftl']),
      otherCallers: { ...boundaryFacts().otherCallers, CLI: new Set(['/api/removed']) },
    })
    const errors = checkBoundaryDoc(buildDoc('post-message'), facts)
    expect(errors).toContain('境界対応表: documents/reverse/cross-boundary-map.md に <!-- BOUNDARY-TABLE:post-message:BEGIN --> 〜 END の表が無い')
    expect(errors).toContain('境界対応表: CLI が叩く /api/removed はルート表に無い（叩く側だけ残っている）')
    expect(errors.some((e) => e.includes('vue-router の /kftl が表に無い'))).toBe(true)
  })

  test('MCP の載るサーバの食い違いと、表と注記に出る /api パスの過不足を報告する', () => {
    const facts = boundaryFacts({
      mcpTools: new Map([['gkill_add_kmemo', new Set(['readwrite'])]]),
      otherCallers: { ...boundaryFacts().otherCallers, MCP: new Set(['/api/add_kmemo']) },
    })
    const errors = checkBoundaryDoc(buildDoc(), facts)
    expect(errors.some((e) => e.includes('gkill_add_kmemo の載るサーバ → 資料「write readwrite」／期待「readwrite」'))).toBe(true)
    expect(errors.some((e) => e.includes('資料にだけある /api/get_kyous_mcp'))).toBe(true)
  })

  test('時計とスマホの Data Layer パスは資料と両側の3者で照合する', () => {
    const facts = boundaryFacts({ wearCompanionPaths: new Set(['/gkill/submit', '/gkill/extra']) })
    expect(checkBoundaryDoc(buildDoc(), facts)).toEqual([
      '境界対応表: wear-datalayer（スマホ）: 資料に無い /gkill/extra（phone_companion の WearRequestHandler.kt にある。表へ1行足すこと）',
    ])
  })
})
