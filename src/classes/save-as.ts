export function saveAs(data: Blob | File | string, filename: string): void {
    // 1) 引数を Blob に整形
    const blob =
        data instanceof Blob
            ? data
            : new Blob([data], { type: "application/octet-stream" });

    // 2) 一時 URL を生成
    const url = URL.createObjectURL(blob);

    // 3) <a download> を動的に作成してクリック
    const anchor = document.createElement("a");
    anchor.style.display = "none";
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();

    // 4) 後片付け
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
}