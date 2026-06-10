package flow

import (
	"fmt"
	"testing"
)

func TestRecentStoreTouchAndOrder(t *testing.T) {
	rs := &RecentStore{}
	rs.Touch("flow-a")
	rs.Touch("flow-b")
	rs.Touch("flow-a") // move a to front

	if len(rs.FlowIDs) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rs.FlowIDs))
	}
	if rs.FlowIDs[0] != "flow-a" {
		t.Fatalf("expected flow-a first, got %s", rs.FlowIDs[0])
	}
	if rs.FlowIDs[1] != "flow-b" {
		t.Fatalf("expected flow-b second, got %s", rs.FlowIDs[1])
	}
}

func TestRecentStoreLimit(t *testing.T) {
	rs := &RecentStore{}
	for i := 0; i < 25; i++ {
		rs.Touch(fmt.Sprintf("flow-%d", i))
	}
	if len(rs.FlowIDs) != 20 {
		t.Fatalf("expected max 20 entries, got %d", len(rs.FlowIDs))
	}
}

func TestRecentStoreRemove(t *testing.T) {
	rs := &RecentStore{}
	rs.Touch("flow-a")
	rs.Touch("flow-b")
	rs.Remove("flow-a")
	if len(rs.FlowIDs) != 1 {
		t.Fatalf("expected 1 after remove, got %d", len(rs.FlowIDs))
	}
}
