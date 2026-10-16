package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_16_0 restructures the business_hours JSONB columns to support
// multiple working time ranges per day and holiday date ranges:
//   - `hours` values change from a single object
//     {"Monday": {"open": "09:00", "close": "17:00"}}
//     to an array of ranges
//     {"Monday": [{"open": "09:00", "close": "12:00"}, {"open": "14:00", "close": "18:00"}]}.
//   - `holidays` entries change from a single date
//     {"name": "New Year", "date": "2024-01-01"}
//     to an inclusive date range
//     {"name": "New Year", "start_date": "2024-01-01", "end_date": "2024-01-03"}.
func V2_16_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Wrap each single {"open", "close"} object in an array. Values that are
	// already arrays are left untouched.
	if _, err := db.Exec(`
		UPDATE business_hours bh
		SET hours = agg.new_hours
		FROM (
			SELECT bh2.id, jsonb_object_agg(
				e.k,
				CASE WHEN jsonb_typeof(e.v) = 'object'
					THEN jsonb_build_array(e.v)
					ELSE e.v END
			) AS new_hours
			FROM business_hours bh2
			CROSS JOIN LATERAL jsonb_each(bh2.hours) AS e(k, v)
			GROUP BY bh2.id
		) AS agg
		WHERE bh.id = agg.id;
	`); err != nil {
		return err
	}

	// Convert legacy {"name", "date"} holidays to {"name", "start_date", "end_date"}.
	if _, err := db.Exec(`
		UPDATE business_hours bh
		SET holidays = agg.new_holidays
		FROM (
			SELECT bh2.id, jsonb_agg(
				CASE
					WHEN h ? 'start_date' THEN h
					ELSE jsonb_build_object(
						'name', h->>'name',
						'start_date', h->>'date',
						'end_date', h->>'date')
				END
			) AS new_holidays
			FROM business_hours bh2
			CROSS JOIN LATERAL jsonb_array_elements(
				CASE WHEN jsonb_typeof(bh2.holidays) = 'array'
					THEN bh2.holidays
					ELSE '[]'::jsonb END
			) AS h
			GROUP BY bh2.id
		) AS agg
		WHERE bh.id = agg.id AND jsonb_typeof(bh.holidays) = 'array';
	`); err != nil {
		return err
	}

	// Normalize any non-array holidays column (e.g. the legacy '{}' default) to an empty array.
	if _, err := db.Exec(`
		UPDATE business_hours
		SET holidays = '[]'::jsonb
		WHERE jsonb_typeof(holidays) <> 'array';
	`); err != nil {
		return err
	}

	return nil
}
