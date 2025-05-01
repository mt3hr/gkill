import AggregateAverageTimeIsStartTime from "../dnote-aggregate-target/aggregate-average-timeis-start-time"
import AggregateAverageTimeisTime from "../dnote-aggregate-target/aggregate-average-timeis-time"
import AggregateSumTimeisTime from "../dnote-aggregate-target/aggregate-sum-timeis-time"

const AggregateTargetDictionary = new Map<String, any>()
AggregateTargetDictionary.set("AggregateSumTimeIs", AggregateSumTimeisTime)
AggregateTargetDictionary.set("AggregateAverageTimeIs", AggregateAverageTimeisTime)
AggregateTargetDictionary.set("AggregateAverageTimeIsStartTime", AggregateAverageTimeIsStartTime)
AggregateTargetDictionary.set("AggregateAverageTimeIsEndTime", AggregateAverageTimeIsStartTime)

export default AggregateTargetDictionary
