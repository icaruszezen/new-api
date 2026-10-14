package com.newapi.admin.core.update

import com.newapi.admin.core.net.ApiJson
import com.newapi.admin.core.net.await
import java.io.File
import java.io.IOException
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.ListSerializer
import okhttp3.OkHttpClient
import okhttp3.Request

/** An app build published as a GitHub release by the Android CI workflow. */
data class AppRelease(
    /** "0.2.15" or "0.2.14-beta". */
    val versionName: String,
    /** The APK's versionCode; it grows with every CI build, beta or stable. */
    val versionCode: Int,
    val isBeta: Boolean,
    /** Release notes as written on GitHub (Markdown source), possibly empty. */
    val notes: String,
    /** The release page, for downloading in a browser. */
    val pageUrl: String,
    val apkUrl: String,
    /** APK size in bytes as reported by GitHub. */
    val apkSize: Long,
    /** Lowercase hex SHA-256 of the APK, when GitHub reports a digest. */
    val apkSha256: String?,
)

/** Failures of [AppUpdater]; the app maps them to localised text. */
sealed class UpdateError(message: String, cause: Throwable? = null) : Exception(message, cause) {
    /** Connection, TLS or timeout failure talking to GitHub. */
    class Network(cause: IOException) : UpdateError(cause.message ?: cause.javaClass.simpleName, cause)

    /** GitHub's limit for unauthenticated API calls (60 per hour per IP) is used up. */
    class RateLimited(val resetAtEpochSeconds: Long?) : UpdateError("GitHub API rate limit exceeded")

    class Http(val status: Int) : UpdateError("HTTP $status")

    class InvalidResponse(detail: String, cause: Throwable? = null) : UpdateError(detail, cause)

    /** The downloaded APK's size or SHA-256 does not match the release. Nothing was kept. */
    class CorruptDownload(detail: String) : UpdateError(detail)

    /** The APK could not be written to the device, usually for lack of space. Nothing was kept. */
    class Storage(cause: IOException) : UpdateError(cause.message ?: cause.javaClass.simpleName, cause)
}

/**
 * Finds and downloads app updates from the GitHub releases of [repository] (`owner/name`).
 *
 * The Android CI workflow tags stable builds `android-v<major>.<minor>.<build>` and beta builds
 * `android-v<major>.<minor>.<build>-beta`, where `<build>` is the APK's versionCode. Other releases of
 * the repository (the server's) are ignored.
 */
