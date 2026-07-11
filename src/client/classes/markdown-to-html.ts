'use strict'

const MARKDOWN_EXTENSIONS = new Set(['md', 'markdown'])

// スキーム付きURL (http:, data:, mailto: など)
const HAS_SCHEME_PATTERN = /^[a-z][a-z0-9+.-]*:/i

function get_extension(file_name: string): string {
    const dot = file_name.lastIndexOf('.')
    return dot >= 0 ? file_name.slice(dot + 1).toLowerCase() : ''
}

export function is_markdown_file_name(file_name: string): boolean {
    return MARKDOWN_EXTENSIONS.has(get_extension(file_name))
}

// Markdownを行境界で切り詰める。
// 文字数で単純に切ると コードフェンス (```) の途中で切れて
// 以降がまるごとコードブロック扱いになるため、
// 行境界で切ったうえで閉じられていないフェンスを閉じる。
export function truncate_markdown(markdown: string, max_length: number): string {
    if (markdown.length <= max_length) {
        return markdown
    }

    const cut = markdown.slice(0, max_length)
    const last_newline = cut.lastIndexOf('\n')
    let truncated = last_newline > 0 ? cut.slice(0, last_newline) : cut

    // 閉じられていないコードフェンスを閉じる
    let fence_count = 0
    for (const line of truncated.split('\n')) {
        if (line.trimStart().startsWith('```')) {
            fence_count++
        }
    }
    if (fence_count % 2 !== 0) {
        truncated += '\n```'
    }

    return truncated + '\n\n…'
}

// 相対URLを base_url 基準の絶対URLに解決する。
// スキーム付きURL・プロトコル相対URL・ページ内アンカーはそのまま返す。
function resolve_url(url: string, base: URL): string {
    if (url === '' || url.startsWith('#') || url.startsWith('//') || HAS_SCHEME_PATTERN.test(url)) {
        return url
    }
    try {
        return new URL(url, base).href
    } catch {
        // 解決できないものはそのまま
        return url
    }
}

// MarkdownをHTMLに変換し、サニタイズしたうえで相対URLを解決して返す。
// marked と dompurify は動的importする (初期バンドルに含めないため)。
export async function markdown_to_safe_html(markdown: string, base_url: string): Promise<string> {
    const [{ Marked }, { default: DOMPurify }] = await Promise.all([
        import('marked'),
        import('dompurify'),
    ])

    // グローバルなmarkedシングルトンはuse()で状態が汚れるためインスタンスを使う
    const marked = new Marked({ gfm: true, breaks: false })
    const dirty_html = await marked.parse(markdown)
    const clean_html = DOMPurify.sanitize(dirty_html)

    // サニタイズ済みHTMLをinertなtemplateに載せてURLを解決する。
    // detachedなdivだと innerHTML 代入時点で画像の読み込みが走り、
    // 書き換え前の相対URLに対して404リクエストが飛んでしまう。
    // template の中身はブラウジングコンテキストを持たないため読み込みが走らない。
    const template = document.createElement('template')
    template.innerHTML = clean_html

    const base = new URL(base_url, document.baseURI)

    for (const img of template.content.querySelectorAll('img')) {
        const src = img.getAttribute('src')
        if (src !== null) {
            img.setAttribute('src', resolve_url(src, base))
        }
    }

    for (const anchor of template.content.querySelectorAll('a')) {
        const href = anchor.getAttribute('href')
        if (href !== null) {
            anchor.setAttribute('href', resolve_url(href, base))
        }
        anchor.setAttribute('target', '_blank')
        anchor.setAttribute('rel', 'noopener noreferrer')
    }

    return template.innerHTML
}
