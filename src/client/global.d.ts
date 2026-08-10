/// <reference types="vite-plugin-pwa/client" />

// User-Agent Client Hints。Chromium系のみ実装されており、TypeScriptの標準lib
// にはまだ含まれていないため最小限だけ宣言する。use-device-kind.ts が使用する。
interface NavigatorUAData {
    readonly mobile?: boolean;
    readonly platform?: string;
}

interface Navigator {
    readonly userAgentData?: NavigatorUAData;
}