package main

import (
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/lib/pq"
)

func main() {
	clean := flag.Bool("clean", false, "remove lingering bot_human_requested keys from conversations.meta")
	flag.Parse()

	dsn := "host=192.168.1.66 port=5432 user=libredesk password=libredesk dbname=libredesk sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if *clean {
		res, err := db.Exec(`UPDATE conversations
			SET meta = COALESCE(meta, '{}'::jsonb) - 'bot_human_requested',
			    updated_at = NOW()
			WHERE meta->>'bot_human_requested' IS NOT NULL`)
		if err != nil {
			fmt.Println("CLEAN ERR:", err)
			return
		}
		n, _ := res.RowsAffected()
		fmt.Printf("cleaned %d conversation(s)\n", n)
	}

	// Show the most recent chat conversations across all contacts, with contact
	// name, status, assignment and the lingering bot_human_requested marker.
	rows, err := db.Query(`
		SELECT c.id, c.uuid, COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') AS contact_name, u.id AS contact_id, cs.name AS status,
		       c.assigned_user_id, c.assigned_team_id,
		       c.created_at, c.meta->>'bot_human_requested' AS bot_human_requested
		FROM conversations c
		JOIN users u ON u.id = c.contact_id
		LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
		ORDER BY c.created_at DESC
		LIMIT 20`)
	if err != nil {
		fmt.Println("ERR:", err)
		return
	}
	defer rows.Close()

	fmt.Println("=== recent conversations ===")
	for rows.Next() {
		var (
			id, assignedUserID, assignedTeamID int
			contactID                           int
			uuid, contactName, status           string
			botHumanRequested                   sql.NullString
			createdAt                           interface{}
			assignedUser, assignedTeam          sql.NullInt64
		)
		if err := rows.Scan(&id, &uuid, &contactName, &contactID, &status, &assignedUser, &assignedTeam, &createdAt, &botHumanRequested); err != nil {
			fmt.Println("SCAN ERR:", err)
			return
		}
		if assignedUser.Valid {
			assignedUserID = int(assignedUser.Int64)
		}
		if assignedTeam.Valid {
			assignedTeamID = int(assignedTeam.Int64)
		}
		bhr := "-"
		if botHumanRequested.Valid {
			bhr = botHumanRequested.String
		}
		fmt.Printf("id=%d uuid=%s contact=%s(uid=%d) status=%s assigned_user=%d assigned_team=%d created=%v bot_human_requested=%s\n",
			id, uuid, contactName, contactID, status, assignedUserID, assignedTeamID, createdAt, bhr)
	}
	if err := rows.Err(); err != nil {
		fmt.Println("ROWS ERR:", err)
	}
}
