const DnoteKyouFilterDictionary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default DnoteKyouFilterDictionary
// 循環参照対策で各KeyGetterの.tsファイルから登録