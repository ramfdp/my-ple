package db

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"my-ple-backend/model"
)

type HermesSource struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	URL     string `json:"url"`
}

var searchWords = regexp.MustCompile(`[\pL\pN]{3,}`)

func hermesQuery(question string) string {
	words := searchWords.FindAllString(strings.ToLower(question), -1)
	stop := map[string]bool{"yang": true, "dan": true, "dari": true, "untuk": true, "dengan": true, "bagaimana": true, "cara": true, "what": true, "how": true, "the": true, "can": true, "this": true, "problem": true, "masalah": true}
	seen := map[string]bool{}
	terms := make([]string, 0, 12)
	for _, word := range words {
		if stop[word] || seen[word] {
			continue
		}
		seen[word] = true
		terms = append(terms, word)
		if len(terms) == 12 {
			break
		}
	}
	return strings.Join(terms, " ")
}

// Search only fields exposed by the corresponding MyHIS views.
func (db *Database) HermesSearch(ctx context.Context, claims model.Claims, question string) ([]HermesSource, error) {
	terms := hermesQuery(question)
	if terms == "" {
		return []HermesSource{}, nil
	}
	division := map[string]string{"Project Engineer": "Project Engineer", "IT": "AI Dev", "Finance": "Keuangan", "Business Development": "Business Development", "HR": "HR", "PMO": "PMO"}[claims.RoleName]
	allReports := claims.RoleName == "Admin" || claims.RoleName == "Management"
	now := time.Now()
	if db.Now != nil {
		now = db.Now()
	}
	weekStart, _ := model.ReportingPeriodWeek(now)
	query := `WITH q AS (SELECT to_tsquery('simple', replace($1, ' ', ' | ')) AS terms), candidates AS (
	 SELECT 'lesson:'||l.id AS id, 'lesson' AS kind, l.title AS title,
	 concat_ws(E'\n', l.source_project, l.title, l.internal_factor, l.external_factor, l.impact, l.action, l.notes) AS excerpt,
	 '/lessons-learned#lesson-'||l.id AS url, ts_rank_cd(to_tsvector('simple', concat_ws(' ', l.source_project,l.title,l.internal_factor,l.external_factor,l.impact,l.action,l.notes)), q.terms) AS score
	 FROM lessons_learned l CROSS JOIN q WHERE l.deleted_at IS NULL AND $3
	 AND to_tsvector('simple', concat_ws(' ', l.source_project,l.title,l.internal_factor,l.external_factor,l.impact,l.action,l.notes)) @@ q.terms
	 UNION ALL
	 SELECT 'task:'||t.id, 'task', t.title, concat_ws(E'\n', d.name, t.title, t.description, t.status),
	 '/workspace/'||t.division_id||'?task='||t.id||CASE WHEN t.archived_at IS NULL THEN '' ELSE '&archived=1' END,
	 ts_rank_cd(to_tsvector('simple', t.title||' '||t.description), q.terms)
	 FROM work_items t JOIN divisions d ON d.id=t.division_id JOIN workspace_access a ON a.division_id=t.division_id AND a.user_id=$2 CROSS JOIN q
	 WHERE to_tsvector('simple', t.title||' '||t.description) @@ q.terms
	 UNION ALL
	 SELECT 'report:'||r.id, 'report', r.work, concat_ws(E'\n', r.work_division,r.work,r.description),
	 '/weekly-progress?division='||replace(r.primary_division,' ','%20')||'#report-'||r.id,
	 ts_rank_cd(to_tsvector('simple', r.work||' '||r.description), q.terms)
	 FROM reports r CROSS JOIN q WHERE ($4 OR r.primary_division=$5) AND ($4 OR $5<>'') AND r.report_week=$6::date
	 AND NOT EXISTS (SELECT 1 FROM weekly_report_archives a WHERE a.primary_division=r.primary_division AND (a.week_start AT TIME ZONE 'Asia/Jakarta')::date=r.report_week)
	 AND to_tsvector('simple', r.work||' '||r.description) @@ q.terms
	 UNION ALL
	 SELECT 'archive:'||a.id||':'||(item->>'id'), 'report', coalesce(item->>'work','Laporan'), concat_ws(E'\n',item->>'workDivision',item->>'work',item->>'description'),
	 '/weekly-progress/history?archive='||a.id||'#report-'||(item->>'id'),
	 ts_rank_cd(to_tsvector('simple', coalesce(item->>'work','')||' '||coalesce(item->>'description','')), q.terms)
	 FROM weekly_report_archives a CROSS JOIN LATERAL jsonb_array_elements(a.snapshot) item CROSS JOIN q
	 WHERE ($4 OR a.primary_division=$5) AND ($4 OR $5<>'')
	 AND to_tsvector('simple', a.snapshot::text) @@ q.terms
	 AND to_tsvector('simple', coalesce(item->>'work','')||' '||coalesce(item->>'description','')) @@ q.terms
	) SELECT id,kind,left(title,200),left(excerpt,1200),url FROM candidates ORDER BY score DESC,id LIMIT 8`
	rows, err := db.Conn.QueryContext(ctx, query, terms, claims.ID, model.CanUseLessonsLearned(claims.RoleName), allReports, division, weekStart.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("Hermes search: %w", err)
	}
	defer rows.Close()
	sources := []HermesSource{}
	for rows.Next() {
		var source HermesSource
		if err := rows.Scan(&source.ID, &source.Kind, &source.Title, &source.Excerpt, &source.URL); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}
