package main

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// unknownWorldTitle は Entering Room の行にワールド名が無かったときのタイトル。
const unknownWorldTitle = "（名前の分からないワールド）"

// stay は開いているワールドの滞在。
type stay struct {
	Ordinal  int
	WorldID  string
	Instance string
	Name     string
	Start    time.Time
	// players は部屋にいる人。キーは usr ID（無ければ "name:" + 表示名）。
	players map[string]*openPlayer
	// joinCounts は同じ人がこの滞在の中で何回入ってきたか。ID の材料にする。
	joinCounts map[string]int
	// lastVideoURL は直前の動画の URL。動画プレイヤーは同じ URL を何度も解決し直すので、続けて同じなら1件にする。
	lastVideoURL string
}

// openPlayer は部屋にいる人。
type openPlayer struct {
	Key   string
	Title string
	Start time.Time
	Join  int
}

// buildRecords は1回の起動ぶんの出来事を記録にする。
//
// 記録はすべてこのファイルの中で閉じる。VRChat の終了の行があればそこで、
// 無ければ（落ちた・プレイ中に運んだ）最後の時刻付きの行で、開いている TimeIs を全部閉じる。
// 結果がログの中身だけで決まるので、いつ取り込んでも同じになる。
//
// Kyou ID は追記で変わらない材料（滞在の通し番号・人ごとの入室の回数・出来事の通し番号）から作る。
// 閉じた順番から作ると、プレイ中に運んだ短い版と後の完全な版とで ID がずれる。
func buildRecords(parsed parsedLog) []record {
	if parsed.SessionKey == "" {
		return nil
	}
	builder := recordBuilder{parsed: parsed}
	builder.run()
	return builder.records
}

type recordBuilder struct {
	parsed     parsedLog
	records    []record
	current    *stay
	stayCount  int
	eventCount int
	// pendingWorldID / pendingInstance は Joining の行で分かった、次に入る部屋のワールド。
	pendingWorldID  string
	pendingInstance string
}

func (b *recordBuilder) run() {
	for _, event := range b.parsed.Events {
		switch event.Kind {
		case evJoining:
			// 今の VRChat は Entering Room のあとに Joining wrld_ を書く（ワールド名が先、ID が後）。
			// 古い書式は逆順なので、開いている滞在にまだ ID が無ければそこへ付け、
			// あれば次の部屋の ID として控える。
			if b.current != nil && b.current.WorldID == "" {
				b.attachWorld(event.ID, event.Value)
				continue
			}
			b.closeStay(event.Time)
			b.pendingWorldID, b.pendingInstance = event.ID, event.Value
		case evEnteringRoom:
			b.closeStay(event.Time)
			b.openStay(event)
		case evLeftRoom:
			b.closeStay(event.Time)
		case evPlayerJoined:
			b.playerJoined(event)
		case evPlayerLeft:
			b.playerLeft(event)
		case evAvatar:
			if _, self := b.parsed.SelfNames[event.Name]; self {
				b.addEvent(event.Time, "アバター変更: "+event.Value)
			}
		case evVideo:
			if b.current != nil {
				if b.current.lastVideoURL == event.Value {
					continue
				}
				b.current.lastVideoURL = event.Value
			}
			b.addEvent(event.Time, "動画: "+event.Value)
		case evQuit:
			b.closeStay(event.Time)
			return
		}
	}
	b.closeStay(b.parsed.LastTime)
}

func (b *recordBuilder) openStay(event logEvent) {
	b.stayCount++
	name := event.Name
	if name == "" {
		name = unknownWorldTitle
	}
	b.current = &stay{
		Ordinal:    b.stayCount,
		Name:       name,
		Start:      event.Time,
		players:    map[string]*openPlayer{},
		joinCounts: map[string]int{},
	}
	if b.pendingWorldID != "" {
		b.attachWorld(b.pendingWorldID, b.pendingInstance)
	}
	b.pendingWorldID, b.pendingInstance = "", ""
}

