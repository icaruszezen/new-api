package com.newapi.admin.ui.tickets

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import com.newapi.admin.core.model.TicketImage
import com.newapi.admin.core.model.TicketLimits
import java.io.ByteArrayInputStream
import java.io.ByteArrayOutputStream
import java.io.FileNotFoundException
import java.io.IOException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

// Android-only: the desktop UI check replaces this file with a stand-in of the same signatures.

/** Why a picked image cannot be attached. */
enum class UploadProblem { UNREADABLE, UNSUPPORTED, TOO_LARGE }

class TicketUploadException(val problem: UploadProblem) : Exception(problem.name)

/** Larger picked files are refused before decoding. */
private const val MAX_SOURCE_BYTES = 64 * 1024 * 1024

/** Pixels decoded at once (48 MB as ARGB); larger images are subsampled first. */
private const val MAX_DECODE_PIXELS = 12_000_000L

/** A converted upload is JPEG at this quality; the server re-encodes it once more. */
private const val JPEG_QUALITY = 90

/** Decodes [bytes] so neither side exceeds [maxSide], or returns null if Android cannot decode them. */
fun decodeTicketImage(bytes: ByteArray, maxSide: Int): ImageBitmap? = try {
    decodeScaled(bytes, maxSide, orientation = ExifInterface.ORIENTATION_NORMAL)?.asImageBitmap()
} catch (e: OutOfMemoryError) {
    null
}

/**
 * Reads a picked image and returns bytes the server accepts. A PNG, JPEG, GIF or WebP within the upload limits and
 * no larger than the server stores it is sent unchanged. Anything else (HEIC, a large camera photo, a long
 * screenshot, a rotated JPEG) is decoded upright, scaled to the stored size and sent as PNG when it came as PNG or GIF
 * or has transparency, else as JPEG, so the upload is smaller and the server's limits never reject it.
 */
suspend fun prepareTicketUpload(context: Context, uri: Uri): TicketImage = withContext(Dispatchers.IO) {
    val bytes = readPicked(context, uri)
    val mimeType = TicketImage.detectMimeType(bytes)
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    BitmapFactory.decodeByteArray(bytes, 0, bytes.size, bounds)
    if (bounds.outWidth <= 0 || bounds.outHeight <= 0) throw TicketUploadException(UploadProblem.UNSUPPORTED)
    val orientation = if (mimeType == "image/jpeg" || mimeType == null) exifOrientation(bytes) else ExifInterface.ORIENTATION_NORMAL
    val fitsAsIs = bytes.size <= TicketLimits.MAX_IMAGE_BYTES &&
        maxOf(bounds.outWidth, bounds.outHeight) <= TicketLimits.STORED_IMAGE_SIDE &&
        orientation == ExifInterface.ORIENTATION_NORMAL
    if (mimeType != null && fitsAsIs) return@withContext TicketImage(bytes, mimeType)

    val bitmap = try {
        decodeScaled(bytes, TicketLimits.STORED_IMAGE_SIDE, orientation)
    } catch (e: OutOfMemoryError) {
        throw TicketUploadException(UploadProblem.TOO_LARGE)
    } ?: throw TicketUploadException(UploadProblem.UNSUPPORTED)
    try {
        if (mimeType == "image/png" || mimeType == "image/gif" || bitmap.hasAlpha()) {
            val png = bitmap.encode(Bitmap.CompressFormat.PNG, 100)
            if (png.size <= TicketLimits.MAX_IMAGE_BYTES) return@withContext TicketImage(png, "image/png")
        }
        // JPEG has no transparency: paint transparent areas white rather than black.
        val opaque = if (bitmap.hasAlpha()) bitmap.onWhite() else bitmap
        val jpeg = opaque.encode(Bitmap.CompressFormat.JPEG, JPEG_QUALITY)
        if (jpeg.size > TicketLimits.MAX_IMAGE_BYTES) throw TicketUploadException(UploadProblem.TOO_LARGE)
        TicketImage(jpeg, "image/jpeg")
    } catch (e: OutOfMemoryError) {
        throw TicketUploadException(UploadProblem.TOO_LARGE)
    }
}

/**
 * The system photo picker for up to [maxItems] images (nothing when it is 0). [onPicked] gets at most [maxItems]:
 * on devices without the photo picker the document picker that replaces it ignores the limit.
 */
