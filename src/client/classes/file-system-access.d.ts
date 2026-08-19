// File System Access API の最小宣言。
//
// TypeScript の標準 lib にはまだ入っていないので、`(window as any).showSaveFilePicker` で
// 逃げていた。使うのは「保存先を選ばせて逐次書き出す」経路（Dnoteの一括書き出し）だけなので、
// そこで必要な形だけを宣言する。cookie-store.d.ts と同じ扱い。
//
// 対応していないブラウザでは `"showSaveFilePicker" in window` が偽になり、
// Blob をまとめて作る従来の経路へ落ちる。

interface FileSystemWritableFileStream {
    write(data: string | BufferSource | Blob): Promise<void>;
    close(): Promise<void>;
}

interface FileSystemFileHandleLike {
    createWritable(): Promise<FileSystemWritableFileStream>;
}

interface ShowSaveFilePickerAcceptType {
    description?: string;
    accept: Record<string, Array<string>>;
}

interface ShowSaveFilePickerOptions {
    suggestedName?: string;
    types?: Array<ShowSaveFilePickerAcceptType>;
}

interface Window {
    showSaveFilePicker?(options?: ShowSaveFilePickerOptions): Promise<FileSystemFileHandleLike>;
}