// attachWorld は開いている滞在にワールド ID とインスタンスを付け、そのワールドの URLog を作る。
// URLog の時刻は滞在の開始（Entering Room）にそろえる。
func (b *recordBuilder) attachWorld(worldID, instance string) {
	current := b.current
	current.WorldID, current.Instance = worldID, instance

	description := "ワールド ID: " + worldID
	if instance != "" {
		description += "\nインスタンス: " + instance
	}
	b.records = append(b.records, record{
		ID:          b.idOf(kindWorldURLog, strconv.Itoa(current.Ordinal)),
		SessionKey:  b.parsed.SessionKey,
		Kind:        kindWorldURLog,
		Title:       current.Name,
		Start:       current.Start,
		URL:         worldURLOf(worldID, instance),
		Description: description,
		Tags:        []string{tagWorld},
		SearchText:  joinSearchText(current.Name, worldID, instance),
	})
}

// closeStay は開いている滞在と、その部屋にいる人を時刻 at で閉じる。
func (b *recordBuilder) closeStay(at time.Time) {
	current := b.current
	if current == nil {
		return
	}
	b.current = nil

	keys := make([]string, 0, len(current.players))
	for key := range current.players {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.closePlayer(current, current.players[key], at)
	}

	b.records = append(b.records, record{
		ID:         b.idOf(kindWorldTimeIs, strconv.Itoa(current.Ordinal)),
		SessionKey: b.parsed.SessionKey,
		Kind:       kindWorldTimeIs,
		Title:      current.Name,
		Start:      current.Start,
		End:        laterOf(current.Start, at),
		Tags:       []string{tagWorld},
		SearchText: joinSearchText(current.Name, current.WorldID, current.Instance),
	})
}

func (b *recordBuilder) isSelf(name, id string) bool {
	if _, self := b.parsed.SelfNames[name]; self {
		return true
	}
	if id == "" {
		return false
	}
	_, self := b.parsed.SelfIDs[id]
	return self
}

func playerKeyOf(name, id string) string {
	if id != "" {
		return id
	}
	return "name:" + name
}

func (b *recordBuilder) playerJoined(event logEvent) {
	current := b.current
	if current == nil || b.isSelf(event.Name, event.ID) {
		return
	}
	key := playerKeyOf(event.Name, event.ID)
	if _, open := current.players[key]; open {
		return
	}
	current.joinCounts[key]++
	title := event.Name
	if event.ID != "" {
		title = event.Name + "（" + event.ID + "）"
	}
	current.players[key] = &openPlayer{Key: key, Title: title, Start: event.Time, Join: current.joinCounts[key]}
}

func (b *recordBuilder) playerLeft(event logEvent) {
	current := b.current
	if current == nil {
		return
	}
	key := playerKeyOf(event.Name, event.ID)
	player, open := current.players[key]
	if !open {
		return
	}
	b.closePlayer(current, player, event.Time)
}

func (b *recordBuilder) closePlayer(current *stay, player *openPlayer, at time.Time) {
	delete(current.players, player.Key)
	b.records = append(b.records, record{
		ID:         b.idOf(kindPlayer, strconv.Itoa(current.Ordinal), player.Key, strconv.Itoa(player.Join)),
		SessionKey: b.parsed.SessionKey,
		Kind:       kindPlayer,
		Title:      player.Title,
		Start:      player.Start,
		End:        laterOf(player.Start, at),
		Tags:       []string{tagRoom},
		// ワールド名でも引けるように、どこで一緒だったかも入れておく
		SearchText: joinSearchText(player.Title, current.Name, current.WorldID),
	})
}

func (b *recordBuilder) addEvent(at time.Time, content string) {
	b.eventCount++
	worldName := ""
	if b.current != nil {
		worldName = b.current.Name
	}
	b.records = append(b.records, record{
		ID:         b.idOf(kindEvent, strconv.Itoa(b.eventCount)),
		SessionKey: b.parsed.SessionKey,
		Kind:       kindEvent,
		Start:      at,
		Content:    content,
		Tags:       []string{tagEvent},
		SearchText: joinSearchText(content, worldName),
	})
}

// idOf は Kyou ID を作る。種類を材料に入れるので、種類をまたいで ID が重ならない
// （索引は同じ ID の Kyou を1レコードにまとめてしまう。ADR-0317）。
func (b *recordBuilder) idOf(kind string, parts ...string) string {
	return uuidV5(vrchatNamespace, appName+"|"+b.parsed.SessionKey+"|"+kind+"|"+strings.Join(parts, "|"))
}

// laterOf は終了が開始より前にならないようにする（時計が戻った行への備え）。
func laterOf(start, end time.Time) time.Time {
	if end.Before(start) {
		return start
	}
	return end
}

func joinSearchText(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n")
}
