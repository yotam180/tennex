CREATE TABLE `accounts` (
	`id` text PRIMARY KEY NOT NULL,
	`wa_jid` text,
	`display_name` text,
	`avatar_url` text,
	`status` text DEFAULT 'disconnected' NOT NULL,
	`last_seen` text,
	`created_at` text DEFAULT datetime('now') NOT NULL,
	`updated_at` text DEFAULT datetime('now') NOT NULL
);
--> statement-breakpoint
CREATE TABLE `conversations` (
	`id` text PRIMARY KEY NOT NULL,
	`account_id` text NOT NULL,
	`display_name` text,
	`last_message` text,
	`last_message_at` text,
	`unread_count` integer DEFAULT 0 NOT NULL,
	`is_pinned` integer DEFAULT false NOT NULL,
	`is_archived` integer DEFAULT false NOT NULL,
	`avatar_url` text,
	`updated_at` text DEFAULT datetime('now') NOT NULL
);
--> statement-breakpoint
CREATE TABLE `events` (
	`seq` integer PRIMARY KEY AUTOINCREMENT NOT NULL,
	`id` text NOT NULL,
	`timestamp` text DEFAULT datetime('now') NOT NULL,
	`type` text NOT NULL,
	`account_id` text NOT NULL,
	`device_id` text,
	`convo_id` text NOT NULL,
	`wa_message_id` text,
	`sender_jid` text,
	`payload` text NOT NULL,
	`attachment_ref` text,
	`applied` integer DEFAULT false NOT NULL,
	`synced_seq` integer
);
--> statement-breakpoint
CREATE TABLE `media_blobs` (
	`content_hash` text PRIMARY KEY NOT NULL,
	`mime_type` text NOT NULL,
	`size_bytes` integer NOT NULL,
	`storage_url` text,
	`local_path` text,
	`download_status` text DEFAULT 'pending' NOT NULL,
	`created_at` text DEFAULT datetime('now') NOT NULL
);
--> statement-breakpoint
CREATE TABLE `outbox` (
	`client_msg_uuid` text PRIMARY KEY NOT NULL,
	`account_id` text NOT NULL,
	`convo_id` text NOT NULL,
	`server_msg_id` integer,
	`status` text DEFAULT 'queued' NOT NULL,
	`last_error` text,
	`created_at` text DEFAULT datetime('now') NOT NULL,
	`updated_at` text DEFAULT datetime('now') NOT NULL,
	`retry_count` integer DEFAULT 0 NOT NULL,
	`next_retry_at` text
);
--> statement-breakpoint
CREATE TABLE `sync_state` (
	`account_id` text PRIMARY KEY NOT NULL,
	`last_sync_seq` integer DEFAULT 0 NOT NULL,
	`last_sync_at` text,
	`is_online` integer DEFAULT false NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `accounts_wa_jid_unique` ON `accounts` (`wa_jid`);--> statement-breakpoint
CREATE UNIQUE INDEX `events_id_unique` ON `events` (`id`);