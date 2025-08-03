// src/types/cookie-store.d.ts
interface CookieListItem {
    name: string;
    value: string;
    domain?: string;
    path?: string;
    expires?: number;
    sameSite?: 'strict' | 'lax' | 'none';
    secure?: boolean;
}

interface CookieStoreSetOptions extends CookieListItem {
    maxAge?: number;
}

interface CookieStoreDeleteOptions {
    name: string;
    domain?: string;
    path?: string;
}

interface CookieStore {
    get(name: string): Promise<CookieListItem | undefined>;
    getAll(name?: string): Promise<CookieListItem[]>;
    set(name: string, value: string): Promise<void>;
    set(options: CookieStoreSetOptions): Promise<void>;
    delete(name: string): Promise<void>;
    delete(options: CookieStoreDeleteOptions): Promise<void>;
}

declare const cookieStore: CookieStore;
