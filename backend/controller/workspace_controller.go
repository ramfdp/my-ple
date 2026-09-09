package controller

import (
	"database/sql"
	"errors"
	"my-ple-backend/db"
	"my-ple-backend/middleware"
	"my-ple-backend/model"
	"my-ple-backend/notification"
	"net/http"
	"strconv"
	"strings"
)

type WorkspaceController struct {
	DB            *db.Database
	Notifications *notification.Service
}

func workspaceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "Gagal memproses workspace."
	switch {
	case errors.Is(err, db.ErrWorkspaceForbidden):
		status = 403
		message = err.Error()
	case errors.Is(err, db.ErrWorkspaceInvalid):
		status = 400
		message = err.Error()
	case errors.Is(err, db.ErrWorkspaceConflict):
		status = 409
		message = err.Error()
	case errors.Is(err, sql.ErrNoRows):
		status = 404
		message = "Data tidak ditemukan."
	}
	WriteJSON(w, status, map[string]string{"error": message})
}
func (c *WorkspaceController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetClaims(r.Context())
	if !ok {
		WriteJSON(w, 401, map[string]string{"error": "Autentikasi diperlukan."})
		return
	}
	path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if path[0] == "workspace-pages" && len(path) == 2 {
		c.workspacePage(w, r, user.ID, path[1])
		return
	}
	if path[0] == "work-items" && len(path) >= 2 {
		c.task(w, r, user.ID, path[1:])
		return
	}
	if path[0] != "workspaces" {
		http.NotFound(w, r)
		return
	}
	if len(path) == 1 && r.Method == "GET" {
		items, err := c.DB.Workspaces(r.Context(), user.ID)
		if err != nil {
			workspaceError(w, err)
			return
		}
		WriteJSON(w, 200, map[string]any{"workspaces": items})
		return
	}
	if len(path) < 2 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(path[1])
	if err != nil || id <= 0 {
		workspaceError(w, db.ErrWorkspaceInvalid)
		return
	}
	manager, err := c.DB.WorkspaceAccess(r.Context(), user.ID, id)
	if err != nil {
		workspaceError(w, err)
		return
	}
	if len(path) == 3 {
		switch path[2] {
		case "workload":
			if r.Method == "GET" {
				a, e := c.DB.WorkspaceWorkload(r.Context(), user.ID, id)
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"workload": a})
				return
			}
		case "overview":
			if r.Method == "GET" {
				a, e := c.DB.WorkspaceOverview(r.Context(), user.ID, id)
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"overview": a})
				return
			}
		case "pages":
			if r.Method == "GET" {
				a, e := c.DB.WorkspacePages(r.Context(), user.ID, id)
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"pages": a})
				return
			}
			if r.Method == "POST" {
				var in model.WorkspacePageInput
				if !DecodeStrictJSON(w, r, &in) {
					return
				}
				a, e := c.DB.SaveWorkspacePage(r.Context(), user.ID, id, "", in)
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 201, map[string]any{"page": a})
				return
			}
		case "tasks":
			if r.Method == "GET" {
				q := r.URL.Query()
				number := func(k string) int { v, _ := strconv.Atoi(q.Get(k)); return v }
				items, total, e := c.DB.WorkItems(r.Context(), user.ID, id, db.WorkFilter{Search: q.Get("search"), Assignee: q.Get("assignee"), Status: q.Get("status"), Priority: q.Get("priority"), Start: q.Get("start"), End: q.Get("end"), Sort: q.Get("sort"), JobID: number("jobId"), Limit: number("limit"), Offset: number("offset")})
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"tasks": items, "total": total})
				return
			}
			if r.Method == "POST" {
				input := model.WorkItemInput{Status: model.WorkTodo, Priority: model.PriorityMedium}
				if !DecodeStrictJSON(w, r, &input) {
					return
				}
				t, e := c.DB.SaveWorkItem(r.Context(), user.ID, id, "", input)
				if e != nil {
					workspaceError(w, e)
					return
				}
				c.changed(r, user.ID, id)
				WriteJSON(w, 201, map[string]any{"task": t})
				return
			}
		case "activity":
			if r.Method == "GET" {
				a, e := c.DB.WorkActivity(r.Context(), user.ID, id, "")
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"activity": a})
				return
			}
		case "members":
			if r.Method == "GET" {
				a, e := c.DB.WorkspaceMembers(r.Context(), user.ID, id)
				if e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"members": a})
				return
			}
		case "jobs":
			if r.Method == "GET" {
				rows, e := c.DB.Conn.QueryContext(r.Context(), `SELECT id,name,is_active FROM jobs WHERE division_id=$1 ORDER BY sort_order,id`, id)
				if e != nil {
					workspaceError(w, e)
					return
				}
				defer rows.Close()
				items := []map[string]any{}
				for rows.Next() {
					var jid int
					var name string
					var active bool
					if e = rows.Scan(&jid, &name, &active); e != nil {
						workspaceError(w, e)
						return
					}
					items = append(items, map[string]any{"id": jid, "name": name, "isActive": active})
				}
				if e = rows.Err(); e != nil {
					workspaceError(w, e)
					return
				}
				WriteJSON(w, 200, map[string]any{"jobs": items})
				return
			}
		}
	}
	if len(path) == 2 && r.Method == "GET" {
		members, err := c.DB.WorkspaceMembers(r.Context(), user.ID, id)
		if err != nil {
			workspaceError(w, err)
			return
		}
		var name string
		err = c.DB.Conn.QueryRowContext(r.Context(), `SELECT name FROM divisions WHERE id=$1`, id).Scan(&name)
		if err != nil {
			workspaceError(w, err)
			return
		}
		WriteJSON(w, 200, map[string]any{"workspace": map[string]any{"id": id, "name": name, "canManage": manager}, "members": members})
		return
	}
	http.NotFound(w, r)
}

