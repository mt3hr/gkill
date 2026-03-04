export function isUrl(str: string | null | undefined): str is string {
  if (!str) return false;
  try {
    const u = new URL(str.trim());
    return u.protocol === 'http:' || u.protocol === 'https:';
  } catch {
    return false;  // URL() が投げたら不正
  }
}