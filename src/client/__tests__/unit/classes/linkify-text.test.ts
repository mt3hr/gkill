import { split_text_by_urls } from '@/classes/linkify-text'

describe('split_text_by_urls', () => {
  test('URLを含まないテキストは全体が1セグメントになる', () => {
    expect(split_text_by_urls('hello world')).toEqual([
      { is_url: false, text: 'hello world' },
    ])
  })

  test('空文字列は空配列を返す', () => {
    expect(split_text_by_urls('')).toEqual([])
  })

  test('URLだけのテキストはURLセグメント1つになる', () => {
    expect(split_text_by_urls('https://example.com')).toEqual([
      { is_url: true, text: 'https://example.com' },
    ])
  })

  test('文中のURLを抽出し前後をテキストセグメントとして残す', () => {
    expect(split_text_by_urls('前置き https://example.com/path?q=1 後書き')).toEqual([
      { is_url: false, text: '前置き ' },
      { is_url: true, text: 'https://example.com/path?q=1' },
      { is_url: false, text: ' 後書き' },
    ])
  })

  test('複数のURLをそれぞれ抽出する', () => {
    expect(split_text_by_urls('a http://a.example.com b https://b.example.com c')).toEqual([
      { is_url: false, text: 'a ' },
      { is_url: true, text: 'http://a.example.com' },
      { is_url: false, text: ' b ' },
      { is_url: true, text: 'https://b.example.com' },
      { is_url: false, text: ' c' },
    ])
  })

  test('改行をまたぐテキストのセグメントが保存される', () => {
    expect(split_text_by_urls('1行目\nhttps://example.com\n3行目')).toEqual([
      { is_url: false, text: '1行目\n' },
      { is_url: true, text: 'https://example.com' },
      { is_url: false, text: '\n3行目' },
    ])
  })

  test('http/https以外のスキームはリンク化しない', () => {
    expect(split_text_by_urls('ftp://example.com')).toEqual([
      { is_url: false, text: 'ftp://example.com' },
    ])
  })

  test('末尾のASCII約物をURLから外す', () => {
    expect(split_text_by_urls('see https://example.com/page.')).toEqual([
      { is_url: false, text: 'see ' },
      { is_url: true, text: 'https://example.com/page' },
      { is_url: false, text: '.' },
    ])
  })

  test('URL直後の和文の約物・本文はURLに含めない', () => {
    expect(split_text_by_urls('参考: https://example.com/page。詳細は後で')).toEqual([
      { is_url: false, text: '参考: ' },
      { is_url: true, text: 'https://example.com/page' },
      { is_url: false, text: '。詳細は後で' },
    ])
  })

  test('全角括弧で囲まれたURLを抽出できる', () => {
    expect(split_text_by_urls('（https://example.com）')).toEqual([
      { is_url: false, text: '（' },
      { is_url: true, text: 'https://example.com' },
      { is_url: false, text: '）' },
    ])
  })

  test('対応の取れた丸括弧はURLの一部として残す', () => {
    expect(split_text_by_urls('https://ja.example.org/wiki/foo_(bar)')).toEqual([
      { is_url: true, text: 'https://ja.example.org/wiki/foo_(bar)' },
    ])
  })

  test('外側の丸括弧は閉じ括弧だけURLから外す', () => {
    expect(split_text_by_urls('(see https://example.com/x_(y))')).toEqual([
      { is_url: false, text: '(see ' },
      { is_url: true, text: 'https://example.com/x_(y)' },
      { is_url: false, text: ')' },
    ])
  })

  test('パーセントエンコード済みURLはそのまま抽出する', () => {
    expect(split_text_by_urls('https://example.com/%E6%97%A5%E6%9C%AC?q=%20')).toEqual([
      { is_url: true, text: 'https://example.com/%E6%97%A5%E6%9C%AC?q=%20' },
    ])
  })
})
