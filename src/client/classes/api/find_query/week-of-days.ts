export enum WeekOfDays {
    sunday = 0,
    monday = 1,
    tuesday = 2,
    wednesday = 3,
    thursday = 4,
    friday = 5,
    saturday = 6,
}

/** 全曜日。検索条件の period_of_time_week_of_days に全7曜日を入れると「曜日制限なし」の意味になる */
export const ALL_WEEK_OF_DAYS: ReadonlyArray<WeekOfDays> = [
    WeekOfDays.sunday,
    WeekOfDays.monday,
    WeekOfDays.tuesday,
    WeekOfDays.wednesday,
    WeekOfDays.thursday,
    WeekOfDays.friday,
    WeekOfDays.saturday,
]

/** 全7曜日を含むか（重複や順序は問わない）。含めば「曜日制限なし」 */
export function is_all_week_of_days(week_of_days: ReadonlyArray<number>): boolean {
    return ALL_WEEK_OF_DAYS.every((w) => week_of_days.includes(w))
}
