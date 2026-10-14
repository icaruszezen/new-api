package com.newapi.admin

import androidx.compose.runtime.staticCompositionLocalOf
import com.newapi.admin.core.api.AdminApi
import com.newapi.admin.core.auth.AuthManager
import com.newapi.admin.core.auth.AuthState
import com.newapi.admin.core.auth.SessionStore
import com.newapi.admin.core.format.QuotaFormatter
import com.newapi.admin.core.model.SelfUser
import com.newapi.admin.core.model.SystemStatus
import com.newapi.admin.core.net.NewApiClient
import com.newapi.admin.core.update.AppUpdater
import com.newapi.admin.data.AppPreferences
import com.newapi.admin.update.UpdateController
import java.io.File
import java.util.Locale
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/** App-wide singletons. Created once by the Application; screens reach it through [LocalAppContainer]. */
class AppContainer(
    sessionStore: SessionStore,
    val preferences: AppPreferences,
    userAgent: String,
    /** versionName and versionCode, shown on the settings screen. */
    val appVersion: String,
    val appVersionCode: Int,
    /** Null for builds that cannot update themselves (see [UpdateController]). */
    updater: AppUpdater?,
    /** Private cache directory for downloaded updates. */
    updateDir: File,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val client = NewApiClient(
        store = sessionStore,
        userAgent = userAgent,
        acceptLanguage = { Locale.getDefault().toLanguageTag() },
    )

    val auth: AuthManager get() = client.auth
    val api: AdminApi get() = client.admin

    /** Signed-in admin, or null. */
    val currentUser: StateFlow<SelfUser?> = auth.state
        .map { (it as? AuthState.SignedIn)?.account?.user }
        .stateIn(scope, SharingStarted.Eagerly, (auth.state.value as? AuthState.SignedIn)?.account?.user)

    private val statusFlow = MutableStateFlow<SystemStatus?>(null)

    /** `/api/status` of the signed-in server (display currency, system name, version). */
    val status: StateFlow<SystemStatus?> = statusFlow.asStateFlow()

    val quotaFormatter: StateFlow<QuotaFormatter> = statusFlow
        .map { status -> status?.let(::QuotaFormatter) ?: QuotaFormatter.Default }
        .stateIn(scope, SharingStarted.Eagerly, QuotaFormatter.Default)

    val events = AppEvents()

    val updates = UpdateController(updater, preferences, appVersionCode, updateDir, scope)

    init {
        updates.onAppStart()
        scope.launch {
            auth.state
                .map { (it as? AuthState.SignedIn)?.account?.baseUrl }
                .distinctUntilChanged()
                .collect { baseUrl ->
                    statusFlow.value = null
                    if (baseUrl != null) refreshStatus()
                }
        }
    }

    fun refreshStatus() {
        scope.launch {
            runCatching { api.status() }.onSuccess { statusFlow.value = it }
        }
    }
}

/**
 * Lets list screens catch up after a detail/edit screen changed something. The payload is the id of
 * the changed (or deleted) row, or null when the list should reload from the top (a row was created).
 */
class AppEvents {
    private val users = MutableSharedFlow<Int?>(extraBufferCapacity = 8)
    private val channels = MutableSharedFlow<Int?>(extraBufferCapacity = 8)
    private val redemptions = MutableSharedFlow<Int?>(extraBufferCapacity = 8)

    val usersChanged: SharedFlow<Int?> = users.asSharedFlow()
    val channelsChanged: SharedFlow<Int?> = channels.asSharedFlow()
    val redemptionsChanged: SharedFlow<Int?> = redemptions.asSharedFlow()

    fun notifyUsersChanged(id: Int? = null) {
        users.tryEmit(id)
    }

    fun notifyChannelsChanged(id: Int? = null) {
        channels.tryEmit(id)
    }

    fun notifyRedemptionsChanged(id: Int? = null) {
        redemptions.tryEmit(id)
    }
}

val LocalAppContainer = staticCompositionLocalOf<AppContainer> { error("AppContainer not provided") }
