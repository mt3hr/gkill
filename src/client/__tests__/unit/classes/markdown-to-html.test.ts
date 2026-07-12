import { describe, it, expect } from 'vitest'
import DOMPurify from 'dompurify'
import { is_markdown_file_name, markdown_to_safe_html, truncate_markdown } from '@/classes/markdown-to-html'

const BASE_URL = '/files/rep1/docs/note.md'

describe('is_markdown_file_name', () => {
    it('md / markdown をMarkdownと判定する', () => {
        expect(is_markdown_file_name('note.md')).toBe(true)
        expect(is_markdown_file_name('note.markdown')).toBe(true)
        expect(is_markdown_file_name('NOTE.MD')).toBe(true)
    })

    it('それ以外はMarkdownと判定しない', () => {
        expect(is_markdown_file_name('note.txt')).toBe(false)
        expect(is_markdown_file_name('photo.jpg')).toBe(false)
        expect(is_markdown_file_name('md')).toBe(false)
    })
})

describe('markdown_to_safe_html', () => {
    // DOMPurifyはwindowが無いとisSupported=falseになり、
    // sanitizeが入力をそのまま返す (=サニタイズされない) ため明示的に確認する
    it('DOMPurifyが有効な環境である', () => {
        expect(DOMPurify.isSupported).toBe(true)
    })

    it('見出し・リスト・コードブロック・テーブルをHTML化する', async () => {
        const md = [
            '# 見出し',
            '',
            '- item1',
            '- item2',
            '',
            '```go',
            'func main() {}',
            '```',
            '',
            '| a | b |',
            '| --- | --- |',
            '| 1 | 2 |',
        ].join('\n')

        const html = await markdown_to_safe_html(md, BASE_URL)

        expect(html).toContain('<h1>見出し</h1>')
        expect(html).toContain('<li>item1</li>')
        expect(html).toContain('<pre>')
        expect(html).toContain('func main()')
        expect(html).toContain('<table>')
        expect(html).toContain('<td>1</td>')
    })

    it('scriptタグとonerror属性を除去する', async () => {
        const md = [
            '<script>alert(1)</script>',
            '',
            '<img src=x onerror="alert(1)">',
        ].join('\n')

        const html = await markdown_to_safe_html(md, BASE_URL)

        expect(html).not.toContain('<script')
        expect(html).not.toContain('onerror')
        expect(html).not.toContain('alert(1)')
    })

    it('javascript: のリンクを除去する', async () => {
        const html = await markdown_to_safe_html('[click](javascript:alert(1))', BASE_URL)
        expect(html).not.toContain('javascript:')
    })

    it('相対パスの画像をfile_url基準で解決する', async () => {
        const html = await markdown_to_safe_html('![alt](img/a.png)', BASE_URL)
        expect(html).toContain('/files/rep1/docs/img/a.png')
    })

    it('相対パスのリンクをfile_url基準で解決する', async () => {
        const html = await markdown_to_safe_html('[other](./other.md)', BASE_URL)
        expect(html).toContain('/files/rep1/docs/other.md')
    })

    it('絶対URLとページ内アンカーは書き換えない', async () => {
        const html = await markdown_to_safe_html('[ext](https://example.com/x) [anc](#sec)', BASE_URL)
        expect(html).toContain('href="https://example.com/x"')
        expect(html).toContain('href="#sec"')
    })

    it('リンクに target=_blank と rel を付与する', async () => {
        const html = await markdown_to_safe_html('[ext](https://example.com/)', BASE_URL)
        expect(html).toContain('target="_blank"')
        expect(html).toContain('rel="noopener noreferrer"')
    })

    it('相対MDリンクに data-gkill-md-link を付与する (値は元の相対パス)', async () => {
        const html = await markdown_to_safe_html('[index](index.md)', BASE_URL)
        expect(html).toContain('data-gkill-md-link="index.md"')
    })

    it('親ディレクトリへの相対MDリンクもマークする', async () => {
        const html = await markdown_to_safe_html('[up](../index.md)', BASE_URL)
        expect(html).toContain('data-gkill-md-link="../index.md"')
    })

    it('日本語ファイル名の相対MDリンクもマークする (markedがパーセントエンコードした値)', async () => {
        const html = await markdown_to_safe_html('[哲学](記録哲学.md)', BASE_URL)
        expect(html).toContain(`data-gkill-md-link="${encodeURIComponent('記録哲学')}.md"`)
    })

    it('絶対URL・ページ内アンカー・非MDの相対リンクはマークしない', async () => {
        const html = await markdown_to_safe_html(
            '[ext](https://example.com/x.md) [anc](#sec) [img](photo.jpg) [txt](note.txt)',
            BASE_URL,
        )
        expect(html).not.toContain('data-gkill-md-link')
    })

    it('/files/ 配下でないbase_urlの相対MDリンクはマークしない', async () => {
        const html = await markdown_to_safe_html('[other](other.md)', '/zip_cache/rep1/hash/note.md')
        expect(html).not.toContain('data-gkill-md-link')
    })
})

describe('truncate_markdown', () => {
    it('上限以下ならそのまま返す', () => {
        expect(truncate_markdown('abc', 10)).toBe('abc')
    })

    it('行境界で切り詰める', () => {
        const result = truncate_markdown('1234567890\nabcdefghij\n', 15)
        expect(result).toBe('1234567890\n\n…')
    })

    it('閉じられていないコードフェンスを閉じる', () => {
        const md = '```go\nfunc main() {}\nvery long line here\n```\n'
        const result = truncate_markdown(md, 20)
        const fence_count = result.split('\n').filter((line) => line.trimStart().startsWith('```')).length
        expect(fence_count % 2).toBe(0)
    })
})
