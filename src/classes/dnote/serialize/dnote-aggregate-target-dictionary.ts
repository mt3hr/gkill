import AggregateAverageGitCommitLogCodeCount from "../dnote-aggregate-target/aggregate-average-git-commit-log-code-count"
import AggregateAverageLantanaMood from "../dnote-aggregate-target/aggregate-average-lantana-mood"
import AggregateAverageNlogAmount from "../dnote-aggregate-target/aggregate-average-nlog-amount"
import AggregateAverageTimeIsStartTime from "../dnote-aggregate-target/aggregate-average-timeis-start-time"
import AggregateAverageTimeisTime from "../dnote-aggregate-target/aggregate-average-timeis-time"
import AggregateCountKyou from "../dnote-aggregate-target/aggregate-count-kyou"
import AggregateSumGitCommitLogCodeCount from "../dnote-aggregate-target/aggregate-sum-git-commit-log-code-count"
import AggregateSumLantanaMood from "../dnote-aggregate-target/aggregate-sum-lantana-mood"
import AggregateSumNlogAmount from "../dnote-aggregate-target/aggregate-sum-nlog-amount"
import AggregateSumTimeisTime from "../dnote-aggregate-target/aggregate-sum-timeis-time"

const AggregateTargetDictionary = new Map<String, any>()
AggregateTargetDictionary.set("AggregateAverageTimeIsTime", AggregateAverageTimeisTime)
AggregateTargetDictionary.set("AggregateAverageTimeIsStartTime", AggregateAverageTimeIsStartTime)
AggregateTargetDictionary.set("AggregateAverageTimeIsEndTime", AggregateAverageTimeIsStartTime)
AggregateTargetDictionary.set("AgregateAverageGitCommitLogCode", AggregateAverageGitCommitLogCodeCount)
AggregateTargetDictionary.set("AggregateAverageLantanaMood", AggregateAverageLantanaMood)
AggregateTargetDictionary.set("AggregateAverageNlogAmount", AggregateAverageNlogAmount)
AggregateTargetDictionary.set("AggregateCountKyou", AggregateCountKyou)
AggregateTargetDictionary.set("AgregateSumGitCommitLogCode", AggregateSumGitCommitLogCodeCount)
AggregateTargetDictionary.set("AggregateSumLantanaMood", AggregateSumLantanaMood)
AggregateTargetDictionary.set("AggregateSumNlogAmount", AggregateSumNlogAmount)
AggregateTargetDictionary.set("AggregateSumTimeIsTime", AggregateSumTimeisTime)

export default AggregateTargetDictionary
