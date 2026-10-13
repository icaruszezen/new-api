package com.newapi.admin.data

/** Small non-secret settings that survive sign-out (used to prefill the login form). */
interface AppPreferences {
    var lastServerUrl: String
    var lastUsername: String
}

class InMemoryAppPreferences : AppPreferences {
    override var lastServerUrl: String = ""
    override var lastUsername: String = ""
}
