package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// logTimeLayout は出力ログの各行の先頭の時刻。タイムゾーンを持たないローカル時刻。
const logTimeLayout = "2006.01.02 15:04:05"

// logLocation はログの時刻を読むときのタイムゾーン。
// VRChat は動いている PC のローカル時刻で書くので、このプラグインが動く PC のローカル時刻で読む。
// テストが固定のゾーンへ差し替える。
var logLocation = time.Local

// maxLineBytes は1行として読む上限。VRChat のログには JSON を丸ごと出す長い行があるが、
// 拾う行はどれも短いので、超えたぶんは読み捨てる（読み捨てた行の数は数えて設定画面に出す）。
const maxLineBytes = 64 * 1024

// logLinePattern は時刻付きの行。`2006.01.02 15:04:05 Debug      -  本文`
// 時刻の無い行（例外のスタックトレースなどの継続行）は拾わない。
var logLinePattern = regexp.MustCompile(`^(\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}) +\S+ +- +(.*)$`)

// 拾う行の本文。どれも行頭の時刻とレベルを外したあとの本文に当てる。
var (
	joiningPattern       = regexp.MustCompile(`^\[Behaviour\] Joining (wrld_[0-9A-Za-z-]+)(?::(\S+))?\s*$`)
	enteringRoomPattern  = regexp.MustCompile(`^\[Behaviour\] Entering Room: (.*?)\s*$`)
	leftRoomPattern      = regexp.MustCompile(`^\[Behaviour\] OnLeftRoom\s*$`)
	localPlayerPattern   = regexp.MustCompile(`^\[Behaviour\] Initialized PlayerAPI "(.*)" is local\s*$`)
	authenticatedPattern = regexp.MustCompile(`User Authenticated: (.+?) \((usr_[0-9A-Za-z-]+)\)`)
	playerJoinedPattern  = regexp.MustCompile(`^\[Behaviour\] OnPlayerJoined (.+?)(?: \((usr_[0-9A-Za-z-]+)\))?\s*$`)
	playerLeftPattern    = regexp.MustCompile(`^\[Behaviour\] OnPlayerLeft (.+?)(?: \((usr_[0-9A-Za-z-]+)\))?\s*$`)
	avatarPattern        = regexp.MustCompile(`^\[Behaviour\] Switching (.+) to avatar (.+?)\s*$`)
	videoPattern         = regexp.MustCompile(`^\[Video Playback\] (?:Attempting to resolve|Resolving) URL '([^']+)'`)
	quitPattern          = regexp.MustCompile(`VRCApplication: (?:HandleApplicationQuit|OnApplicationQuit)`)
)

// 出来事の種類。
const (
	evJoining = iota + 1
	evEnteringRoom
	evLeftRoom
	evPlayerJoined
	evPlayerLeft
	evAvatar
	evVideo
	evQuit
)

// logEvent はログから拾った出来事1つ。
type logEvent struct {
	Kind int
	Time time.Time
	// Name はワールド名・人の表示名・アバターを切り替えた人の表示名。
	Name string
	// ID はワールド ID か人の usr ID。
	ID string
	// Value はインスタンス・アバター名・動画の URL。
	Value string
}

// parsedLog は1ファイル（1回の起動）の解析結果。
type parsedLog struct {
	// SessionKey は最初の時刻付きの行の時刻（書いてあるとおりの文字列）。
	// ファイル名や置き場所が変わっても同じ起動なら同じ値になるので、Kyou ID の土台にする。
	// 時刻付きの行が1つも無ければ空。
	SessionKey string
	Events     []logEvent
	// SelfNames / SelfIDs はこのログを書いた本人の表示名と usr ID。
	// 同じ部屋にいた人から本人を外し、アバターの切り替えを本人のものだけにするのに使う。
	SelfNames map[string]struct{}
	SelfIDs   map[string]struct{}
	// LastTime は最後の時刻付きの行の時刻。終了の行が無い（落ちた・プレイ中）ときはここで全部閉じる。
	LastTime time.Time
	// Dropped は長すぎて読み捨てた行の数。
	Dropped int
}

