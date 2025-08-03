const urlRegex = /^(https?:\/\/)(?:\S+(?::\S*)?@)?((?:[A-Za-z0-9\p{L}\p{N}-]+\.)+[A-Za-z\p{L}]{2,}|\d{1,3}(?:\.\d{1,3}){3})(?::\d{2,5})?(?:\/[^\s?#]*)?(?:\?[^\s#]*)?(?:#[^\s]*)?$/u;

export function looksLikeUrl(str: string | null | undefined): str is string {
    return !!str && urlRegex.test(str.trim());
}