-- Add geolocation coordinates to users
ALTER TABLE auth_users ADD COLUMN last_login_latitude DOUBLE PRECISION;
ALTER TABLE auth_users ADD COLUMN last_login_longitude DOUBLE PRECISION;
