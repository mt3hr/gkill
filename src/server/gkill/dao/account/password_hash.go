package account

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
)

// パスワードの保存形式について
//
// クライアントは平文パスワードではなく、そのSHA-256を64桁hexにしたものを送ってくる
// (ワイヤ形式 password_sha256 / new_password_sha256)。
// そのhex文字列を「資格情報」としてそのまま扱い、サーバ側でArgon2idをかけて保存する。
// つまり保存値は Argon2id(SHA-256(平文パスワード)) になる。
//
// 前段のSHA-256はArgon2idの強度を下げない。攻撃者はオフライン総当たりでも
// 候補ごとに SHA-256 (安価) → Argon2id (高価) を計算する必要があり、コストはArgon2idが支配する。
// 加えて、Argon2idへの入力が64桁hex固定になるので長大入力によるDoSも塞がる。

const (
	// argon2idMemory はArgon2idのメモリコスト(KiB)。RFC 9106の第2推奨パラメータ。
	argon2idMemory = 64 * 1024
	// argon2idTime はArgon2idの反復回数。
	argon2idTime = 3
	// argon2idParallelism はArgon2idの並列度(レーン数)。
	argon2idParallelism = 4
	// argon2idSaltLength はソルトのバイト長。
	argon2idSaltLength = 16
	// argon2idKeyLength は導出する鍵のバイト長。
	argon2idKeyLength = 32
)

// CredentialMaxLength は受け付ける資格情報文字列の最大長。
// 正しいクライアントは64桁hexしか送ってこないので、これを超えるものは
// 照合するまでもなく捨てる (長大なリクエストボディに対する保険)。
const CredentialMaxLength = 256

// credentialPattern はクライアントが送ってくる資格情報 (平文パスワードのSHA-256を
// 小文字hexにしたもの) の形式。
var credentialPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// userIDPattern は受け付けるユーザIDの形式。
// ユーザIDはキャッシュディレクトリ名 (caches/zip_cache/{userID}/,
// caches/plugin_cache/{userID}/ など) としてそのまま使われるので、
// パス区切りや上位ディレクトリ参照になりうる文字を最初から通さない。
var userIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

// IsValidUserID はユーザIDとして受け付けられる文字列かを返す。
// 既存アカウントには適用しない (新規作成時のみ検査する)。
func IsValidUserID(userID string) bool {
	if !userIDPattern.MatchString(userID) {
		return false
	}
	// 先頭が英数字に限定されているので ".." そのものは弾かれるが、
	// 途中に現れる場合も念のため拒否する
	return !strings.Contains(userID, "..")
}

// IsValidCredentialFormat は資格情報がワイヤ形式 (64桁の小文字hex) に沿っているかを返す。
// パスワードを新しく設定するときだけ使う。ログイン時は形式で早期に弾かず、
// 常に同じ照合経路を通す (形式の違いが応答から読み取れないようにするため)。
func IsValidCredentialFormat(credential string) bool {
	return credentialPattern.MatchString(credential)
}

// HashPassword は資格情報をArgon2idでハッシュ化し、PHC文字列を返す。
//
//	$argon2id$v=19$m=65536,t=3,p=4$<base64ソルト>$<base64ハッシュ>
//
// パラメータを保存値自身に持たせているので、後からコストを変えても
// 既存の保存値はそのまま照合できる。
func HashPassword(credential string) (string, error) {
	salt := make([]byte, argon2idSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("error at generate salt for password hash: %w", err)
	}

	hash := argon2.IDKey([]byte(credential), salt, argon2idTime, argon2idMemory, argon2idParallelism, argon2idKeyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2idMemory,
		argon2idTime,
		argon2idParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword は保存済みのPHC文字列と資格情報を照合する。
// 保存値が壊れている場合はerrorを返す。照合結果が偽のときはerrorではなくfalseを返す。
func VerifyPassword(stored string, credential string) (bool, error) {
	memory, time, parallelism, salt, wantHash, err := parsePHC(stored)
	if err != nil {
		return false, err
	}

	gotHash := argon2.IDKey([]byte(credential), salt, time, memory, parallelism, uint32(len(wantHash)))

	return subtle.ConstantTimeCompare(gotHash, wantHash) == 1, nil
}

// constantTimeEquals は2つの文字列を実行時間が内容に依存しない形で比較する。
// リセットトークンのように総当たりされうる秘密値の照合に使う。
func constantTimeEquals(a string, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// parsePHC はPHC文字列からArgon2idのパラメータ、ソルト、ハッシュを取り出す。
func parsePHC(stored string) (memory uint32, time uint32, parallelism uint8, salt []byte, hash []byte, err error) {
	// $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash> を分割すると
	// 先頭の空文字を含めて6要素になる
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[0] != "" {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash: invalid format")
	}

	if parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash: unsupported algorithm %q", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash version: %w", err)
	}
	if version != argon2.Version {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash: unsupported version %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &parallelism); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash params: %w", err)
	}
	// argon2.IDKey は 0 を渡すとpanicする。
	// DBが壊れていてもログインハンドラを巻き込まないよう、ここで弾いておく
	if memory < 8 || time < 1 || parallelism < 1 {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash params: out of range m=%d,t=%d,p=%d", memory, time, parallelism)
	}

	// Strict() で復号する。base64は末尾文字に余剰ビット(データを表さない下位ビット)を
	// 持ちうるが、非Strictだとそこが書き換えられていても黙って捨ててしまい、
	// 改竄された保存値が元と同じバイト列に復号されてしまう。
	// HashPassword の EncodeToString は常に余剰ビット0の正規形を出すので、
	// 正規に作られた保存値がこれで弾かれることはない。
	salt, err = base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at decode password hash salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at decode password hash: %w", err)
	}
	if len(hash) == 0 {
		return 0, 0, 0, nil, nil, fmt.Errorf("error at parse password hash: empty hash")
	}

	return memory, time, parallelism, salt, hash, nil
}
