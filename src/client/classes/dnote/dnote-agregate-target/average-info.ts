export default class AverageInfo {
    public total_count: number = 0
    public total_value: any = null
    clone(): AverageInfo {
        const clone = new AverageInfo()
        clone.total_count = this.total_count
        clone.total_value = this.total_value
        return clone
    }
}