import DataTypeGetter from "../dnote-key-getter/data-type-getter"
import LantanaMoodGetter from "../dnote-key-getter/lantana-mood-getter"
import NlogShopNameGetter from "../dnote-key-getter/nlog-shop-name-getter"
import RelatedMonthGetter from "../dnote-key-getter/related-month-getter"
import RelatedWeekDayGetter from "../dnote-key-getter/related-week-day-getter"
import RelatedWeekGetter from "../dnote-key-getter/related-week-getter"
import RelatedDateGetter from "../dnote-key-getter/rerated-date-getter"
import TagGetter from "../dnote-key-getter/tag-getter"
import TitleGetter from "../dnote-key-getter/title-getter"

const DnoteKeyGetterDictionary = new Map<String, any>()
DnoteKeyGetterDictionary.set("DataTypeGetter", DataTypeGetter)
DnoteKeyGetterDictionary.set("LantanaMoodGetter", LantanaMoodGetter)
DnoteKeyGetterDictionary.set("NlogShopNameGetter", NlogShopNameGetter)
DnoteKeyGetterDictionary.set("RelatedMonthGetter", RelatedMonthGetter)
DnoteKeyGetterDictionary.set("RelatedWeekDayGetter", RelatedWeekDayGetter)
DnoteKeyGetterDictionary.set("RelatedWeekGetter", RelatedWeekGetter)
DnoteKeyGetterDictionary.set("RelatedDateGetter", RelatedDateGetter)
DnoteKeyGetterDictionary.set("TagGetter", TagGetter)
DnoteKeyGetterDictionary.set("TitleGetter", TitleGetter)

export default DnoteKeyGetterDictionary