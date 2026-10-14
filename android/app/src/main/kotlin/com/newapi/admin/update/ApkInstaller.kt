package com.newapi.admin.update

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import androidx.core.content.FileProvider
import java.io.File

/**
 * Hands [apk] (a file in the update cache) to the system package installer. The installer asks the
 * user to allow installs from this app the first time. False when no app on the device can install it.
 */
fun Context.installApk(apk: File): Boolean {
    // The authority is declared by the FileProvider in AndroidManifest.xml.
    val uri = FileProvider.getUriForFile(this, "$packageName.updates", apk)
    val intent = Intent(Intent.ACTION_VIEW)
        .setDataAndType(uri, "application/vnd.android.package-archive")
        .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_ACTIVITY_NEW_TASK)
    return try {
        startActivity(intent)
        true
    } catch (e: ActivityNotFoundException) {
        false
    }
}
