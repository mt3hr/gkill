'use strict'

// ファイルと base64 の相互変換。サーバとのファイルのやりとりは JSON の中の base64 で行う
// （gkill_fetch は JSON 以外の応答を受け付けず、セッションも本文の JSON で渡すため）。

// read_file_as_data_url はファイルを data URI（data:<type>;base64,...）として読む。
// サーバ側は "," より前を捨てて復号するので、そのまま送ってよい（handle_upload_files.go / handle_upload_skill.go）。
export function read_file_as_data_url(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.readAsDataURL(file)
        reader.onload = () => resolve(reader.result as string)
        reader.onerror = (error) => reject(error)
    })
}

// base64_to_blob は base64（data URI の接頭辞は付けない）を Blob に戻す。
export function base64_to_blob(base64: string, type: string): Blob {
    const binary = window.atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
    }
    return new Blob([bytes], { type })
}
