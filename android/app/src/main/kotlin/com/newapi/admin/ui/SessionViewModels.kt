package com.newapi.admin.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.ViewModelStoreOwner

/**
 * Owns the screen ViewModels of one signed-in session. It lives in the Activity's store, so a
 * rotation keeps every screen's state, while a sign-out or account switch clears all ViewModels of
 * the old session, saved tab back stacks included (NavHost never clears those itself, and they would
 * keep polling the new server with the new credentials).
 */
internal class SessionViewModels : ViewModel(), ViewModelStoreOwner {
    private var sessionKey: Any? = null
    private val retired = mutableListOf<ViewModelStore>()

    override var viewModelStore = ViewModelStore()
        private set

    /** Switches to a fresh store when [key] changes. The old store is cleared by [clearRetired]. */
    fun bind(key: Any?) {
        if (key == sessionKey) return
        retired += viewModelStore
        viewModelStore = ViewModelStore()
        sessionKey = key
    }

    /** Clears the stores of ended sessions; call once their screens have left the composition. */
    fun clearRetired() {
        retired.forEach { it.clear() }
        retired.clear()
    }

    override fun onCleared() {
        clearRetired()
        viewModelStore.clear()
    }
}
