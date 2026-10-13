package com.newapi.admin.data

import android.content.Context

/** [AppPreferences] in a plain private SharedPreferences file (nothing secret is stored here). */
class SharedPrefsAppPreferences(context: Context) : AppPreferences {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override var lastServerUrl: String
        get() = prefs.getString(KEY_LAST_SERVER_URL, null).orEmpty()
        set(value) = prefs.edit().putString(KEY_LAST_SERVER_URL, value).apply()

    override var lastUsername: String
        get() = prefs.getString(KEY_LAST_USERNAME, null).orEmpty()
        set(value) = prefs.edit().putString(KEY_LAST_USERNAME, value).apply()

    private companion object {
        const val PREFS_NAME = "app_preferences"
        const val KEY_LAST_SERVER_URL = "last_server_url"
        const val KEY_LAST_USERNAME = "last_username"
    }
}
