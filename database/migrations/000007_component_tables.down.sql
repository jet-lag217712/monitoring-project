DROP INDEX IF EXISTS idx_power_readings_device_time;
DROP INDEX IF EXISTS idx_power_components_device;
DROP INDEX IF EXISTS idx_temp_readings_device_time;
DROP INDEX IF EXISTS idx_temp_components_device;

DROP TABLE IF EXISTS device_power_readings;
DROP TABLE IF EXISTS device_power_components;
DROP TABLE IF EXISTS device_temperature_readings;
DROP TABLE IF EXISTS device_temperature_components;
