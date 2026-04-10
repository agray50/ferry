package store

// DiffResult holds the difference between a local and remote manifest.
type DiffResult struct {
	New     []Component
	Changed []Component
	Removed []Component
	Same    []Component
}

// DiffManifests compares a local manifest against a remote manifest.
func DiffManifests(local, remote *Manifest) DiffResult {
	remoteByID := make(map[string]Component)
	for _, c := range remote.Components {
		remoteByID[c.ID] = c
	}

	localByID := make(map[string]Component)
	for _, c := range local.Components {
		localByID[c.ID] = c
	}

	var result DiffResult

	for _, lc := range local.Components {
		rc, exists := remoteByID[lc.ID]
		if !exists {
			result.New = append(result.New, lc)
		} else if lc.Hash != rc.Hash {
			result.Changed = append(result.Changed, lc)
		} else {
			result.Same = append(result.Same, lc)
		}
	}

	for _, rc := range remote.Components {
		if _, exists := localByID[rc.ID]; !exists {
			result.Removed = append(result.Removed, rc)
		}
	}

	return result
}

// DiffSize returns total compressed bytes for New + Changed components.
func (d DiffResult) DiffSize() int64 {
	var total int64
	for _, c := range d.New {
		total += c.SizeCompressed
	}
	for _, c := range d.Changed {
		total += c.SizeCompressed
	}
	return total
}
