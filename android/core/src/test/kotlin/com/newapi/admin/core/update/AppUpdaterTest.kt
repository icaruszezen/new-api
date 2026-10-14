package com.newapi.admin.core.update

import com.newapi.admin.core.testing.FakeServer
import com.newapi.admin.core.testing.jsonResponse
import java.io.File
import java.nio.file.Files
import java.security.MessageDigest
import kotlinx.coroutines.runBlocking
import okhttp3.mockwebserver.MockResponse
import okio.Buffer
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test

class AppUpdaterTest {
    private val server = FakeServer()
    private val updater = AppUpdater(
        repository = "owner/repo",
        userAgent = "NewApiAdmin/test",
        apiBaseUrl = server.baseUrl,
    )
    private val workDir: File = Files.createTempDirectory("updater").toFile()

    @After
    fun tearDown() {
        server.close()
        workDir.deleteRecursively()
    }

    private fun asset(name: String = "new-api-admin.apk", size: Long = 2_000_000, digest: String? = null, state: String = "uploaded") =
        """{"name":"$name","size":$size,"state":"$state",${digest?.let { "\"digest\":\"$it\"," } ?: ""}
            "browser_download_url":"${server.baseUrl}/download/$name"}"""

    private fun release(tag: String, prerelease: Boolean = false, draft: Boolean = false, body: String = "", assets: List<String> = listOf(asset())) =
        """{"tag_name":"$tag","prerelease":$prerelease,"draft":$draft,"body":"$body",
            "html_url":"https://github.com/owner/repo/releases/tag/$tag","assets":[${assets.joinToString(",")}]}"""

    private fun servePages(vararg pages: List<String>) {
        server.handler = { request ->
            val page = request.query("page")!!.toInt()
            jsonResponse("[" + pages.getOrElse(page - 1) { emptyList() }.joinToString(",") + "]")
        }
    }

    @Test
    fun stableChannelPicksTheNewestStableBuildAndSkipsServerReleasesAndBetas() = runBlocking {
        servePages(
            listOf(
                release("android-v0.2.21-beta", prerelease = true),
                release("v1.0.0-rc.40"),
                release("android-v0.2.20", body = "<!-- generated -->\\nAdd update checks (#12)\\n"),
                release("android-v0.2.18"),
            ),
        )

        val update = updater.findUpdate(installedVersionCode = 14, includeBeta = false)!!

        assertEquals("0.2.20", update.versionName)
        assertEquals(20, update.versionCode)
        assertFalse(update.isBeta)
        assertEquals("Add update checks (#12)", update.notes)
        assertEquals("https://github.com/owner/repo/releases/tag/android-v0.2.20", update.pageUrl)
        assertEquals("${server.baseUrl}/download/new-api-admin.apk", update.apkUrl)
        assertEquals(2_000_000, update.apkSize)
        val request = server.requests.single()
        assertEquals("/repos/owner/repo/releases", request.path)
        assertEquals("NewApiAdmin/test", request.header("User-Agent"))
        assertEquals("application/vnd.github+json", request.header("Accept"))
    }

    @Test
    fun betaChannelTakesWhicheverBuildIsNewer() = runBlocking {
        servePages(listOf(release("android-v0.2.21-beta", prerelease = true), release("android-v0.2.20")))
        assertEquals("0.2.21-beta", updater.findUpdate(14, includeBeta = true)!!.versionName)

        servePages(listOf(release("android-v0.2.22"), release("android-v0.2.21-beta", prerelease = true)))
        val update = updater.findUpdate(14, includeBeta = true)!!
        assertEquals("0.2.22", update.versionName)
        assertFalse(update.isBeta)
    }

    @Test
    fun noUpdateWhenTheInstalledBuildIsCurrentOrNewer() = runBlocking {
        servePages(listOf(release("android-v0.2.20")))
        assertNull(updater.findUpdate(20, includeBeta = false))
        // An installed beta newer than the last stable build is not downgraded.
        assertNull(updater.findUpdate(21, includeBeta = false))
    }

    @Test
    fun releasesThatCannotBeInstalledAreIgnored() = runBlocking {
        servePages(
            listOf(
                release("android-v0.2.30", draft = true),
                release("android-v0.2.29", assets = emptyList()),
                release("android-v0.2.28", assets = listOf(asset(state = "open"))),
                release("android-v0.2.27", assets = listOf(asset(name = "checksums.txt"))),
                release("android-v0.2", assets = listOf(asset())),
                release("android-v0.2.0x"),
                release("android-v0.2.26-rc"),
                release("android-v0.2.25"),
            ),
        )
        assertEquals(25, updater.findUpdate(1, includeBeta = true)!!.versionCode)
    }

