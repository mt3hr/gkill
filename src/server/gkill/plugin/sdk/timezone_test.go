package sdk

import (
	"errors"
	"testing"
	"time"
)

// Android では端末のゾーン名を読んで time.Local を差し替える。ほかの OS では読みもしない（ADR-0319）。
func TestFixTimezoneFor(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })

	readCalled := false
	if _, ok := fixTimezoneFor("windows", func() (string, error) { readCalled = true; return "Asia/Tokyo", nil }); ok || readCalled {
		t.Fatalf("Android 以外で直した（ok=%v）か、ゾーン名を読んだ（%v）", ok, readCalled)
	}
	if time.Local != original {
		t.Fatal("Android 以外で time.Local が変わった")
	}

	time.Local = time.UTC
	name, ok := fixTimezoneFor("android", func() (string, error) { return "Asia/Tokyo\n", nil })
	if !ok || name != "Asia/Tokyo" {
		t.Fatalf("fixTimezoneFor = %q %v, want Asia/Tokyo true（getprop の末尾の改行は落とす）", name, ok)
	}
	if got := time.Local.String(); got != "Asia/Tokyo" {
		t.Errorf("time.Local = %q, want Asia/Tokyo", got)
	}
	// 直したあとに time.Local で読む時刻は端末のゾーンの壁時計になる
	parsed, err := time.ParseInLocation("2006.01.02 15:04:05", "2024.01.01 09:00:00", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Asia/Tokyo の 09:00 = %v, want 00:00 UTC", parsed.UTC())
	}
}

// 読めない・空・知らないゾーン名なら UTC のまま続ける（起動は止めない）。
func TestFixTimezoneForKeepsUTCOnFailure(t *testing.T) {
	original := time.Local
	t.Cleanup(func() { time.Local = original })

	cases := map[string]func() (string, error){
		"getprop が失敗": func() (string, error) { return "", errors.New("no getprop") },
		"空":           func() (string, error) { return "\n", nil },
		"知らないゾーン名":    func() (string, error) { return "Not/AZone", nil },
	}
	for label, read := range cases {
		time.Local = time.UTC
		if _, ok := fixTimezoneFor("android", read); ok {
			t.Errorf("%s: 直したことになった", label)
		}
		if time.Local != time.UTC {
			t.Errorf("%s: time.Local = %v, want UTC のまま", label, time.Local)
		}
	}
}