@Composable
fun rememberTicketImagePicker(maxItems: Int, onPicked: (List<Uri>) -> Unit): () -> Unit {
    val single = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) onPicked(listOf(uri))
    }
    // The multiple-item picker rejects a limit below 2, so one remaining slot uses the single picker.
    val multipleContract = remember(maxItems) { ActivityResultContracts.PickMultipleVisualMedia(maxItems.coerceAtLeast(2)) }
    val multiple = rememberLauncherForActivityResult(multipleContract) { uris ->
        if (uris.isNotEmpty()) onPicked(uris.take(maxItems))
    }
    return {
        val request = PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)
        when {
            maxItems == 1 -> single.launch(request)
            maxItems > 1 -> multiple.launch(request)
        }
    }
}

private fun readPicked(context: Context, uri: Uri): ByteArray {
    val input = try {
        context.contentResolver.openInputStream(uri)
    } catch (e: FileNotFoundException) {
        null
    } catch (e: SecurityException) {
        // The picker's read grant ended, for example after the app process was restarted.
        null
    } ?: throw TicketUploadException(UploadProblem.UNREADABLE)
    try {
        input.use { stream ->
            val out = ByteArrayOutputStream()
            val buffer = ByteArray(64 * 1024)
            while (true) {
                val read = stream.read(buffer)
                if (read < 0) break
                if (out.size() + read > MAX_SOURCE_BYTES) throw TicketUploadException(UploadProblem.TOO_LARGE)
                out.write(buffer, 0, read)
            }
            return out.toByteArray()
        }
    } catch (e: IOException) {
        throw TicketUploadException(UploadProblem.UNREADABLE)
    }
}

private fun exifOrientation(bytes: ByteArray): Int = try {
    ExifInterface(ByteArrayInputStream(bytes)).getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL)
} catch (e: Exception) {
    // Unreadable or malformed metadata; ExifInterface has thrown more than IOException for those.
    ExifInterface.ORIENTATION_NORMAL
}

/**
 * Decodes [bytes] subsampled to at least [maxSide] where memory allows, then turns it upright for the EXIF
 * [orientation] and scales it so neither side exceeds [maxSide].
 */
private fun decodeScaled(bytes: ByteArray, maxSide: Int, orientation: Int): Bitmap? {
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    BitmapFactory.decodeByteArray(bytes, 0, bytes.size, bounds)
    val width = bounds.outWidth
    val height = bounds.outHeight
    if (width <= 0 || height <= 0) return null
    var sample = 1
    while (maxOf(width, height) / (sample * 2) >= maxSide) sample *= 2
    while ((width / sample).toLong() * (height / sample) > MAX_DECODE_PIXELS) sample *= 2
    val decoded = BitmapFactory.decodeByteArray(bytes, 0, bytes.size, BitmapFactory.Options().apply { inSampleSize = sample })
        ?: return null
    val matrix = Matrix()
    when (orientation) {
        ExifInterface.ORIENTATION_FLIP_HORIZONTAL -> matrix.setScale(-1f, 1f)
        ExifInterface.ORIENTATION_ROTATE_180 -> matrix.setRotate(180f)
        ExifInterface.ORIENTATION_FLIP_VERTICAL -> matrix.setScale(1f, -1f)
        ExifInterface.ORIENTATION_TRANSPOSE -> {
            matrix.setRotate(90f)
            matrix.postScale(-1f, 1f)
        }
        ExifInterface.ORIENTATION_ROTATE_90 -> matrix.setRotate(90f)
        ExifInterface.ORIENTATION_TRANSVERSE -> {
            matrix.setRotate(-90f)
            matrix.postScale(-1f, 1f)
        }
        ExifInterface.ORIENTATION_ROTATE_270 -> matrix.setRotate(-90f)
    }
    val longSide = maxOf(decoded.width, decoded.height)
    if (longSide > maxSide) {
        val scale = maxSide.toFloat() / longSide
        matrix.postScale(scale, scale)
    }
    if (matrix.isIdentity) return decoded
    val result = Bitmap.createBitmap(decoded, 0, 0, decoded.width, decoded.height, matrix, true)
    if (result !== decoded) decoded.recycle()
    return result
}

private fun Bitmap.encode(format: Bitmap.CompressFormat, quality: Int): ByteArray {
    val out = ByteArrayOutputStream()
    if (!compress(format, quality, out)) throw TicketUploadException(UploadProblem.UNSUPPORTED)
    return out.toByteArray()
}

private fun Bitmap.onWhite(): Bitmap {
    val opaque = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
    Canvas(opaque).apply {
        drawColor(Color.WHITE)
        drawBitmap(this@onWhite, 0f, 0f, null)
    }
    return opaque
}
