package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import android.util.Log
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * SharedPreferencesへ保存する秘密値の暗号化・復号を担う。
 *
 * SharedPreferencesはMODE_PRIVATEでもroot化された端末では読めてしまうため、
 * password_sha256 と session_id は保存前に暗号化する。
 * gkillでは password_sha256 自体が認証情報としてそのまま送られるので、
 * 平文パスワードと同等に扱う必要がある。
 */
interface GkillSecretCipher {
    /** 平文を保存用の文字列に変換する。失敗した場合はnullを返す。 */
    fun encrypt(plain: String): String?

    /** 保存用の文字列を平文に戻す。失敗した場合はnullを返す。 */
    fun decrypt(stored: String): String?

    /** [stored] がこの実装で暗号化された値かどうか。旧バージョンが書いた平文と区別する。 */
    fun isEncrypted(stored: String): Boolean
}

/**
 * Android Keystoreの非エクスポータブルな鍵でAES/GCM暗号化する既定の実装。
 *
 * 鍵はKeystore内から取り出せないため、/dataを直接読まれても平文は復元できない。
 */
class AndroidKeystoreSecretCipher : GkillSecretCipher {

    private companion object {
        const val TAG = "gkill"
        const val KEY_ALIAS = "gkill_wear_credential_key"
        const val KEYSTORE_TYPE = "AndroidKeyStore"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val GCM_TAG_BITS = 128
        const val IV_LENGTH = 12

        /** 暗号化済みであることを示す目印。これが無い値は旧バージョンが書いた平文とみなす。 */
        const val PREFIX = "enc1:"
    }

    override fun isEncrypted(stored: String): Boolean = stored.startsWith(PREFIX)

    override fun encrypt(plain: String): String? {
        return try {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.ENCRYPT_MODE, secretKey())
            val encrypted = cipher.doFinal(plain.toByteArray(Charsets.UTF_8))
            PREFIX + Base64.encodeToString(cipher.iv + encrypted, Base64.NO_WRAP)
        } catch (e: Exception) {
            Log.w(TAG, "認証情報の暗号化に失敗しました", e)
            null
        }
    }

    override fun decrypt(stored: String): String? {
        return try {
            val raw = Base64.decode(stored.removePrefix(PREFIX), Base64.NO_WRAP)
            if (raw.size <= IV_LENGTH) {
                return null
            }
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(
                Cipher.DECRYPT_MODE,
                secretKey(),
                GCMParameterSpec(GCM_TAG_BITS, raw, 0, IV_LENGTH)
            )
            String(cipher.doFinal(raw, IV_LENGTH, raw.size - IV_LENGTH), Charsets.UTF_8)
        } catch (e: Exception) {
            Log.w(TAG, "認証情報の復号に失敗しました", e)
            null
        }
    }

    private fun secretKey(): SecretKey {
        val keyStore = KeyStore.getInstance(KEYSTORE_TYPE).apply { load(null) }
        val existing = keyStore.getEntry(KEY_ALIAS, null) as? KeyStore.SecretKeyEntry
        if (existing != null) {
            return existing.secretKey
        }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE_TYPE)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .build()
        )
        return generator.generateKey()
    }
}
