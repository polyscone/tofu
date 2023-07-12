CREATE TABLE system__config (
	id                       TEXT NOT NULL PRIMARY KEY,
	system_email             TEXT NOT NULL,
	security_email           TEXT NOT NULL,
	sign_up_enabled          INTEGER NOT NULL,
	totp_required            INTEGER NOT NULL,
	totp_sms_enabled         INTEGER NOT NULL,
	google_sign_in_enabled   INTEGER NOT NULL,
	google_sign_in_client_id TEXT NOT NULL,
	twilio_sid               TEXT NOT NULL,
	twilio_token             TEXT NOT NULL,
	twilio_from_tel          TEXT NOT NULL,
	created_at               DATETIME NOT NULL,
	updated_at               DATETIME
);
