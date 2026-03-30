const PredicateDictonary = new Map<string, { from_json: (json: Record<string, unknown>) => unknown }>()
export default PredicateDictonary
// 循環参照対策で各Predicateの.tsファイルから登録