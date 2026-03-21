import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'

describe('KFTLTemplateElementData', () => {
  test('can be instantiated', () => {
    const elem = new KFTLTemplateElementData()
    expect(elem).toBeInstanceOf(KFTLTemplateElementData)
  })

  describe('default field values', () => {
    let elem: KFTLTemplateElementData

    beforeEach(() => {
      elem = new KFTLTemplateElementData()
    })

    test('name defaults to empty string', () => {
      expect(elem.name).toBe('')
    })

    test('id defaults to empty string', () => {
      expect(elem.id).toBe('')
    })

    test('title defaults to empty string', () => {
      expect(elem.title).toBe('')
    })

    test('template defaults to empty string', () => {
      expect(elem.template).toBe('')
    })

    test('children defaults to null', () => {
      expect(elem.children).toBeNull()
    })

    test('key defaults to empty string', () => {
      expect(elem.key).toBe('')
    })

    test('is_checked defaults to false', () => {
      expect(elem.is_checked).toBe(false)
    })

    test('indeterminate defaults to false', () => {
      expect(elem.indeterminate).toBe(false)
    })

    test('is_dir defaults to false', () => {
      expect(elem.is_dir).toBe(false)
    })

    test('is_open_default defaults to false', () => {
      expect(elem.is_open_default).toBe(false)
    })
  })

  describe('field assignment', () => {
    test('can set name and template', () => {
      const elem = new KFTLTemplateElementData()
      elem.name = 'daily-log'
      elem.template = 'tag テスト\nmemo '
      expect(elem.name).toBe('daily-log')
      expect(elem.template).toBe('tag テスト\nmemo ')
    })

    test('can set children array', () => {
      const parent = new KFTLTemplateElementData()
      const child = new KFTLTemplateElementData()
      child.name = 'child-template'
      parent.children = [child]
      expect(parent.children).toHaveLength(1)
      expect(parent.children[0].name).toBe('child-template')
    })

    test('can set is_dir and is_open_default', () => {
      const elem = new KFTLTemplateElementData()
      elem.is_dir = true
      elem.is_open_default = true
      expect(elem.is_dir).toBe(true)
      expect(elem.is_open_default).toBe(true)
    })
  })

  describe('FoldableStructModel interface', () => {
    test('has all FoldableStructModel required fields', () => {
      const elem = new KFTLTemplateElementData()
      // FoldableStructModel requires: name, id, children, key, is_checked, indeterminate, is_dir
      expect('name' in elem).toBe(true)
      expect('id' in elem).toBe(true)
      expect('children' in elem).toBe(true)
      expect('key' in elem).toBe(true)
      expect('is_checked' in elem).toBe(true)
      expect('indeterminate' in elem).toBe(true)
      expect('is_dir' in elem).toBe(true)
    })
  })
})