func (c *WorkspaceController) workspacePage(w http.ResponseWriter, r *http.Request, userID, id string) {
	p, err := c.DB.WorkspacePage(r.Context(), userID, id)
	if err != nil {
		workspaceError(w, err)
		return
	}
	switch r.Method {
	case "GET":
		WriteJSON(w, 200, map[string]any{"page": p})
		return
	case "PUT", "PATCH":
		in := model.WorkspacePageInput{ParentID: p.ParentID, Title: p.Title, Content: p.Content, Visibility: p.Visibility}
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		updated, e := c.DB.SaveWorkspacePage(r.Context(), userID, p.WorkspaceID, id, in)
		if e != nil {
			workspaceError(w, e)
			return
		}
		WriteJSON(w, 200, map[string]any{"page": updated})
		return
	case "DELETE":
		var in struct {
			Version int `json:"version"`
		}
		if !DecodeStrictJSON(w, r, &in) {
			return
		}
		if e := c.DB.DeleteWorkspacePage(r.Context(), userID, id, in.Version); e != nil {
			workspaceError(w, e)
			return
		}
		w.WriteHeader(204)
		return
	}
	http.NotFound(w, r)
}

func (c *WorkspaceController) changed(r *http.Request, actor string, division int) {
	if c.Notifications == nil || c.Notifications.Hub == nil {
		return
	}
	ids, err := c.DB.WorkspaceRecipients(r.Context(), division)
	if err == nil {
		c.Notifications.Hub.SendWorkspace(ids, division, actor)
	}
}
func (c *WorkspaceController) task(w http.ResponseWriter, r *http.Request, userID string, path []string) {
	t, err := c.DB.WorkItem(r.Context(), userID, path[0])
	if err != nil {
		workspaceError(w, err)
		return
	}
	if len(path) == 1 {
		switch r.Method {
		case "GET":
			WriteJSON(w, 200, map[string]any{"task": t})
			return
		case "PATCH", "PUT":
			in := model.WorkItemInput{Title: t.Title, Description: t.Description, RelatedURL: t.RelatedURL, JobID: t.JobID, StartDate: t.StartDate, DueDate: t.DueDate, Status: t.Status, Priority: t.Priority}
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			updated, e := c.DB.SaveWorkItem(r.Context(), userID, t.WorkspaceID, t.ID, in)
			if e != nil {
				workspaceError(w, e)
				return
			}
			c.changed(r, userID, t.WorkspaceID)
			WriteJSON(w, 200, map[string]any{"task": updated})
			return
		case "DELETE":
			var in struct {
				Version int `json:"version"`
			}
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			if e := c.DB.ArchiveWorkItem(r.Context(), userID, t.ID, in.Version); e != nil {
				workspaceError(w, e)
				return
			}
			c.changed(r, userID, t.WorkspaceID)
			w.WriteHeader(204)
			return
		}
	}
	if len(path) == 2 && path[1] == "comments" {
		if r.Method == "GET" {
			a, e := c.DB.WorkComments(r.Context(), userID, t.ID)
			if e != nil {
				workspaceError(w, e)
				return
			}
			WriteJSON(w, 200, map[string]any{"comments": a})
			return
		}
		if r.Method == "POST" {
			var in struct {
				Content string `json:"content"`
			}
			if !DecodeStrictJSON(w, r, &in) {
				return
			}
			if e := c.DB.AddWorkComment(r.Context(), userID, t.ID, in.Content); e != nil {
				workspaceError(w, e)
				return
			}
			c.changed(r, userID, t.WorkspaceID)
			w.WriteHeader(204)
			return
		}
	}
	if len(path) == 2 && path[1] == "activity" && r.Method == "GET" {
		a, e := c.DB.WorkActivity(r.Context(), userID, t.WorkspaceID, t.ID)
		if e != nil {
			workspaceError(w, e)
			return
		}
		WriteJSON(w, 200, map[string]any{"activity": a})
		return
	}
	http.NotFound(w, r)
}
