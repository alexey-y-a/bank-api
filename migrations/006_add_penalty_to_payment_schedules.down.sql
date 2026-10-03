-- Откат миграции: удаляем колонку penalty.
ALTER TABLE payment_schedules
DROP COLUMN penalty;