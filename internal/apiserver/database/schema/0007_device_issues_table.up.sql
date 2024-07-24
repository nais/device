ALTER TABLE devices
  DROP COLUMN issues;

CREATE TABLE kolide_issues (
	id INTEGER PRIMARY KEY,
	device_id TEXT NOT NULL,
	check_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	detected_at TEXT NOT NULL,
	resolved_at TEXT,
	last_updated TEXT NOT NULL,
	ignored BOOLEAN NOT NULL DEFAULT 0,

	FOREIGN KEY (check_id) REFERENCES kolide_checks(id) ON DELETE CASCADE
);
CREATE INDEX kolide_issues_device_id ON kolide_issues (device_id);

CREATE TABLE kolide_checks (
	id INTEGER PRIMARY KEY,
	tags TEXT NOT NULL,
	display_name TEXT NOT NULL,
	description TEXT NOT NULL
);
