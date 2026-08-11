package reps

// プラグインが返したタグ・テキスト・通知を、既存のリポジトリ契約に見せかけるアダプタ。
//
// 型別アダプタ（plugin_typed_adapters.go）との唯一の違いは
// GetLatestDataRepositoryAddress を実装することにある。
//
// TagReps / TextReps / NotificationReps は UpdateCache の
// getAddrTargets に含まれ、find_filter が isLatestData(tag.ID, tag.UpdateTime) で
// 各件をふるいにかける。アドレスを返さないと、--cache_in_memory=false で
// プラグインのタグ・テキストが全部落ちる。
// GkillRepositories.FindTags の rep 名別ID絞り込みにも要る。

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// ---- Tag ----

type pluginTagRepositoryImpl struct{ pluginAdapterBase }

var _ TagRepository = (*pluginTagRepositoryImpl)(nil)

func (p *pluginTagRepositoryImpl) FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error) {
	snapshot := p.index.Ensure(ctx)
	tags := []Tag{}
	for _, tag := range snapshot.tagsByID {
		if !pluginMatchIDs(tag.ID, query) {
			continue
		}
		// タグの検索対象はタグ名。
		if !pluginMatchWords(tag.Tag, tag.ID, query) {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (p *pluginTagRepositoryImpl) GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error) {
	snapshot := p.index.Ensure(ctx)
	tag, exist := snapshot.tagsByID[id]
	if !exist {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(tag.UpdateTime, *updateTime) {
		return nil, nil
	}
	return &tag, nil
}

func (p *pluginTagRepositoryImpl) GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error) {
	snapshot := p.index.Ensure(ctx)
	tags := []Tag{}
	for _, tag := range snapshot.tagsByID {
		if tag.Tag == tagname {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// GetTagsByTargetID は全Kyouに対して呼ばれる最頻経路。map1回引きで即答する。
func (p *pluginTagRepositoryImpl) GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error) {
	snapshot := p.index.Ensure(ctx)
	return append([]Tag{}, snapshot.tagsByTarget[target_id]...), nil
}

func (p *pluginTagRepositoryImpl) GetTagHistories(ctx context.Context, id string) ([]Tag, error) {
	tag, err := p.GetTag(ctx, id, nil)
	if err != nil || tag == nil {
		return []Tag{}, err
	}
	return []Tag{*tag}, nil
}

func (p *pluginTagRepositoryImpl) GetAllTags(ctx context.Context) ([]Tag, error) {
	snapshot := p.index.Ensure(ctx)
	tags := make([]Tag, 0, len(snapshot.tagsByID))
	for _, tag := range snapshot.tagsByID {
		tags = append(tags, tag)
	}
	return tags, nil
}

// GetAllTagNames はプラグインのタグ名をタグ一覧に載せる。
//
// これがあるおかげで、プラグインが付けたタグがrykvの既定の絞り込み
// 「タグ無し」から漏れて何も表示されなくなる、という問題が起きない。
func (p *pluginTagRepositoryImpl) GetAllTagNames(ctx context.Context) ([]string, error) {
	snapshot := p.index.Ensure(ctx)
	nameSet := map[string]struct{}{}
	for _, tag := range snapshot.tagsByID {
		nameSet[tag.Tag] = struct{}{}
	}
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	return names, nil
}

func (p *pluginTagRepositoryImpl) AddTagInfo(_ context.Context, _ Tag) error {
	return p.readOnlyError("AddTagInfo")
}

func (p *pluginTagRepositoryImpl) GetLatestDataRepositoryAddress(ctx context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	snapshot := p.index.Ensure(ctx)
	addresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(snapshot.tagsByID))
	repName := p.repName()
	now := time.Now()
	for _, tag := range snapshot.tagsByID {
		targetID := tag.TargetID
		addresses = append(addresses, gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &targetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: now,
		})
	}
	return addresses, nil
}

func (p *pluginTagRepositoryImpl) UnWrapTyped() ([]TagRepository, error) {
	return []TagRepository{p}, nil
}

// ---- Text ----

type pluginTextRepositoryImpl struct{ pluginAdapterBase }

var _ TextRepository = (*pluginTextRepositoryImpl)(nil)

func (p *pluginTextRepositoryImpl) FindTexts(ctx context.Context, query *find.FindQuery) ([]Text, error) {
	snapshot := p.index.Ensure(ctx)
	texts := []Text{}
	for _, text := range snapshot.textsByID {
		if !pluginMatchIDs(text.ID, query) {
			continue
		}
		if !pluginMatchWords(text.Text, text.ID, query) {
			continue
		}
		texts = append(texts, text)
	}
	return texts, nil
}

func (p *pluginTextRepositoryImpl) GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error) {
	snapshot := p.index.Ensure(ctx)
	text, exist := snapshot.textsByID[id]
	if !exist {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(text.UpdateTime, *updateTime) {
		return nil, nil
	}
	return &text, nil
}

// GetTextsByTargetID は契約どおり削除済みを除外しない。
func (p *pluginTextRepositoryImpl) GetTextsByTargetID(ctx context.Context, target_id string) ([]Text, error) {
	snapshot := p.index.Ensure(ctx)
	return append([]Text{}, snapshot.textsByTarget[target_id]...), nil
}

func (p *pluginTextRepositoryImpl) GetTextHistories(ctx context.Context, id string) ([]Text, error) {
	text, err := p.GetText(ctx, id, nil)
	if err != nil || text == nil {
		return []Text{}, err
	}
	return []Text{*text}, nil
}

func (p *pluginTextRepositoryImpl) AddTextInfo(_ context.Context, _ Text) error {
	return p.readOnlyError("AddTextInfo")
}

func (p *pluginTextRepositoryImpl) GetLatestDataRepositoryAddress(ctx context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	snapshot := p.index.Ensure(ctx)
	addresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(snapshot.textsByID))
	repName := p.repName()
	now := time.Now()
	for _, text := range snapshot.textsByID {
		targetID := text.TargetID
		addresses = append(addresses, gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              text.IsDeleted,
			TargetID:                               text.ID,
			TargetIDInData:                         &targetID,
			DataUpdateTime:                         text.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: now,
		})
	}
	return addresses, nil
}

