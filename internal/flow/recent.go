package flow

// RecentStore tracks recently opened flows in memory.
// Persistence is handled by the SQLite RecentRepo in internal/db.
type RecentStore struct {
	FlowIDs []string `json:"flowIds"`
}

// Touch moves a flow ID to the front of the recent list.
func (rs *RecentStore) Touch(flowID string) {
	// Remove existing
	var filtered []string
	for _, id := range rs.FlowIDs {
		if id != flowID {
			filtered = append(filtered, id)
		}
	}
	// Add to front
	rs.FlowIDs = append([]string{flowID}, filtered...)
	if len(rs.FlowIDs) > 20 {
		rs.FlowIDs = rs.FlowIDs[:20]
	}
}

// Remove removes a flow ID.
func (rs *RecentStore) Remove(flowID string) {
	var filtered []string
	for _, id := range rs.FlowIDs {
		if id != flowID {
			filtered = append(filtered, id)
		}
	}
	rs.FlowIDs = filtered
}
