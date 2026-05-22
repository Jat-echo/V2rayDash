-- Migration: Migrate existing subscription-server relationships to subscription_accounts
-- NOTE: subscription_accounts table is now created in schema.sql
-- This migration only handles data migration from old subscriptions.server_id

-- 迁移: 将现有订阅的 server_id 下的 enabled=true 账号迁移到关联表
INSERT INTO subscription_accounts (subscription_id, account_id)
SELECT s.id, a.id
FROM subscriptions s
INNER JOIN accounts a ON a.server_id = s.server_id AND a.enabled = true
WHERE s.server_id IS NOT NULL
ON CONFLICT (subscription_id, account_id) DO NOTHING;