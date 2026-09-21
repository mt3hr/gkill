// verify_docs.mjs の解析部の検証。
//
// 件数の突き合わせは実ツリーを読むので、ここでは「正規表現と composeTools の読み方」だけを
// 合成のソースで固定する。ここが空振りしても verify_docs は「件数 0 が資料と合わない」でしか
// 落ちず、資料側を 0 に直されると気付けない。
import { describe, expect, test } from 'vitest'
import { MCP_TEST_RE, countComposedToolNames } from '../verify_docs.mjs'

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
})
