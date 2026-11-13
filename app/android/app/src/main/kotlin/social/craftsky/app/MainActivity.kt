package social.craftsky.app

import android.app.NotificationChannel
import android.app.NotificationManager
import android.media.AudioAttributes
import android.media.RingtoneManager
import android.os.Build
import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import com.android.billingclient.api.BillingClient
import com.android.billingclient.api.BillingClientStateListener
import com.android.billingclient.api.BillingResult
import com.android.billingclient.api.PendingPurchasesParams
import com.android.billingclient.api.Purchase
import com.android.billingclient.api.QueryPurchasesParams
import android.os.Handler
import android.os.Looper
import java.security.MessageDigest

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "social.craftsky/store_ownership")
            .setMethodCallHandler { call, result ->
                if (call.method != "check") {
                    result.notImplemented()
                    return@setMethodCallHandler
                }
                val appUserId = call.argument<String>("appUserId")
                if (appUserId.isNullOrBlank()) {
                    result.success(mapOf("status" to "unverified"))
                    return@setMethodCallHandler
                }
                checkStoreOwnership(appUserId, result)
            }
    }

    private fun checkStoreOwnership(appUserId: String, result: MethodChannel.Result) {
        // RevenueCat Android 10.26.0 sets the SHA-256 app user ID as Google's
        // obfuscatedAccountId. The store query is read-only and never posts a receipt.
        val expected = MessageDigest.getInstance("SHA-256")
            .digest(appUserId.toByteArray(Charsets.UTF_8))
            .joinToString("") { "%02x".format(it) }
        val main = Handler(Looper.getMainLooper())
        var completed = false
        lateinit var client: BillingClient
        val timeout = Runnable {
            if (!completed) {
                completed = true
                client.endConnection()
                result.success(mapOf("status" to "unverified"))
            }
        }
        fun finish(status: String, ownerHash: String? = null) {
            main.post {
                if (!completed) {
                    completed = true
                    main.removeCallbacks(timeout)
                    client.endConnection()
                    result.success(mapOf("status" to status, "ownerHash" to ownerHash))
                }
            }
        }
        client = BillingClient.newBuilder(applicationContext)
            .setListener { _, _ -> }
            .enablePendingPurchases(PendingPurchasesParams.newBuilder().enableOneTimeProducts().build())
            .build()
        main.postDelayed(timeout, 12_000)
        client.startConnection(object : BillingClientStateListener {
            override fun onBillingServiceDisconnected() = finish("unverified")
            override fun onBillingSetupFinished(billingResult: BillingResult) {
                if (billingResult.responseCode != BillingClient.BillingResponseCode.OK) {
                    finish("unverified")
                    return
                }
                client.queryPurchasesAsync(
                    QueryPurchasesParams.newBuilder().setProductType(BillingClient.ProductType.SUBS).build()
                ) { queryResult, purchases ->
                    if (queryResult.responseCode != BillingClient.BillingResponseCode.OK) {
                        finish("unverified")
                    } else if (purchases.any { it.purchaseState != Purchase.PurchaseState.PURCHASED }) {
                        // A pending payment must resolve before another plan can be bought.
                        finish("unverified")
                    } else if (purchases.any { it.accountIdentifiers?.obfuscatedAccountId.isNullOrBlank() }) {
                        finish("unverified")
                    } else if (purchases.any { it.accountIdentifiers?.obfuscatedAccountId != expected }) {
                        finish("conflict", purchases.first { it.accountIdentifiers?.obfuscatedAccountId != expected }.accountIdentifiers?.obfuscatedAccountId)
                    } else {
                        finish("allowed")
                    }
                }
            }
        })
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val sound = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION)
            val audioAttributes = AudioAttributes.Builder()
                .setUsage(AudioAttributes.USAGE_NOTIFICATION)
                .build()
            val channel = NotificationChannel(
                getString(R.string.default_notification_channel_id),
                getString(R.string.default_notification_channel_name),
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply {
                setSound(sound, audioAttributes)
                enableVibration(true)
            }
            getSystemService(NotificationManager::class.java)
                .createNotificationChannel(channel)
        }
    }
}
