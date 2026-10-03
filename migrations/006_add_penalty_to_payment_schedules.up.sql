-- Добавляем колонку penalty в таблицу payment_schedules.
ALTER TABLE payments_schedules
ADD COLUMN penalty BIGINT NOT NULL DEFAULT 0;