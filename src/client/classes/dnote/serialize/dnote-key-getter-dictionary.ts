const DnoteKeyGetterDictionary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default DnoteKeyGetterDictionary
// 循環参照対策で各KeyGetterの.tsファイルから登録