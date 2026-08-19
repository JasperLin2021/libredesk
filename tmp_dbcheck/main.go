package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=192.168.1.66 port=5432 user=libredesk password=libredesk dbname=libredesk sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	// Visitor 150 details
	fmt.Println("=== visitor 150 ===")
	var id int
	var typ, fn, email sql.NullString
	var hasVT bool
	var vt sql.NullString
	var enabled sql.NullBool
	var createdAt, updatedAt, deletedAt sql.NullTime
	err = db.QueryRow(`SELECT id, type, first_name, email, visitor_token, visitor_token<>'' AS has_vt, enabled, created_at, updated_at, deleted_at FROM users WHERE id=150`).
		Scan(&id, &typ, &fn, &email, &vt, &hasVT, &enabled, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		fmt.Println("query err:", err)
	} else {
		fmt.Printf("id=%d type=%s name=%s email=%v\n", id, typ.String, fn.String, email)
		fmt.Printf("has_vt=%v vt=%v enabled=%v\n", hasVT, vt.String, enabled.Bool)
		fmt.Printf("created=%v updated=%v deleted=%v\n", fmtTime(createdAt), fmtTime(updatedAt), deletedAt.Valid)
	}

	// Conversations of visitor 150
	fmt.Println("=== conversations of visitor 150 ===")
	cols2, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name='conversations' ORDER BY ordinal_position`)
	if err != nil {
		fmt.Println("cols err:", err)
		return
	}
	var allCols []string
	for cols2.Next() {
		var cn string
		if err := cols2.Scan(&cn); err != nil {
			continue
		}
		allCols = append(allCols, cn)
	}
	cols2.Close()
	fmt.Println("conv cols:", allCols)

	rows, err := db.Query(`SELECT id, uuid, contact_id, inbox_id, created_at, updated_at FROM conversations WHERE contact_id=150`)
	if err != nil {
		fmt.Println("conv query err:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var uuid string
		var contactID, inboxID int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&cid, &uuid, &contactID, &inboxID, &createdAt, &updatedAt); err != nil {
			fmt.Println("scan err:", err)
			continue
		}
		fmt.Printf("conv id=%d uuid=%s contact=%d inbox=%d created=%s updated=%s\n",
			cid, uuid, contactID, inboxID, createdAt.Format("2006-01-02 15:04:05"), updatedAt.Format("2006-01-02 15:04:05"))
	}

	// conversations table columns
	fmt.Println("=== conversations columns ===")
	rows3, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name='conversations' ORDER BY ordinal_position`)
	if err != nil {
		fmt.Println("cols err:", err)
		return
	}
	var cols []string
	for rows3.Next() {
		var cn string
		if err := rows3.Scan(&cn); err != nil {
			continue
		}
		cols = append(cols, cn)
	}
	rows3.Close()
	fmt.Println(cols)

	// inboxes
	fmt.Println("=== inboxes ===")
	rows6, err := db.Query(`SELECT id, uuid, name FROM inboxes ORDER BY id`)
	if err != nil {
		fmt.Println("inbox err:", err)
	} else {
		for rows6.Next() {
			var iid int
			var iuuid, iname sql.NullString
			if err := rows6.Scan(&iid, &iuuid, &iname); err != nil {
				continue
			}
			fmt.Printf("inbox id=%d uuid=%s name=%s\n", iid, iuuid.String, iname.String)
		}
		rows6.Close()
	}

	// Contact id 150's bixiao contact if any
	fmt.Println("=== users with external_user_id non-null (contacts) ===")
	rows4, err := db.Query(`SELECT id, type, first_name, external_user_id FROM users WHERE external_user_id IS NOT NULL AND external_user_id <> '' ORDER BY id DESC LIMIT 10`)
	if err != nil {
		fmt.Println("contact query err:", err)
	} else {
		for rows4.Next() {
			var cid int
			var ctyp, cfn, ceid sql.NullString
			if err := rows4.Scan(&cid, &ctyp, &cfn, &ceid); err != nil {
				continue
			}
			fmt.Printf("id=%d type=%s name=%s ext_id=%s\n", cid, ctyp.String, cfn.String, ceid.String)
		}
		rows4.Close()
	}

	// All sessions in redis? Can't directly. Check conversations of the bixiao contact
	fmt.Println("=== conversations of contact users (type='contact') ===")
	rows5, err := db.Query(`SELECT u.id, u.first_name, count(c.id) FROM users u
		LEFT JOIN conversations c ON c.contact_id = u.id
		WHERE u.type='contact' GROUP BY u.id HAVING count(c.id) > 0 ORDER BY u.id DESC LIMIT 20`)
	if err != nil {
		fmt.Println("contact conv err:", err)
	} else {
		for rows5.Next() {
			var cid, cnt int
			var cfn string
			if err := rows5.Scan(&cid, &cfn, &cnt); err != nil {
				continue
			}
			fmt.Printf("contact_id=%d name=%s conv_count=%d\n", cid, cfn, cnt)
		}
		rows5.Close()
	}
}

func fmtTime(t sql.NullTime) string {
	if t.Valid {
		return t.Time.Format("2006-01-02 15:04:05")
	}
	return "NULL"
}
