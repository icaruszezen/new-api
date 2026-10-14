package com.newapi.admin.data

/** Small non-secret settings that survive sign-out (login form prefill, update checks). */
interface AppPreferences {
    var lastServerUrl: String
    var lastUsername: String

    /** The user opted in to beta builds. */
    var includeBetaUpdates: Boolean

    /** Device time of the last successful update check, 0 when never. */
    var lastUpdateCheckMillis: Long
}

class InMemoryAppPreferences : AppPreferences {
    override var lastServerUrl: String = ""
    override var lastUsername: String = ""
    override var includeBetaUpdates: Boolean = false
    override var lastUpdateCheckMillis: Long = 0
}
