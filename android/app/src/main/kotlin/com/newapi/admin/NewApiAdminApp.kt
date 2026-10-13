package com.newapi.admin

import android.app.Application
import android.os.Build
import com.newapi.admin.data.SecureSessionStore
import com.newapi.admin.data.SharedPrefsAppPreferences

class NewApiAdminApp : Application() {
    /** Created once in [onCreate]; the activity hands it to Compose through [LocalAppContainer]. */
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        // The server shows this per session in the web session list. OkHttp rejects non-ASCII header
        // values, and Build.MODEL is vendor text, so keep printable ASCII only.
        val userAgent = "NewApiAdmin/${BuildConfig.VERSION_NAME} (Android ${Build.VERSION.RELEASE}; ${Build.MODEL})"
            .filter { it in ' '..'~' }
        container = AppContainer(
            sessionStore = SecureSessionStore(this),
            preferences = SharedPrefsAppPreferences(this),
            userAgent = userAgent,
            appVersion = BuildConfig.VERSION_NAME,
        )
    }
}
