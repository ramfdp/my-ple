package db

import (
	"my-ple-backend/model"
	"strings"
)

// Origin refers to a paragraph in the version checked by SaveWorkspacePage.
// Only authors from stored content or the authenticated session are trusted.
func attributePageContent(old, next []model.PageLine, editor model.PageAuthor) {
	used := make([]bool, len(old))
	for i := range next {
		block := &next[i]
		index := block.Origin - 1
		block.Origin = 0
		block.Authors = nil
		if index == -1 {
			// Older clients send no origin. Preserve identical paragraphs when possible.
			for j, prior := range old {
				if !used[j] && prior.Text == block.Text && prior.Kind == block.Kind && prior.URL == block.URL && prior.Checked == block.Checked {
					index = j
					break
				}
			}
		}
		var prior model.PageLine
		if index >= 0 && index < len(old) && !used[index] {
			used[index] = true
			prior = old[index]
		}
		lines := strings.Split(block.Text, "\n")
		authors := make([]model.PageAuthor, len(lines))
		for j := range authors {
			authors[j] = editor
		}
		if prior.Kind == block.Kind && prior.URL == block.URL && prior.Checked == block.Checked {
			before := strings.Split(prior.Text, "\n")
			preserve := func(to, from int) {
				authors[to] = model.PageAuthor{} // Legacy notes have no recorded editor.
				if from < len(prior.Authors) {
					authors[to] = prior.Authors[from]
				}
			}
			left, endOld, endNew := 0, len(before), len(lines)
			for left < endOld && left < endNew && before[left] == lines[left] {
				preserve(left, left)
				left++
			}
			for left < endOld && left < endNew && before[endOld-1] == lines[endNew-1] {
				endOld--
				endNew--
				preserve(endNew, endOld)
			}
			positions := map[string][]int{}
			for j := left; j < endOld; j++ {
				positions[before[j]] = append(positions[before[j]], j)
			}
			for j := left; j < endNew; j++ {
				if matches := positions[lines[j]]; len(matches) > 0 {
					preserve(j, matches[0])
					positions[lines[j]] = matches[1:]
				}
			}
		}
		block.Authors = authors
	}
}
