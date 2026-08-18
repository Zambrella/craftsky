import 'dart:async';

import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:path/path.dart' as path;
import 'package:sqflite/sqflite.dart';
import 'package:uuid/uuid.dart';

enum NotificationDeliveryStage {
  presented,
  foregroundEffectEmitted,
  opened,
}

abstract interface class NotificationDeliveryDedupeStore {
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  });

  Future<void> clearAccountPartition(String accountPartition);
}

/// A bounded SQLite compare-and-set store shared by foreground/background
/// plugin isolates. The unique key is the validated notification UUID plus a
/// stage; the account partition exists only for targeted cache clearing.
final class SqliteNotificationDeliveryDedupeStore
    implements NotificationDeliveryDedupeStore {
  SqliteNotificationDeliveryDedupeStore({
    DatabaseFactory? databaseFactory,
    String? databasePath,
    int capacity = defaultCapacity,
    Duration ttl = defaultTtl,
    DateTime Function()? now,
  }) : _databaseFactory = databaseFactory ?? databaseFactoryDefault,
       // Keep the public named parameter descriptive; an initializing formal
       // would expose the private field name as the constructor API.
       // ignore: prefer_initializing_formals
       _databasePath = databasePath,
       _capacity = _validateCapacity(capacity),
       _ttl = _validateTtl(ttl),
       _now = now ?? DateTime.now;

  static const int minCapacity = 32;
  static const int defaultCapacity = 512;
  static const int maxCapacity = 4096;
  static const Duration minTtl = Duration(hours: 1);
  static const Duration defaultTtl = Duration(days: 7);
  static const Duration maxTtl = Duration(days: 30);
  static const String _databaseFilename =
      'craftsky_notification_delivery_dedupe.sqlite';
  static const String _table = 'notification_delivery_stages';

  static DatabaseFactory get databaseFactoryDefault => databaseFactory;

  final DatabaseFactory _databaseFactory;
  final String? _databasePath;
  final int _capacity;
  final Duration _ttl;
  final DateTime Function() _now;
  Future<Database>? _database;

  @override
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  }) async {
    if (!Uuid.isValidUUID(fromString: notificationId)) {
      throw const FormatException('Invalid notification delivery ID');
    }
    AccountSubscriptionId.parse(accountPartition);

    final database = await _openDatabase();
    final timestamp = _now().toUtc().microsecondsSinceEpoch;
    final expiresBefore = timestamp - _ttl.inMicroseconds;
    final normalizedId = notificationId.toLowerCase();
    final stageName = stage.name;
    final claimNonce = const Uuid().v4();

    return database.transaction((transaction) async {
      await transaction.delete(
        _table,
        where: 'last_seen_at <= ?',
        whereArgs: [expiresBefore],
      );
      await transaction.insert(
        _table,
        {
          'notification_id': normalizedId,
          'stage': stageName,
          'account_partition': accountPartition,
          'claim_nonce': claimNonce,
          'created_at': timestamp,
          'last_seen_at': timestamp,
        },
        conflictAlgorithm: ConflictAlgorithm.ignore,
      );
      final rows = await transaction.query(
        _table,
        columns: ['claim_nonce'],
        where: 'notification_id = ? AND stage = ?',
        whereArgs: [normalizedId, stageName],
        limit: 1,
      );
      final won = rows.length == 1 && rows.single['claim_nonce'] == claimNonce;
      if (!won) {
        await transaction.update(
          _table,
          {'last_seen_at': timestamp},
          where: 'notification_id = ? AND stage = ?',
          whereArgs: [normalizedId, stageName],
        );
      }
      await transaction.rawDelete(
        '''
        DELETE FROM $_table
        WHERE rowid IN (
          SELECT rowid FROM $_table
          ORDER BY last_seen_at DESC, notification_id DESC, stage DESC
          LIMIT -1 OFFSET ?
        )
      ''',
        [_capacity],
      );
      return won;
    });
  }

  @override
  Future<void> clearAccountPartition(String accountPartition) async {
    AccountSubscriptionId.parse(accountPartition);
    final database = await _openDatabase();
    await database.delete(
      _table,
      where: 'account_partition = ?',
      whereArgs: [accountPartition],
    );
  }

  Future<Database> _openDatabase() => _database ??= _open();

  Future<Database> _open() async {
    final resolvedPath =
        _databasePath ??
        path.join(await _databaseFactory.getDatabasesPath(), _databaseFilename);
    try {
      return await _openAtPath(resolvedPath);
    } on DatabaseException catch (error) {
      if (!_isCorruptDatabase(error)) rethrow;
      // This database is a disposable suppression cache, never authority. A
      // corrupt file cannot be queried consistently by any isolate, so reset
      // it once and rebuild the closed schema instead of permanently blocking
      // notification delivery.
      await _databaseFactory.deleteDatabase(resolvedPath);
      return _openAtPath(resolvedPath);
    }
  }

  Future<Database> _openAtPath(String resolvedPath) {
    return _databaseFactory.openDatabase(
      resolvedPath,
      options: OpenDatabaseOptions(
        version: 1,
        // Opening the same path should reuse one connection per isolate.
        // ignore: avoid_redundant_argument_values
        singleInstance: true,
        // Explicitly retain in-flight transactions when another isolate opens
        // the shared database; this is part of the cross-isolate CAS contract.
        rollbackActiveTransactionOnOpen: false,
        onConfigure: (database) async {
          await database.execute('PRAGMA busy_timeout = 5000');
          await database.execute('PRAGMA journal_mode = WAL');
        },
        onCreate: (database, _) => _createSchema(database),
        onOpen: (database) async {
          await _createSchema(database);
          // Cache rows that fail the current closed-stage/shape contract are
          // disposable and must never poison future delivery handling.
          await database.rawDelete('''
            DELETE FROM $_table
            WHERE length(notification_id) <> 36
               OR stage NOT IN ('presented','foregroundEffectEmitted','opened')
               OR length(account_partition) NOT BETWEEN 1 AND 128
          ''');
        },
      ),
    );
  }

  static bool _isCorruptDatabase(DatabaseException error) {
    final resultCode = error.getResultCode();
    if (resultCode != null) {
      final primaryCode = resultCode & 0xff;
      if (primaryCode == 11 || primaryCode == 26) return true;
    }
    final message = error.toString().toLowerCase();
    return message.contains('database disk image is malformed') ||
        message.contains('file is not a database');
  }

  static Future<void> _createSchema(Database database) async {
    await database.execute('''
      CREATE TABLE IF NOT EXISTS $_table (
        notification_id TEXT NOT NULL CHECK(length(notification_id) = 36),
        stage TEXT NOT NULL CHECK(
          stage IN ('presented','foregroundEffectEmitted','opened')
        ),
        account_partition TEXT NOT NULL CHECK(
          length(account_partition) BETWEEN 1 AND 128
        ),
        claim_nonce TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        last_seen_at INTEGER NOT NULL,
        PRIMARY KEY(notification_id, stage)
      )
    ''');
    await database.execute('''
      CREATE INDEX IF NOT EXISTS notification_delivery_partition_idx
      ON $_table(account_partition, notification_id, stage)
    ''');
  }

  static int _validateCapacity(int capacity) {
    if (capacity < minCapacity || capacity > maxCapacity) {
      throw RangeError.range(
        capacity,
        minCapacity,
        maxCapacity,
        'capacity',
      );
    }
    return capacity;
  }

  static Duration _validateTtl(Duration ttl) {
    if (ttl < minTtl || ttl > maxTtl) {
      throw RangeError.range(
        ttl.inMicroseconds,
        minTtl.inMicroseconds,
        maxTtl.inMicroseconds,
        'ttl',
      );
    }
    return ttl;
  }
}
