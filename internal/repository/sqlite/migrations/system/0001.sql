CREATE TABLE system__config (
	id              TEXT NOT NULL PRIMARY KEY,
	system_email    TEXT NOT NULL,
	twilio_sid      TEXT NOT NULL,
	twilio_token    TEXT NOT NULL,
	twilio_from_tel TEXT NOT NULL,
	created_at      DATETIME NOT NULL,
	updated_at      DATETIME
);