// parseLogFile は出力ログを1つ読む。
func parseLogFile(path string) (parsedLog, error) {
	file, err := os.Open(path)
	if err != nil {
		return parsedLog{}, err
	}
	defer func() { _ = file.Close() }()
	return parseLog(file)
}

// parseLog は出力ログを読んで出来事の列にする。
func parseLog(r io.Reader) (parsedLog, error) {
	parsed := parsedLog{SelfNames: map[string]struct{}{}, SelfIDs: map[string]struct{}{}}
	reader := bufio.NewReaderSize(r, 64*1024)

	for {
		line, tooLong, err := readLine(reader)
		if tooLong {
			parsed.Dropped++
		}
		if line != "" && !tooLong {
			parseLine(&parsed, line)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return parsed, nil
			}
			return parsed, err
		}
	}
}

// readLine は1行読む。maxLineBytes を超えた行は残りを読み捨てて tooLong を返す。
func readLine(reader *bufio.Reader) (string, bool, error) {
	var buf []byte
	tooLong := false
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if !tooLong {
			if len(buf)+len(chunk) > maxLineBytes {
				tooLong = true
				buf = nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		if err != nil {
			return string(buf), tooLong, err
		}
		if !isPrefix {
			return string(buf), tooLong, nil
		}
	}
}

// parseLine は時刻付きの行を1つ解釈する。
func parseLine(parsed *parsedLog, line string) {
	matched := logLinePattern.FindStringSubmatch(strings.TrimRight(line, "\r"))
	if matched == nil {
		return
	}
	at, err := time.ParseInLocation(logTimeLayout, matched[1], logLocation)
	if err != nil {
		return
	}
	if parsed.SessionKey == "" {
		parsed.SessionKey = matched[1]
	}
	parsed.LastTime = at
	body := matched[2]

	add := func(event logEvent) {
		event.Time = at
		parsed.Events = append(parsed.Events, event)
	}

	switch {
	case strings.HasPrefix(body, "[Behaviour] Joining wrld_"):
		if m := joiningPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evJoining, ID: m[1], Value: m[2]})
		}
	case strings.HasPrefix(body, "[Behaviour] Entering Room: "):
		if m := enteringRoomPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evEnteringRoom, Name: m[1]})
		}
	case leftRoomPattern.MatchString(body):
		add(logEvent{Kind: evLeftRoom})
	case strings.HasPrefix(body, "[Behaviour] OnPlayerJoined "):
		if m := playerJoinedPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evPlayerJoined, Name: m[1], ID: m[2]})
		}
	case strings.HasPrefix(body, "[Behaviour] OnPlayerLeft "):
		if m := playerLeftPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evPlayerLeft, Name: m[1], ID: m[2]})
		}
	case strings.HasPrefix(body, "[Behaviour] Switching ") && strings.Contains(body, " to avatar "):
		if m := avatarPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evAvatar, Name: m[1], Value: m[2]})
		}
	case strings.HasPrefix(body, "[Video Playback] "):
		if m := videoPattern.FindStringSubmatch(body); m != nil {
			add(logEvent{Kind: evVideo, Value: m[1]})
		}
	case strings.HasPrefix(body, "[Behaviour] Initialized PlayerAPI "):
		if m := localPlayerPattern.FindStringSubmatch(body); m != nil && m[1] != "" {
			parsed.SelfNames[m[1]] = struct{}{}
		}
	case quitPattern.MatchString(body):
		add(logEvent{Kind: evQuit})
	default:
		if m := authenticatedPattern.FindStringSubmatch(body); m != nil {
			parsed.SelfNames[m[1]] = struct{}{}
			parsed.SelfIDs[m[2]] = struct{}{}
		}
	}
}
