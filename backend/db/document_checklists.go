package db

import (
	"context"
	"database/sql"
	"my-ple-backend/model"
)

func (db *Database) DocumentChecklist(ctx context.Context, jobID, checklistID int) (*model.DocumentChecklist, error) {
	tx, err := db.Conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	c := &model.DocumentChecklist{JobID: jobID, Signers: []model.ChecklistSigner{}, Items: []model.ChecklistItem{}}
	err = tx.QueryRowContext(ctx, `SELECT id,title,require_digital_signature,version FROM document_checklists WHERE job_id=$1 AND id=$2`, jobID, checklistID).Scan(&c.ID, &c.Title, &c.RequireDigitalSignature, &c.Version)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,signer_type,name,position,display_order,archived_at IS NOT NULL FROM document_checklist_signers WHERE checklist_id=$1 ORDER BY display_order,id`, c.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var s model.ChecklistSigner
		if err = rows.Scan(&s.ID, &s.Type, &s.Name, &s.Position, &s.DisplayOrder, &s.Archived); err != nil {
			rows.Close()
			return nil, err
		}
		c.Signers = append(c.Signers, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,document_name,digital_signed,document_position,to_char(sent_date,'YYYY-MM-DD'),notes,created_at,updated_at FROM document_checklist_items WHERE checklist_id=$1 AND deleted_at IS NULL ORDER BY id`, c.ID)
	if err != nil {
		return nil, err
	}
	indexes := map[int]int{}
	for rows.Next() {
		item := model.ChecklistItem{Signatures: map[int]bool{}}
		if err = rows.Scan(&item.ID, &item.DocumentName, &item.DigitalSigned, &item.DocumentPosition, &item.SentDate, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		indexes[item.ID] = len(c.Items)
		c.Items = append(c.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT item_id,signer_id,signed FROM document_checklist_signatures WHERE checklist_id=$1`, c.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item, signer int
		var signed bool
		if err = rows.Scan(&item, &signer, &signed); err != nil {
			rows.Close()
			return nil, err
		}
		if i, ok := indexes[item]; ok {
			c.Items[i].Signatures[signer] = signed
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return c, tx.Commit()
}

func (db *Database) SaveDocumentChecklist(ctx context.Context, jobID, checklistID int, in model.ChecklistConfigInput) (int, error) {
	if err := in.Validate(); err != nil {
		return 0, err
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id, version int
	if checklistID == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO document_checklists(job_id,title,require_digital_signature) VALUES($1,$2,$3) RETURNING id`, jobID, in.Title, in.RequireDigitalSignature).Scan(&id)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT id,version FROM document_checklists WHERE job_id=$1 AND id=$2 FOR UPDATE`, jobID, checklistID).Scan(&id, &version)
		if err != nil {
			return 0, err
		}
		if version != in.Version {
			return 0, model.ErrChecklistConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE document_checklists SET title=$2,require_digital_signature=$3,version=version+1,updated_at=NOW() WHERE id=$1`, id, in.Title, in.RequireDigitalSignature)
	}
	if err != nil {
		return 0, err
	}
	// Archive removed/changed identities; never reassign their historical checkboxes.
	rows, err := tx.QueryContext(ctx, `SELECT id,signer_type,name,position FROM document_checklist_signers WHERE checklist_id=$1 AND archived_at IS NULL`, id)
	if err != nil {
		return 0, err
	}
	old := map[int]model.ChecklistSigner{}
	for rows.Next() {
		var s model.ChecklistSigner
		if err = rows.Scan(&s.ID, &s.Type, &s.Name, &s.Position); err != nil {
			rows.Close()
			return 0, err
		}
		old[s.ID] = s
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, s := range in.Signers {
		if s.ID > 0 {
			previous, ok := old[s.ID]
			if !ok {
				return 0, model.ErrInvalidChecklist
			}
			if previous.Type == s.Type && previous.Name == s.Name && previous.Position == s.Position {
				if _, err = tx.ExecContext(ctx, `UPDATE document_checklist_signers SET display_order=$2 WHERE id=$1`, s.ID, s.DisplayOrder); err != nil {
					return 0, err
				}
				delete(old, s.ID)
				continue
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO document_checklist_signers(checklist_id,signer_type,name,position,display_order) VALUES($1,$2,$3,$4,$5)`, id, s.Type, s.Name, s.Position, s.DisplayOrder); err != nil {
			return 0, err
		}
	}
	for signerID := range old {
		if _, err = tx.ExecContext(ctx, `UPDATE document_checklist_signers SET archived_at=NOW() WHERE id=$1`, signerID); err != nil {
			return 0, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE jobs SET used_at=COALESCE(used_at,NOW()) WHERE id=$1`, jobID); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (db *Database) MutateChecklistItem(ctx context.Context, jobID, checklistID, itemID int, method string, patch model.ChecklistItemPatch) error {
	if method != "DELETE" {
		if err := patch.Validate(); err != nil {
			return err
		}
	}
	tx, err := db.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var configID int
	var digital bool
	if err = tx.QueryRowContext(ctx, `SELECT id,require_digital_signature FROM document_checklists WHERE job_id=$1 AND id=$2 FOR UPDATE`, jobID, checklistID).Scan(&configID, &digital); err != nil {
		return err
	}
	if method == "POST" {
		if patch.Field != "documentName" {
			return model.ErrInvalidChecklist
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO document_checklist_items(checklist_id,document_name) VALUES($1,$2)`, configID, patch.Value)
	} else {
		var found int
		if err = tx.QueryRowContext(ctx, `SELECT id FROM document_checklist_items WHERE id=$1 AND checklist_id=$2 AND deleted_at IS NULL FOR UPDATE`, itemID, configID).Scan(&found); err != nil {
			return err
		}
		switch {
		case method == "DELETE":
			_, err = tx.ExecContext(ctx, `UPDATE document_checklist_items SET deleted_at=NOW(),updated_at=NOW() WHERE id=$1`, itemID)
		case patch.Field == "signature":
			if err = tx.QueryRowContext(ctx, `SELECT id FROM document_checklist_signers WHERE id=$1 AND checklist_id=$2 AND archived_at IS NULL`, patch.SignerID, configID).Scan(&found); err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO document_checklist_signatures(checklist_id,item_id,signer_id,signed) VALUES($1,$2,$3,$4) ON CONFLICT(item_id,signer_id) DO UPDATE SET signed=EXCLUDED.signed,updated_at=NOW()`, configID, itemID, patch.SignerID, patch.Checked)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE document_checklist_items SET updated_at=NOW() WHERE id=$1`, itemID)
			}
		case patch.Field == "digitalSigned":
			if !digital {
				return model.ErrInvalidChecklist
			}
			_, err = tx.ExecContext(ctx, `UPDATE document_checklist_items SET digital_signed=$2,updated_at=NOW() WHERE id=$1`, itemID, patch.Checked)
		default:
			// Column names come exclusively from this allowlist, never from user input.
			column := map[string]string{"documentName": "document_name", "documentPosition": "document_position", "sentDate": "sent_date", "notes": "notes"}[patch.Field]
			if column == "" {
				return model.ErrInvalidChecklist
			}
			var value any = patch.Value
			if patch.Field == "sentDate" && patch.Value == "" {
				value = nil
			}
			_, err = tx.ExecContext(ctx, `UPDATE document_checklist_items SET `+column+`=$2,updated_at=NOW() WHERE id=$1`, itemID, value)
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *Database) DocumentChecklists(ctx context.Context, jobID int) ([]*model.DocumentChecklist, error) {
	rows, err := db.Conn.QueryContext(ctx, `SELECT id FROM document_checklists WHERE job_id=$1 ORDER BY id`, jobID)
	if err != nil {
		return nil, err
	}
	ids := []int{}
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// ponytail: reuse per-table snapshots; batch reads if large table counts make refresh slow.
	checklists := []*model.DocumentChecklist{}
	for _, id := range ids {
		checklist, err := db.DocumentChecklist(ctx, jobID, id)
		if err != nil {
			return nil, err
		}
		checklists = append(checklists, checklist)
	}
	return checklists, nil
}
