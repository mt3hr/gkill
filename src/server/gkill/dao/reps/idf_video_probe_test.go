package reps

// 互換動画へ変換するかどうかの判定のテスト。
//
// クライアントには再生失敗の受け皿が無く、再生できなければエラーも出ずに
// 無音の黒枠になる。したがって判定は「原本のまま確実に再生できる」と
// 言い切れるときだけ変換を省く、という一方向の非対称でなければならない。
//
// ffprobe を実行せずに回せるよう、判定は純粋関数に切り出してある。

import (
	"encoding/json"
	"testing"
)

func videoStream(codec string, profile string, pixFmt string) ffprobeStream {
	return ffprobeStream{CodecType: "video", CodecName: codec, Profile: profile, PixFmt: pixFmt}
}

func audioStream(codec string) ffprobeStream {
	return ffprobeStream{CodecType: "audio", CodecName: codec}
}

func TestVideoNeedsCompat(t *testing.T) {
	cases := []struct {
		name    string
		ext     string
		streams []ffprobeStream
		want    bool
	}{
		{
			name:    "mp4のH.264とAACはそのまま配信できる",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High", "yuv420p"), audioStream("aac")},
			want:    false,
		},
		{
			name:    "音声が無くてもよい",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "Main", "yuvj420p")},
			want:    false,
		},
		{
			name:    "movのH.264もそのまま配信できる",
			ext:     ".mov",
			streams: []ffprobeStream{videoStream("h264", "Baseline", "yuv420p"), audioStream("aac")},
			want:    false,
		},
		{
			name:    "webmのVP9とOpusはそのまま配信できる",
			ext:     ".webm",
			streams: []ffprobeStream{videoStream("vp9", "Profile 0", "yuv420p"), audioStream("opus")},
			want:    false,
		},
		{
			name:    "拡張子の大小は問わない",
			ext:     ".MP4",
			streams: []ffprobeStream{videoStream("H264", "High", "YUV420P"), audioStream("AAC")},
			want:    false,
		},
		{
			name:    "HEVCは変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("hevc", "Main", "yuv420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "mp4のVP9は変換する（Safariが再生できない）",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("vp9", "Profile 0", "yuv420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "対応しないコンテナは中身がH.264でも変換する",
			ext:     ".mkv",
			streams: []ffprobeStream{videoStream("h264", "High", "yuv420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "10bitは変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High 10", "yuv420p10le"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "4:2:2は変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High 4:2:2", "yuv422p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "画素形式が通っていてもプロファイルが対象外なら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High 10", "yuv420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "音声がAC3なら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High", "yuv420p"), audioStream("ac3")},
			want:    true,
		},
		{
			name:    "webmにAACが入っていたら変換する",
			ext:     ".webm",
			streams: []ffprobeStream{videoStream("vp9", "Profile 0", "yuv420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "映像が2本あったら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High", "yuv420p"), videoStream("mjpeg", "", "yuvj420p"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "音声が2本あったら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{videoStream("h264", "High", "yuv420p"), audioStream("aac"), audioStream("aac")},
			want:    true,
		},
		{
			name:    "映像が無かったら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{audioStream("aac")},
			want:    true,
		},
		{
			name:    "probeが空なら変換する",
			ext:     ".mp4",
			streams: []ffprobeStream{},
			want:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := videoNeedsCompat(ffprobeResult{Streams: c.streams}, c.ext); got != c.want {
				t.Errorf("videoNeedsCompat = %v, want %v", got, c.want)
			}
		})
	}
}

// ffprobe の JSON はストリームごとに codec_name が codec_type より前に出る。
// 文字列検索で `"codec_type":"video"` を見つけてから後方の codec_name を探すと、
// 映像を読み飛ばして音声のコーデックを返してしまう。
// 2026-02の初版から一度も有効にならなかった旧実装はこの形だった。
func TestFFProbeResultReadsVideoCodecNotAudio(t *testing.T) {
	// 実際の ffprobe -show_streams の並び（codec_name が codec_type より前）
	raw := `{
	  "streams": [
	    {
	      "index": 0,
	      "codec_name": "h264",
	      "codec_long_name": "H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10",
	      "profile": "High",
	      "codec_type": "video",
	      "pix_fmt": "yuv420p"
	    },
	    {
	      "index": 1,
	      "codec_name": "aac",
	      "codec_long_name": "AAC (Advanced Audio Coding)",
	      "profile": "LC",
	      "codec_type": "audio"
	    }
	  ]
	}`

	result := ffprobeResult{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if got := result.primaryVideoCodec(); got != "h264" {
		t.Errorf("primaryVideoCodec = %q, want %q（音声のコーデックを拾っている）", got, "h264")
	}
	if videoNeedsCompat(result, ".mp4") {
		t.Error("H.264 + AAC の mp4 を変換しようとしている")
	}
}

func TestFFProbeResultPrimaryVideoCodecWithoutVideo(t *testing.T) {
	result := ffprobeResult{Streams: []ffprobeStream{audioStream("aac")}}
	if got := result.primaryVideoCodec(); got != "" {
		t.Errorf("primaryVideoCodec = %q, want empty", got)
	}
}
