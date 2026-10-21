package social.craftsky.app

import android.app.Activity
import android.content.Intent
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.util.concurrent.Executors

/** Returns a cache path, never export bytes or provider-selected names over the codec. */
class InstagramExportPicker(private val activity: Activity) {
    private var pending: MethodChannel.Result? = null
    private val executor = Executors.newSingleThreadExecutor()
    @Volatile private var disposed = false

    fun open(result: MethodChannel.Result) {
        if (pending != null) {
            result.error("picker_busy", "An export selection is already pending", null)
            return
        }
        pending = result
        try {
            val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                addCategory(Intent.CATEGORY_OPENABLE)
                type = "*/*"
                putExtra(Intent.EXTRA_MIME_TYPES, arrayOf("application/json", "application/zip", "application/x-zip-compressed"))
            }
            activity.startActivityForResult(intent, REQUEST_CODE)
        } catch (_: Exception) {
            pending = null
            result.error("read_error", "Export selection could not be opened", null)
        }
    }

    fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?): Boolean {
        if (requestCode != REQUEST_CODE) return false
        val result = pending ?: return true
        if (resultCode != Activity.RESULT_OK) {
            pending = null
            result.success(null)
            return true
        }
        val uri = data?.data
        if (uri == null) {
            pending = null
            result.error("read_error", "Selected export is unavailable", null)
            return true
        }
        executor.execute {
            var temporary: File? = null
            try {
                temporary = File.createTempFile("instagram-export-", ".tmp", activity.cacheDir)
                activity.contentResolver.openInputStream(uri).use { input ->
                    checkNotNull(input)
                    temporary.outputStream().use { output ->
                        val buffer = ByteArray(64 * 1024)
                        while (true) {
                            check(!disposed)
                            val count = input.read(buffer)
                            if (count < 0) break
                            output.write(buffer, 0, count)
                        }
                    }
                }
                val path = temporary.absolutePath
                activity.runOnUiThread {
                    pending = null
                    if (disposed) File(path).delete() else result.success(path)
                }
            } catch (_: Exception) {
                temporary?.delete()
                activity.runOnUiThread {
                    pending = null
                    if (!disposed) result.error("read_error", "Selected export could not be copied", null)
                }
            }
        }
        return true
    }

    fun dispose() {
        disposed = true
        pending = null
        executor.shutdownNow()
    }

    companion object {
        private const val REQUEST_CODE = 47219
    }
}
