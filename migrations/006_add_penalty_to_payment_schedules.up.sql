-- Добавляем колонку penalty в таблицу payment_schedules.
ALTER TABLE payment_schedules
ADD COLUMN penalty BIGINT NOT NULL DEFAULT 0;