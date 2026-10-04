package domain

import "time"

type EventTimeline struct {
	Entities      map[string]time.Time `json:"entities"`
	LastCommitted time.Time            `json:"last_committed"`
}

func EmptyEventTimeline() EventTimeline {
	return EventTimeline{Entities: map[string]time.Time{}}
}

func EventSubject(kind, id string) string {
	return kind + "|" + id
}

func (t EventTimeline) Clone() EventTimeline {
	cloned := EmptyEventTimeline()
	cloned.LastCommitted = t.LastCommitted
	for subject, stamp := range t.Entities {
		cloned.Entities[subject] = stamp
	}
	return cloned
}

// Commit assigns the unified business time of one successful commit to every
// subject the commit touches. The returned stamp never moves backwards: a
// candidate earlier than the last committed business time is advanced to it,
// which keeps operations ordered even when the wall clock stalls or rewinds
// across a restart. All subjects receive the same stamp, so related entities
// (a run and its vessel, for example) can never be ordered against each other,
// and each subject always moves past its previously confirmed time.
func (t *EventTimeline) Commit(subjects []string, candidate time.Time) time.Time {
	if t.Entities == nil {
		t.Entities = map[string]time.Time{}
	}
	stamp := candidate.UTC()
	if !t.LastCommitted.IsZero() && stamp.Before(t.LastCommitted) {
		stamp = t.LastCommitted
	}
	for _, subject := range subjects {
		t.Entities[subject] = stamp
	}
	if stamp.After(t.LastCommitted) {
		t.LastCommitted = stamp
	}
	return stamp
}

// RebuildEventTimeline restores the per-entity business timeline from the
// confirmed timestamps carried by the entities themselves. Every subject
// keeps its own stamp and the committed horizon is the latest one, so loading
// a snapshot never moves a confirmed entity to an earlier time.
func RebuildEventTimeline(values map[string]time.Time) EventTimeline {
	rebuilt := EmptyEventTimeline()
	var latest time.Time
	for subject, value := range values {
		stamp := value.UTC()
		if stamp.IsZero() {
			continue
		}
		rebuilt.Entities[subject] = stamp
		if latest.IsZero() || stamp.After(latest) {
			latest = stamp
		}
	}
	rebuilt.LastCommitted = latest
	return rebuilt
}

type Snapshot struct {
	Revision    int64                      `json:"revision"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	Events      EventTimeline              `json:"event_timeline"`
	Formulas    map[string]Formula         `json:"formulas"`
	Vessels     map[string]Vessel          `json:"vessels"`
	Runs        map[string]FermentationRun `json:"runs"`
	NameIndex   map[string]string          `json:"formula_name_index"`
	VesselIndex map[string]string          `json:"vessel_code_index"`
	RunIndex    map[string]string          `json:"run_code_index"`
}

func EmptySnapshot() Snapshot {
	return Snapshot{
		Revision:    0,
		Events:      EmptyEventTimeline(),
		Formulas:    map[string]Formula{},
		Vessels:     map[string]Vessel{},
		Runs:        map[string]FermentationRun{},
		NameIndex:   map[string]string{},
		VesselIndex: map[string]string{},
		RunIndex:    map[string]string{},
	}
}

func (s Snapshot) Clone() Snapshot {
	cloned := EmptySnapshot()
	cloned.Revision = s.Revision
	cloned.UpdatedAt = s.UpdatedAt
	cloned.Events = s.Events.Clone()
	for key, value := range s.Formulas {
		cloned.Formulas[key] = value.Clone()
	}
	for key, value := range s.Vessels {
		cloned.Vessels[key] = value
		if value.ActiveRunID != nil {
			runID := *value.ActiveRunID
			cloned.Vessels[key] = withActiveRun(value, &runID)
		}
	}
	for key, value := range s.Runs {
		cloned.Runs[key] = value.Clone()
	}
	for key, value := range s.NameIndex {
		cloned.NameIndex[key] = value
	}
	for key, value := range s.VesselIndex {
		cloned.VesselIndex[key] = value
	}
	for key, value := range s.RunIndex {
		cloned.RunIndex[key] = value
	}
	return cloned
}

func withActiveRun(v Vessel, runID *string) Vessel {
	value := v
	if runID == nil {
		value.ActiveRunID = nil
		return value
	}
	copied := *runID
	value.ActiveRunID = &copied
	return value
}
