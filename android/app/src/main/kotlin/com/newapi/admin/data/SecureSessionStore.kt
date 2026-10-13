package com.newapi.admin.data

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Log
import com.newapi.admin.core.auth.SessionStore
import com.newapi.admin.core.auth.StoredAccount
import java.security.KeyStore
import java.util.Base64
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import kotlinx.serialization.json.Json

/**
 * [SessionStore] that keeps the signed-in account encrypted at rest.
 *
 * The account is serialized to JSON and sealed with AES-256-GCM under a non-exportable AndroidKeyStore
 * key; IV and ciphertext are stored Base64 in a private SharedPreferences file. Writes are synchronous
 * (`commit()`) because the auth layer saves a rotated refresh token from an OkHttp thread and the
 * previous token is revoked 30 s later. Anything that cannot be decrypted (key invalidated, data
 * restored from another device, format change) is wiped and reads as signed out.
 */
class SecureSessionStore(context: Context) : SessionStore {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val json = Json { ignoreUnknownKeys = true }

    private var loaded = false
    private var cached: StoredAccount? = null

    @Synchronized
    override fun load(): StoredAccount? {
        if (loaded) return cached
        cached = decryptStored()
        loaded = true
        return cached
    }

    @Synchronized
    override fun save(account: StoredAccount?) {
        cached = account
        loaded = true
        if (account == null) {
            prefs.edit().clear().commit()
            return
        }
        val plaintext = json.encodeToString(StoredAccount.serializer(), account).encodeToByteArray()
        val sealed = try {
            encrypt(plaintext)
        } catch (first: Exception) {
            // A broken or invalidated key: start over with a fresh one.
            Log.w(TAG, "Session encryption failed, regenerating key", first)
            deleteKey()
            try {
                encrypt(plaintext)
            } catch (second: Exception) {
                // Never leave an older (soon revoked) session on disk; this one lives in memory only.
                Log.w(TAG, "Session encryption failed again, not persisting", second)
                prefs.edit().clear().commit()
                return
            }
        }
        val written = prefs.edit()
            .putInt(KEY_VERSION, FORMAT_VERSION)
            .putString(KEY_IV, base64Encoder.encodeToString(sealed.iv))
            .putString(KEY_CIPHERTEXT, base64Encoder.encodeToString(sealed.ciphertext))
            .commit()
        if (!written) Log.w(TAG, "Session could not be written to disk")
    }

    private fun decryptStored(): StoredAccount? {
        val iv = prefs.getString(KEY_IV, null)
        val ciphertext = prefs.getString(KEY_CIPHERTEXT, null)
        if (iv == null && ciphertext == null) return null
        return try {
            if (iv == null || ciphertext == null || prefs.getInt(KEY_VERSION, 0) != FORMAT_VERSION) {
                error("Unknown session format")
            }
            val key = keyStore().getKey(KEY_ALIAS, null) as? SecretKey ?: error("Session key is missing")
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, key, GCMParameterSpec(GCM_TAG_BITS, base64Decoder.decode(iv)))
            val plaintext = cipher.doFinal(base64Decoder.decode(ciphertext))
            json.decodeFromString(StoredAccount.serializer(), plaintext.decodeToString())
        } catch (e: Exception) {
            Log.w(TAG, "Stored session is unreadable, discarding it", e)
            prefs.edit().clear().commit()
            deleteKey()
            null
        }
    }

    private fun encrypt(plaintext: ByteArray): Sealed {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        // The keystore picks a random 12-byte IV (randomized encryption is required for this key).
        cipher.init(Cipher.ENCRYPT_MODE, existingOrNewKey())
        return Sealed(iv = cipher.iv, ciphertext = cipher.doFinal(plaintext))
    }

    private fun existingOrNewKey(): SecretKey {
        (keyStore().getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
        generator.init(
            KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return generator.generateKey()
    }

    private fun deleteKey() {
        try {
            keyStore().deleteEntry(KEY_ALIAS)
        } catch (e: Exception) {
            Log.w(TAG, "Could not delete session key", e)
        }
    }

    private fun keyStore(): KeyStore = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }

    private class Sealed(val iv: ByteArray, val ciphertext: ByteArray)

    private companion object {
        const val TAG = "SecureSessionStore"
        const val PREFS_NAME = "secure_session"
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "newapi_session_v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val GCM_TAG_BITS = 128

        const val KEY_VERSION = "format_version"
        const val KEY_IV = "iv"
        const val KEY_CIPHERTEXT = "ciphertext"
        const val FORMAT_VERSION = 1

        val base64Encoder: Base64.Encoder = Base64.getEncoder()
        val base64Decoder: Base64.Decoder = Base64.getDecoder()
    }
}