    @Test
    fun aStableTagMarkedPreReleaseIsOnlyOfferedToBetaUsers() = runBlocking {
        servePages(listOf(release("android-v0.2.21", prerelease = true), release("android-v0.2.20")))
        assertEquals(20, updater.findUpdate(1, includeBeta = false)!!.versionCode)
        assertTrue(updater.findUpdate(1, includeBeta = true)!!.isBeta)
    }

    @Test
    fun pagingStopsAtTheFirstPageWithAStableBuild() = runBlocking {
        val serverReleases = (1..50).map { release("v1.0.0-rc.$it") }
        servePages(
            serverReleases,
            listOf(release("android-v0.2.12-beta", prerelease = true)) + List(49) { release("v0.9.$it") },
            listOf(release("android-v0.2.11"), release("android-v0.2.10-beta", prerelease = true)),
            listOf(release("android-v0.2.9")),
        )

        assertEquals(11, updater.findUpdate(1, includeBeta = false)!!.versionCode)
        assertEquals(listOf("1", "2", "3"), server.requests.map { it.query("page") })

        server.requests.clear()
        assertEquals(12, updater.findUpdate(1, includeBeta = true)!!.versionCode)
        assertEquals(listOf("1", "2", "3"), server.requests.map { it.query("page") })
    }

    @Test
    fun usedUpRateLimitIsReported() {
        server.handler = {
            jsonResponse("""{"message":"API rate limit exceeded"}""", code = 403)
                .setHeader("x-ratelimit-remaining", "0")
                .setHeader("x-ratelimit-reset", "1791889200")
        }
        val error = assertThrows(UpdateError.RateLimited::class.java) { runBlocking { updater.findUpdate(1, includeBeta = false) } }
        assertEquals(1_791_889_200L, error.resetAtEpochSeconds)

        server.handler = { jsonResponse("""{"message":"Not Found"}""", code = 404) }
        assertEquals(404, assertThrows(UpdateError.Http::class.java) { runBlocking { updater.findUpdate(1, includeBeta = false) } }.status)

        server.handler = { jsonResponse("""<html>proxy login</html>""") }
        assertThrows(UpdateError.InvalidResponse::class.java) { runBlocking { updater.findUpdate(1, includeBeta = false) } }
    }

    @Test
    fun downloadKeepsTheApkOnlyWhenSizeAndDigestMatch() = runBlocking {
        val apk = ByteArray(300_000) { (it % 251).toByte() }
        val sha256 = MessageDigest.getInstance("SHA-256").digest(apk).joinToString("") { "%02x".format(it) }
        server.handler = { request ->
            if (request.path.startsWith("/repos/")) {
                jsonResponse("[" + release("android-v0.2.20", assets = listOf(asset(size = apk.size.toLong(), digest = "sha256:$sha256"))) + "]")
            } else {
                MockResponse().setBody(Buffer().write(apk))
            }
        }
        val update = updater.findUpdate(1, includeBeta = false)!!
        assertEquals(sha256, update.apkSha256)
        val target = File(workDir, "updates/20.apk")
        val progress = mutableListOf<Pair<Long, Long>>()

        updater.download(update, target) { done, total -> progress += done to total }

        assertArrayEquals(apk, target.readBytes())
        assertEquals(apk.size.toLong() to apk.size.toLong(), progress.last())
        assertTrue(progress.all { it.second == apk.size.toLong() })
        assertEquals(listOf("20.apk"), target.parentFile!!.list()!!.toList())

        target.delete()
        val tampered = update.copy(apkSha256 = "0".repeat(64))
        assertThrows(UpdateError.CorruptDownload::class.java) { runBlocking { updater.download(tampered, target) { _, _ -> } } }
        assertFalse(target.exists())
        assertEquals(emptyList<String>(), target.parentFile!!.list()!!.toList())

        val truncated = update.copy(apkSize = apk.size + 1L, apkSha256 = null)
        assertThrows(UpdateError.CorruptDownload::class.java) { runBlocking { updater.download(truncated, target) { _, _ -> } } }
        assertFalse(target.exists())
    }

    @Test
    fun failedDownloadReportsTheStatusAndLeavesNothing() = runBlocking {
        server.handler = { MockResponse().setResponseCode(404) }
        val release = AppRelease("0.2.20", 20, false, "", "", "${server.baseUrl}/download/x.apk", 10, null)
        val target = File(workDir, "20.apk")

        val error = assertThrows(UpdateError.Http::class.java) { runBlocking { updater.download(release, target) { _, _ -> } } }

        assertEquals(404, error.status)
        assertEquals(emptyList<String>(), workDir.list()!!.toList())
    }
}