class AppUpdater(
    private val repository: String,
    userAgent: String,
    okHttpBuilder: OkHttpClient.Builder = OkHttpClient.Builder(),
    private val apiBaseUrl: String = "https://api.github.com",
    private val clock: () -> Long = System::currentTimeMillis,
) {
    private val apiClient: OkHttpClient = okHttpBuilder
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .callTimeout(60, TimeUnit.SECONDS)
        .addInterceptor { chain -> chain.proceed(chain.request().newBuilder().header("User-Agent", userAgent).build()) }
        .build()

    // Release downloads redirect to GitHub's asset CDN and may be slow; only a stalled read fails them.
    private val downloadClient: OkHttpClient = apiClient.newBuilder()
        .readTimeout(60, TimeUnit.SECONDS)
        .callTimeout(0, TimeUnit.SECONDS)
        .build()

    /**
     * The newest release above [installedVersionCode], or null when the installed build is current.
     * Beta releases count only when [includeBeta] is set.
     */
    suspend fun findUpdate(installedVersionCode: Int, includeBeta: Boolean): AppRelease? {
        var best: AppRelease? = null
        for (page in 1..MAX_PAGES) {
            val releases = fetchPage(page)
            var sawStable = false
            for (release in releases.mapNotNull { it.toAppRelease() }) {
                if (!release.isBeta) sawStable = true
                if (release.isBeta && !includeBeta) continue
                if (best == null || release.versionCode > best.versionCode) best = release
            }
            // GitHub lists releases newest first by the date of the tagged commit. Stable builds tag
            // the merge commit, so for the stable channel every release after the first stable one is
            // older. Betas tag the pull request's head, which can be older than releases published
            // before the beta, so a beta check reads all [MAX_PAGES] pages.
            if ((sawStable && !includeBeta) || releases.size < PAGE_SIZE) break
        }
        return best?.takeIf { it.versionCode > installedVersionCode }
    }

    /**
     * Downloads [release]'s APK to [target]. The file appears only once its size and SHA-256 (when
     * GitHub reports one) match; a failed or cancelled download leaves nothing behind. [onProgress]
     * receives the bytes received so far and the expected total (0 when unknown).
     */
    suspend fun download(release: AppRelease, target: File, onProgress: (downloaded: Long, total: Long) -> Unit) {
        withContext(Dispatchers.IO) {
            val partial = File(target.path + ".part")
            val call = downloadClient.newCall(Request.Builder().url(release.apkUrl).build())
            // Cancelling the coroutine cancels the call, which unblocks a connect or read stuck on the network.
            val canceller = launch {
                try {
                    awaitCancellation()
                } finally {
                    call.cancel()
                }
            }
            try {
                target.parentFile?.mkdirs()
                val response = try {
                    call.execute()
                } catch (e: IOException) {
                    ensureActive()
                    throw UpdateError.Network(e)
                }
                response.use {
                    if (!response.isSuccessful) throw UpdateError.Http(response.code)
                    val body = response.body ?: throw UpdateError.InvalidResponse("download has no body")
                    val total = release.apkSize.takeIf { it > 0 } ?: body.contentLength().coerceAtLeast(0)
                    val digest = MessageDigest.getInstance("SHA-256")
                    var downloaded = 0L
                    body.byteStream().use { input ->
                        val output = try {
                            partial.outputStream()
                        } catch (e: IOException) {
                            throw UpdateError.Storage(e)
                        }
                        output.use {
                            val buffer = ByteArray(64 * 1024)
                            while (true) {
                                ensureActive()
                                val read = try {
                                    input.read(buffer)
                                } catch (e: IOException) {
                                    ensureActive()
                                    throw UpdateError.Network(e)
                                }
                                if (read < 0) break
                                downloaded += read
                                if (downloaded > MAX_APK_BYTES) throw UpdateError.CorruptDownload("download exceeds $MAX_APK_BYTES bytes")
                                try {
                                    output.write(buffer, 0, read)
                                } catch (e: IOException) {
                                    throw UpdateError.Storage(e)
                                }
                                digest.update(buffer, 0, read)
                                onProgress(downloaded, total)
                            }
                        }
                    }
                    if (release.apkSize > 0 && downloaded != release.apkSize) {
                        throw UpdateError.CorruptDownload("expected ${release.apkSize} bytes, got $downloaded")
                    }
                    val sha256 = digest.digest().joinToString("") { "%02x".format(it) }
                    if (release.apkSha256 != null && sha256 != release.apkSha256) {
                        throw UpdateError.CorruptDownload("SHA-256 mismatch")
                    }
                }
                target.delete()
                if (!partial.renameTo(target)) throw UpdateError.Storage(IOException("cannot move the download to $target"))
            } catch (e: Throwable) {
                partial.delete()
                throw e
            } finally {
                canceller.cancel()
            }
        }
    }

    private suspend fun fetchPage(page: Int): List<GitHubRelease> {
        val request = Request.Builder()
            .url("$apiBaseUrl/repos/$repository/releases?per_page=$PAGE_SIZE&page=$page")
            .header("Accept", "application/vnd.github+json")
            .header("X-GitHub-Api-Version", "2022-11-28")
            .build()
        val result = try {
            apiClient.await(request, clock)
        } catch (e: CancellationException) {
            throw e
        } catch (e: IOException) {
            throw UpdateError.Network(e)
        }
        if (!result.isSuccessful) {
            val exhausted = result.headers["x-ratelimit-remaining"] == "0"
            if (result.code == 429 || (result.code == 403 && exhausted)) {
                throw UpdateError.RateLimited(result.headers["x-ratelimit-reset"]?.toLongOrNull())
            }
            throw UpdateError.Http(result.code)
        }
        return try {
            ApiJson.decodeFromString(ListSerializer(GitHubRelease.serializer()), result.body)
        } catch (e: SerializationException) {
            throw UpdateError.InvalidResponse("unexpected releases response: ${e.message}", e)
        } catch (e: IllegalArgumentException) {
            throw UpdateError.InvalidResponse("unexpected releases response: ${e.message}", e)
        }
    }

    private companion object {
        const val PAGE_SIZE = 100
        const val MAX_PAGES = 2
        const val MAX_APK_BYTES = 100L * 1024 * 1024
    }
}

private val ReleaseTag = Regex("""android-v(\d{1,4}\.\d{1,4}\.(\d{1,9})(-beta)?)""")
private val HtmlComment = Regex("""<!--.*?-->""", RegexOption.DOT_MATCHES_ALL)
private val Sha256Digest = Regex("""sha256:([0-9a-f]{64})""")

@Serializable
internal data class GitHubRelease(
    @SerialName("tag_name") val tagName: String = "",
    val body: String? = null,
    val draft: Boolean = false,
    val prerelease: Boolean = false,
    @SerialName("html_url") val htmlUrl: String = "",
    val assets: List<GitHubAsset> = emptyList(),
) {
    /** The installable build this release carries, or null for anything else (server releases, drafts, no APK). */
    fun toAppRelease(): AppRelease? {
        if (draft) return null
        val match = ReleaseTag.matchEntire(tagName) ?: return null
        val versionCode = match.groupValues[2].toIntOrNull()?.takeIf { it > 0 } ?: return null
        val apk = assets.firstOrNull { it.name.endsWith(".apk") && it.state == "uploaded" && it.downloadUrl.isNotBlank() }
            ?: return null
        return AppRelease(
            versionName = match.groupValues[1],
            versionCode = versionCode,
            // A stable tag marked pre-release on GitHub was pulled back to testing; treat it as beta.
            isBeta = prerelease || match.groupValues[3].isNotEmpty(),
            notes = body.orEmpty().replace(HtmlComment, "").trim(),
            pageUrl = htmlUrl,
            apkUrl = apk.downloadUrl,
            apkSize = apk.size,
            apkSha256 = apk.digest?.lowercase()?.let { Sha256Digest.matchEntire(it)?.groupValues?.get(1) },
        )
    }
}

@Serializable
internal data class GitHubAsset(
    val name: String = "",
    val size: Long = 0,
    val state: String = "",
    val digest: String? = null,
    @SerialName("browser_download_url") val downloadUrl: String = "",
)
