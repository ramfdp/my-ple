package db

import "context"

func (db *Database) WorkspaceRecipients(ctx context.Context, division int) ([]string, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT user_id FROM workspace_access WHERE division_id=$1`, division)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