func (p *pluginTextRepositoryImpl) UnWrapTyped() ([]TextRepository, error) {
	return []TextRepository{p}, nil
}

// ---- Notification ----

type pluginNotificationRepositoryImpl struct{ pluginAdapterBase }

var _ NotificationRepository = (*pluginNotificationRepositoryImpl)(nil)

func (p *pluginNotificationRepositoryImpl) FindNotifications(ctx context.Context, query *find.FindQuery) ([]Notification, error) {
	snapshot := p.index.Ensure(ctx)
	notifications := []Notification{}
	for _, notification := range snapshot.notificationsByID {
		if !pluginMatchIDs(notification.ID, query) {
			continue
		}
		if !pluginMatchWords(notification.Content, notification.ID, query) {
			continue
		}
		notifications = append(notifications, notification)
	}
	return notifications, nil
}

func (p *pluginNotificationRepositoryImpl) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	snapshot := p.index.Ensure(ctx)
	notification, exist := snapshot.notificationsByID[id]
	if !exist {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(notification.UpdateTime, *updateTime) {
		return nil, nil
	}
	return &notification, nil
}

func (p *pluginNotificationRepositoryImpl) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]Notification, error) {
	snapshot := p.index.Ensure(ctx)
	return append([]Notification{}, snapshot.notificationsByTarget[target_id]...), nil
}

// GetNotificationsBetweenNotificationTime は契約どおり最新版に絞らず、
// 削除済み・通知済みも除外しない（絞るのは GkillNotificator の仕事）。
func (p *pluginNotificationRepositoryImpl) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]Notification, error) {
	snapshot := p.index.Ensure(ctx)
	notifications := []Notification{}
	for _, notification := range snapshot.notificationsByID {
		notificationTime := notification.NotificationTime.Local()
		if notificationTime.Before(startTime.Local()) || notificationTime.After(endTime.Local()) {
			continue
		}
		notifications = append(notifications, notification)
	}
	return notifications, nil
}

func (p *pluginNotificationRepositoryImpl) GetNotificationHistories(ctx context.Context, id string) ([]Notification, error) {
	notification, err := p.GetNotification(ctx, id, nil)
	if err != nil || notification == nil {
		return []Notification{}, err
	}
	return []Notification{*notification}, nil
}

func (p *pluginNotificationRepositoryImpl) AddNotificationInfo(_ context.Context, _ Notification) error {
	return p.readOnlyError("AddNotificationInfo")
}

func (p *pluginNotificationRepositoryImpl) GetLatestDataRepositoryAddress(ctx context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	snapshot := p.index.Ensure(ctx)
	addresses := make([]gkill_cache.LatestDataRepositoryAddress, 0, len(snapshot.notificationsByID))
	repName := p.repName()
	now := time.Now()
	for _, notification := range snapshot.notificationsByID {
		targetID := notification.TargetID
		addresses = append(addresses, gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              notification.IsDeleted,
			TargetID:                               notification.ID,
			TargetIDInData:                         &targetID,
			DataUpdateTime:                         notification.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: now,
		})
	}
	return addresses, nil
}

func (p *pluginNotificationRepositoryImpl) UnWrapTyped() ([]NotificationRepository, error) {
	return []NotificationRepository{p}, nil
}
