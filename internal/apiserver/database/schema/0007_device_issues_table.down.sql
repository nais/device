DROP TABLE kolide_checks;
DROP TABLE kolide_issues;

ALTER TABLE devices
  ADD COLUMN issues TEXT;

