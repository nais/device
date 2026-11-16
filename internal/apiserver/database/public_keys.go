package database

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/sirupsen/logrus"
)

func repairRawPublicKeys(db *sql.DB, log logrus.FieldLogger) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin public key repair: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	type repair struct {
		id  int64
		key string
	}
	counts := logrus.Fields{}
	for _, table := range []string{"devices", "gateways"} {
		rows, err := tx.Query(fmt.Sprintf("SELECT rowid, public_key FROM %s WHERE length(CAST(public_key AS BLOB)) = 32", table))
		if err != nil {
			return fmt.Errorf("read %s public keys: %w", table, err)
		}

		var repairs []repair
		for rows.Next() {
			var r repair
			if err := rows.Scan(&r.id, &r.key); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan %s public key: %w", table, err)
			}
			repairs = append(repairs, r)
		}
		readErr := rows.Err()
		closeErr := rows.Close()
		if readErr != nil {
			return fmt.Errorf("iterate %s public keys: %w", table, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s public keys: %w", table, closeErr)
		}

		for _, r := range repairs {
			canonical := base64.StdEncoding.EncodeToString([]byte(r.key))
			if _, err := tx.Exec(fmt.Sprintf("UPDATE %s SET public_key = ? WHERE rowid = ?", table), canonical, r.id); err != nil {
				return fmt.Errorf("repair %s public key for row %d: %w", table, r.id, err)
			}
		}
		counts[table] = len(repairs)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit public key repair: %w", err)
	}
	if counts["devices"] != 0 || counts["gateways"] != 0 {
		log.WithFields(counts).Info("repaired raw WireGuard public keys")
	}
	return nil
}
