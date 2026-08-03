package account

import (
	"strings"
	"testing"
	"time"
)

const testCredential = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("hash = %q, want it to start with $argon2id$v=19$", hash)
	}
	if strings.Contains(hash, testCredential) {
		t.Error("hash must not contain the credential itself")
	}

	ok, err := VerifyPassword(hash, testCredential)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Error("VerifyPassword returned false for the correct credential")
	}
}

func TestVerifyPasswordRejectsWrongCredential(t *testing.T) {
	hash, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	for _, wrong := range []string{
		"",
		"wrong",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b854", // 1文字違い
		strings.ToUpper(testCredential),
	} {
		ok, err := VerifyPassword(hash, wrong)
		if err != nil {
			t.Fatalf("VerifyPassword(%q) failed: %v", wrong, err)
		}
		if ok {
			t.Errorf("VerifyPassword(%q) returned true, want false", wrong)
		}
	}
}

func TestHashPasswordUsesDifferentSaltEachTime(t *testing.T) {
	first, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	second, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if first == second {
		t.Error("同じ資格情報から同じハッシュが出た。ソルトが効いていない")
	}

	// ソルトが違っても、どちらも同じ資格情報で照合できること
	for _, hash := range []string{first, second} {
		ok, err := VerifyPassword(hash, testCredential)
		if err != nil {
			t.Fatalf("VerifyPassword failed: %v", err)
		}
		if !ok {
			t.Error("VerifyPassword returned false for the correct credential")
		}
	}
}

func TestVerifyPasswordDetectsTamperedHash(t *testing.T) {
	hash, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// ハッシュ部分の末尾1文字を書き換える
	tampered := []rune(hash)
	last := len(tampered) - 1
	if tampered[last] == 'A' {
		tampered[last] = 'B'
	} else {
		tampered[last] = 'A'
	}

	ok, err := VerifyPassword(string(tampered), testCredential)
	if err == nil && ok {
		t.Error("改竄されたハッシュで照合が通ってしまった")
	}
}

func TestVerifyPasswordRejectsMalformedStored(t *testing.T) {
	// 旧形式 (無塩SHA-256をそのまま入れていたもの) を含め、
	// PHC文字列でない保存値はエラーにする。誤って通してはいけない
	for _, stored := range []string{
		"",
		testCredential,
		"$argon2id$",
		"$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=1$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$!!!$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$",
		// argon2.IDKey が panic するパラメータ。errorで返ること
		"$argon2id$v=19$m=65536,t=0,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=0$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=0,t=3,p=4$c2FsdA$aGFzaA",
	} {
		ok, err := VerifyPassword(stored, testCredential)
		if ok {
			t.Errorf("VerifyPassword(%q) returned true, want false", stored)
		}
		if err == nil {
			t.Errorf("VerifyPassword(%q) returned no error, want an error", stored)
		}
	}
}

func TestIsValidCredentialFormat(t *testing.T) {
	valid := []string{
		testCredential,
		"0000000000000000000000000000000000000000000000000000000000000000",
	}
	invalid := []string{
		"",
		"abc",
		strings.ToUpper(testCredential),                                     // 大文字は受け付けない
		testCredential + "0",                                                // 長すぎる
		testCredential[:63],                                                 // 短すぎる
		"g3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",  // hexでない
		" e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // 前後の空白
	}

	for _, credential := range valid {
		if !IsValidCredentialFormat(credential) {
			t.Errorf("IsValidCredentialFormat(%q) = false, want true", credential)
		}
	}
	for _, credential := range invalid {
		if IsValidCredentialFormat(credential) {
			t.Errorf("IsValidCredentialFormat(%q) = true, want false", credential)
		}
	}
}

func TestIsValidUserID(t *testing.T) {
	valid := []string{
		"admin",
		"e2e_user",
		"myuser_auto_DESKTOP-ABC",
		"user.name",
		"a",
	}
	// ユーザIDはキャッシュディレクトリ名になるので、
	// パスとして解釈されうるものは通してはいけない
	invalid := []string{
		"",
		"..",
		".",
		"../admin",
		"a/b",
		`a\b`,
		"a..b",
		".hidden",
		"_leading",
		"-leading",
		"c:admin",
		"admin ",
		"ユーザ",
		strings.Repeat("a", 65),
	}

	for _, userID := range valid {
		if !IsValidUserID(userID) {
			t.Errorf("IsValidUserID(%q) = false, want true", userID)
		}
	}
	for _, userID := range invalid {
		if IsValidUserID(userID) {
			t.Errorf("IsValidUserID(%q) = true, want false", userID)
		}
	}
}

func TestAccountVerifyPasswordIsFailClosed(t *testing.T) {
	// パスワード未設定のアカウントは、どんな入力でもログインさせない。
	// 旧実装は「保存値がnilなら比較をスキップ」していたので、何を送っても通ってしまった
	empty := ""
	for _, target := range []*Account{
		{UserID: "u"},
		{UserID: "u", PasswordHash: &empty},
	} {
		for _, credential := range []string{"", testCredential, "anything"} {
			ok, err := target.VerifyPassword(credential)
			if err != nil {
				t.Fatalf("VerifyPassword failed: %v", err)
			}
			if ok {
				t.Errorf("パスワード未設定のアカウントが credential=%q でログインできてしまった", credential)
			}
		}
	}
}

func TestAccountVerifyPasswordRejectsOverlongCredential(t *testing.T) {
	hash, err := HashPassword(testCredential)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	target := &Account{UserID: "u", PasswordHash: &hash}

	ok, err := target.VerifyPassword(strings.Repeat("a", CredentialMaxLength+1))
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if ok {
		t.Error("長すぎる資格情報が通ってしまった")
	}
}

func TestIsPasswordResetTokenValid(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	token := "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	t.Run("トークンが無いアカウントは常に不可", func(t *testing.T) {
		target := &Account{UserID: "u"}
		if target.IsPasswordResetTokenValid(token, now) {
			t.Error("トークン未発行なのに通ってしまった")
		}
		if target.IsPasswordResetTokenValid("", now) {
			t.Error("空文字が通ってしまった")
		}
	})

	t.Run("一致していて期限内なら可", func(t *testing.T) {
		target := &Account{UserID: "u", PasswordResetToken: &token, PasswordResetTokenExpiration: &future}
		if !target.IsPasswordResetTokenValid(token, now) {
			t.Error("有効なトークンが弾かれた")
		}
	})

	t.Run("期限切れは不可", func(t *testing.T) {
		target := &Account{UserID: "u", PasswordResetToken: &token, PasswordResetTokenExpiration: &past}
		if target.IsPasswordResetTokenValid(token, now) {
			t.Error("期限切れのトークンが通ってしまった")
		}
	})

	t.Run("不一致は不可", func(t *testing.T) {
		target := &Account{UserID: "u", PasswordResetToken: &token, PasswordResetTokenExpiration: &future}
		for _, wrong := range []string{"", "3f2504e0-4f89-11d3-9a0c-0305e82c3302", token + "x", token[:len(token)-1]} {
			if target.IsPasswordResetTokenValid(wrong, now) {
				t.Errorf("不一致のトークン %q が通ってしまった", wrong)
			}
		}
	})

	t.Run("期限が未設定なら期限判定はしない", func(t *testing.T) {
		target := &Account{UserID: "u", PasswordResetToken: &token}
		if !target.IsPasswordResetTokenValid(token, now) {
			t.Error("期限未設定のトークンが弾かれた")
		}
	})
}
