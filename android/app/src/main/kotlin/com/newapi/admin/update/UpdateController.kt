package com.newapi.admin.update

import com.newapi.admin.R
import com.newapi.admin.core.update.AppRelease
import com.newapi.admin.core.update.AppUpdater
import com.newapi.admin.data.AppPreferences
import com.newapi.admin.ui.common.UiMessage
import java.io.File
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** Download of [UpdateState.available]. */
sealed interface UpdateDownload {
    /** [totalBytes] is 0 when unknown. */
    data class Running(val downloadedBytes: Long, val totalBytes: Long) : UpdateDownload

    data class Ready(val apk: File) : UpdateDownload

    data class Failed(val error: Throwable) : UpdateDownload
}

data class UpdateState(
    val isChecking: Boolean = false,
    /** The newest build above the installed one, from the last successful check. */
    val available: AppRelease? = null,
    /** Device time of the last successful check, 0 when never. */
    val lastCheckedMillis: Long = 0,
    val includeBeta: Boolean = false,
    /** The update dialog is open. */
    val promptVisible: Boolean = false,
    val download: UpdateDownload? = null,
)

/**
 * App-wide update checks against the GitHub releases the Android CI publishes. [updater] is null for
 * builds that cannot update themselves (local builds, builds not signed with the release key); then
 * every action is a no-op and [isSupported] is false.
 */
class UpdateController(
    private val updater: AppUpdater?,
    private val preferences: AppPreferences,
    private val installedVersionCode: Int,
    /** Private cache directory the APKs are downloaded to; emptied on every app start. */
    private val downloadDir: File,
    private val scope: CoroutineScope,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    val isSupported: Boolean get() = updater != null

    private val mutableState = MutableStateFlow(
        UpdateState(lastCheckedMillis = preferences.lastUpdateCheckMillis, includeBeta = preferences.includeBetaUpdates),
    )
    val state: StateFlow<UpdateState> = mutableState.asStateFlow()

    private val messageChannel = Channel<UiMessage>(Channel.BUFFERED)

    /** Outcome of checks the user asked for: "up to date" or the error. */
    val messages: Flow<UiMessage> = messageChannel.receiveAsFlow()

    private var checkJob: Job? = null
    private var downloadJob: Job? = null

    /** Clears the downloads of earlier runs; called once per process. */
    fun onAppStart() {
        // Downloads run in this process only, so anything left over is from a run that has ended.
        scope.launch(Dispatchers.IO) { downloadDir.listFiles()?.forEach { it.delete() } }
    }

    /** The app came to the foreground: checks quietly, at most once per [AUTO_CHECK_INTERVAL_MILLIS]. */
    fun onForeground() {
        if (checkJob?.isActive == true) return
        val sinceLastCheck = clock() - preferences.lastUpdateCheckMillis
        if (sinceLastCheck in 0 until AUTO_CHECK_INTERVAL_MILLIS) return
        check(report = false, prompt = true)
    }

    /** "Check for updates": opens the dialog when there is an update and reports when there is none. */
    fun checkNow() = check(report = true, prompt = true)

    fun setIncludeBeta(include: Boolean) {
        if (include == mutableState.value.includeBeta) return
        preferences.includeBetaUpdates = include
        mutableState.update { state ->
            // A beta that is already downloading or downloaded stays; the user chose to install it.
            val dropBeta = !include && state.available?.isBeta == true && state.download == null
            state.copy(
                includeBeta = include,
                available = if (dropBeta) null else state.available,
                promptVisible = state.promptVisible && !dropBeta,
            )
        }
        // Opting in is a request to see betas now; opting out only refreshes what Settings shows.
        check(report = include, prompt = include)
    }

    /** Closes the dialog. A running download keeps the dialog open; cancel it with [cancelDownload]. */
    fun dismissPrompt() {
        mutableState.update { state ->
            if (state.download is UpdateDownload.Running) return@update state
            state.copy(promptVisible = false, download = state.download as? UpdateDownload.Ready)
        }
    }

    fun startDownload() {
        val updater = updater ?: return
        val state = mutableState.value
        val release = state.available ?: return
        if (state.download is UpdateDownload.Running) return
        mutableState.update { it.copy(download = UpdateDownload.Running(0, release.apkSize)) }
        downloadJob = scope.launch {
            val apk = File(downloadDir, "new-api-admin-${release.versionCode}.apk")
            try {
                updater.download(release, apk) { downloaded, total ->
                    mutableState.update { current ->
                        if (current.download is UpdateDownload.Running) current.copy(download = UpdateDownload.Running(downloaded, total)) else current
                    }
                }
                mutableState.update { it.copy(download = UpdateDownload.Ready(apk)) }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(download = UpdateDownload.Failed(e)) }
            }
        }
    }

    fun cancelDownload() {
        downloadJob?.cancel()
        downloadJob = null
        mutableState.update { it.copy(download = null) }
    }

    /** The downloaded APK is gone (the system cleared the cache); offer the download again. */
    fun forgetDownload() {
        mutableState.update { if (it.download is UpdateDownload.Ready) it.copy(download = null) else it }
    }

    /** [report] sends the outcome to [messages]; [prompt] lets the result open the dialog. */
    private fun check(report: Boolean, prompt: Boolean) {
        val updater = updater ?: return
        checkJob?.cancel()
        mutableState.update { it.copy(isChecking = true) }
        checkJob = scope.launch {
            try {
                val release = updater.findUpdate(installedVersionCode, preferences.includeBetaUpdates)
                val now = clock()
                preferences.lastUpdateCheckMillis = now
                mutableState.update { state ->
                    // The build being downloaded, or downloaded and still current, stays until the user is done with it.
                    val keepDownload = when (state.download) {
                        is UpdateDownload.Running -> true
                        is UpdateDownload.Ready -> release != null && release.versionCode == state.available?.versionCode
                        else -> false
                    }
                    if (keepDownload) {
                        return@update state.copy(isChecking = false, lastCheckedMillis = now, promptVisible = state.promptVisible || (prompt && report))
                    }
                    // A quiet check opens the dialog only for a build the user has not been shown yet.
                    val isNew = release != null && release.versionCode != state.available?.versionCode
                    state.copy(
                        isChecking = false,
                        lastCheckedMillis = now,
                        available = release,
                        download = null,
                        promptVisible = release != null && (state.promptVisible || (prompt && (report || isNew))),
                    )
                }
                if (report && release == null) messageChannel.send(UiMessage.Res(R.string.update_up_to_date))
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                mutableState.update { it.copy(isChecking = false) }
                if (report) messageChannel.send(UiMessage.Error(e))
            }
        }
    }

    private companion object {
        const val AUTO_CHECK_INTERVAL_MILLIS = 6L * 60 * 60 * 1000
    }
}
