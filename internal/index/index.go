package index

// FindByNodeID returns a pointer to the matching entry or nil.
func (i *Index) FindByNodeID(nodeID string) *Entry {
	for idx := range i.Entries {
		if i.Entries[idx].NodeID == nodeID {
			return &i.Entries[idx]
		}
	}
	return nil
}

// FindByTags returns entries that match at least one requested tag.
func (i *Index) FindByTags(tags []string) []Entry {
	if len(tags) == 0 {
		return nil
	}

	tagSet := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	matched := make([]Entry, 0)
	for _, entry := range i.Entries {
		if hasAnyTag(entry.Tags, tagSet) {
			matched = append(matched, entry)
		}
	}
	return matched
}

// FindRelated traverses depends_on with BFS up to maxDepth.
func (i *Index) FindRelated(nodeID string, maxDepth int) []Entry {
	if maxDepth <= 0 {
		return nil
	}

	byID := make(map[string]Entry, len(i.Entries))
	deps := make(map[string][]string, len(i.Entries))
	for _, entry := range i.Entries {
		byID[entry.NodeID] = entry
		deps[entry.NodeID] = entry.DependsOn
	}

	root, ok := deps[nodeID]
	if !ok {
		return nil
	}

	type qItem struct {
		id    string
		depth int
	}

	queue := make([]qItem, 0, len(root))
	visited := map[string]bool{nodeID: true}
	results := make([]Entry, 0)

	for _, dep := range root {
		queue = append(queue, qItem{id: dep, depth: 1})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true

		entry, exists := byID[current.id]
		if !exists {
			continue
		}
		results = append(results, entry)

		if current.depth >= maxDepth {
			continue
		}
		for _, dep := range deps[current.id] {
			if !visited[dep] {
				queue = append(queue, qItem{id: dep, depth: current.depth + 1})
			}
		}
	}

	return results
}

func hasAnyTag(entryTags []string, tagSet map[string]struct{}) bool {
	for _, tag := range entryTags {
		if _, ok := tagSet[tag]; ok {
			return true
		}
	}
	return false
}
