-- Generated with github.com/jinzhu/gorm v1.9.16 and the pre-migration models.
CREATE TABLE "gee_tiny" ("id" integer primary key autoincrement,"created_at" datetime NOT NULL,"updated_at" datetime NOT NULL,"ori_link" text NOT NULL,"ori_md5" varchar(32) NOT NULL,"tiny_key" varchar(32) NOT NULL,"one_time" bool NOT NULL DEFAULT false,"visit_count" int NOT NULL DEFAULT 0);
CREATE TABLE "kv_counters" ("id" integer primary key autoincrement,"created_at" datetime NOT NULL,"updated_at" datetime NOT NULL,"key" varchar(128) NOT NULL,"value" bigint NOT NULL DEFAULT 0,"visibility" varchar(16) NOT NULL DEFAULT 'private');
CREATE TABLE "kv_entries" ("id" integer primary key autoincrement,"created_at" datetime NOT NULL,"updated_at" datetime NOT NULL,"key" varchar(128) NOT NULL,"value" text NOT NULL,"content_type" varchar(64) NOT NULL DEFAULT 'text/plain',"visibility" varchar(16) NOT NULL DEFAULT 'private');
CREATE INDEX idx_gee_tiny_created_at ON "gee_tiny"(created_at);
CREATE INDEX idx_gee_tiny_updated_at ON "gee_tiny"(updated_at);
CREATE INDEX idx_kv_counters_created_at ON "kv_counters"(created_at);
CREATE INDEX idx_kv_counters_updated_at ON "kv_counters"(updated_at);
CREATE INDEX idx_kv_entries_created_at ON "kv_entries"(created_at);
CREATE INDEX idx_kv_entries_updated_at ON "kv_entries"(updated_at);
CREATE INDEX idx_tiny_key ON "gee_tiny"(tiny_key);
CREATE UNIQUE INDEX uk_kv_counters_key ON "kv_counters"("key");
CREATE UNIQUE INDEX uk_kv_entries_key ON "kv_entries"("key");
CREATE UNIQUE INDEX uk_tiny_ori_md5 ON "gee_tiny"(ori_md5);
