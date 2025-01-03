package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

type NotificationRepositories []NotificationRepository

func (t NotificationRepositories) FindNotifications(ctx context.Context, query *find.FindQuery) ([]*Notification, error) {
	matchNotifications := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.FindNotifications(ctx, query)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find notification: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := matchNotifications[notification.ID]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						matchNotifications[notification.ID] = notification
					}
				} else {
					matchNotifications[notification.ID] = notification
				}
			}
		default:
			break loop
		}
	}

	matchNotificationsList := []*Notification{}
	for _, notification := range matchNotifications {
		if notification == nil {
			continue
		}
		matchNotificationsList = append(matchNotificationsList, notification)
	}

	sort.Slice(matchNotificationsList, func(i, j int) bool {
		return matchNotificationsList[i].NotificationTime.After(matchNotificationsList[j].NotificationTime)
	})
	return matchNotificationsList, nil
}

func (t NotificationRepositories) Close(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			err = rep.Close(ctx)
			if err != nil {
				errch <- err
				return
			}
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at close: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	return nil
}

func (t NotificationRepositories) GetNotification(ctx context.Context, id string, updateTime *time.Time) (*Notification, error) {
	matchNotification := &Notification{}
	matchNotification = nil
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan *Notification, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationInRep, err := rep.GetNotification(ctx, id, updateTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at get notification: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationInRep := <-ch:
			if matchNotificationInRep == nil {
				continue loop
			}
			if matchNotification != nil {
				if matchNotificationInRep.UpdateTime.Before(matchNotification.UpdateTime) {
					matchNotification = matchNotificationInRep
				}
			} else {
				matchNotification = matchNotificationInRep
			}
		default:
			break loop
		}
	}

	return matchNotification, nil
}

func (t NotificationRepositories) GetNotificationsByTargetID(ctx context.Context, target_id string) ([]*Notification, error) {
	matchNotifications := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.GetNotificationsByTargetID(ctx, target_id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get notification histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := matchNotifications[notification.ID]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						matchNotifications[notification.ID] = notification
					}
				} else {
					matchNotifications[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []*Notification{}
	for _, notification := range matchNotifications {
		if notification == nil {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	sort.Slice(notificationHistoriesList, func(i, j int) bool {
		return notificationHistoriesList[i].UpdateTime.After(notificationHistoriesList[j].UpdateTime)
	})

	return notificationHistoriesList, nil
}

func (t NotificationRepositories) GetNotificationsBetweenNotificationTime(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Notification, error) {
	matchNotifications := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.GetNotificationsBetweenNotificationTime(ctx, startTime, endTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get notification histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := matchNotifications[notification.ID]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						matchNotifications[notification.ID] = notification
					}
				} else {
					matchNotifications[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []*Notification{}
	for _, notification := range matchNotifications {
		if notification == nil {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	sort.Slice(notificationHistoriesList, func(i, j int) bool {
		return notificationHistoriesList[i].UpdateTime.After(notificationHistoriesList[j].UpdateTime)
	})

	return notificationHistoriesList, nil
}

func (t NotificationRepositories) UpdateCache(ctx context.Context) error {
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	errch := make(chan error, len(t))
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			err = rep.UpdateCache(ctx)
			if err != nil {
				errch <- err
				return
			}
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at update cache: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return err
	}

	return nil
}

func (t NotificationRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements NotificationReps.GetPath")
	return "", err
}

func (t NotificationRepositories) GetRepName(ctx context.Context) (string, error) {
	return "NotificationReps", nil
}

func (t NotificationRepositories) GetNotificationHistories(ctx context.Context, id string) ([]*Notification, error) {
	notificationHistories := map[string]*Notification{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*Notification, len(t))
	errch := make(chan error, len(t))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range t {
		wg.Add(1)

		go func(rep NotificationRepository) {
			defer wg.Done()
			matchNotificationsInRep, err := rep.GetNotificationHistories(ctx, id)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchNotificationsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find get notification histories: %w", e)
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// Notification集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchNotificationsInRep := <-ch:
			if matchNotificationsInRep == nil {
				continue loop
			}
			for _, notification := range matchNotificationsInRep {
				if existNotification, exist := notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)]; exist {
					if notification.UpdateTime.After(existNotification.UpdateTime) {
						notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
					}
				} else {
					notificationHistories[notification.ID+notification.UpdateTime.Format(sqlite3impl.TimeLayout)] = notification
				}
			}
		default:
			break loop
		}
	}

	notificationHistoriesList := []*Notification{}
	for _, notification := range notificationHistories {
		if notification == nil {
			continue
		}
		notificationHistoriesList = append(notificationHistoriesList, notification)
	}

	sort.Slice(notificationHistoriesList, func(i, j int) bool {
		return notificationHistoriesList[i].UpdateTime.After(notificationHistoriesList[j].UpdateTime)
	})

	return notificationHistoriesList, nil
}

func (t NotificationRepositories) AddNotificationInfo(ctx context.Context, text *Notification) error {
	err := fmt.Errorf("not implements NotificationReps.AddNotificationInfo")
	return err
}
