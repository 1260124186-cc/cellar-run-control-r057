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

// Next stamps the subject with the unified business time of the current
// commit. Every successful commit advances the subject, and the commit time
// never moves before the last committed business time.
func (t *EventTimeline) Next(subject string, candidate time.Time) time.Time {
	if t.Entities == nil {
		t.Entities = map[string]time.Time{}
	}
	stamp := candidate.UTC()
	if !t.LastCommitted.IsZero() && !stamp.After(t.LastCommitted) {
		stamp = t.LastCommitted.Add(time.Nanosecond)
	}
	t.record(subject, stamp)
	return stamp
}

// CommitPair stamps two related subjects with one unified commit time so
// their business times cannot diverge or invert within the commit.
func (t *EventTimeline) CommitPair(
	first string,
	second string,
	candidate time.Time,
) (time.Time, time.Time) {
	stamp := t.Next(first, candidate)
	t.record(second, stamp)
	return stamp, stamp
}

func (t *EventTimeline) record(subject string, stamp time.Time) {
	if t.Entities == nil {
		t.Entities = map[string]time.Time{}
	}
	t.Entities[subject] = stamp
	if stamp.After(t.LastCommitted) {
		t.LastCommitted = stamp
	}
}

func (t EventTimeline) Horizon(subject string) time.Time {
	return t.Entities[subject]
}

// RebuildEventTimeline restores the committed business-time horizon from
// confirmed entity timestamps. It never moves a confirmed time: each subject
// keeps its own value and the horizon resumes at the latest one.
func RebuildEventTimeline(values map[string]time.Time) EventTimeline {
	rebuilt := EmptyEventTimeline()
	var latest time.Time
	for subject, value := range values {
		if value.IsZero() {
			continue
		}
		stamp := value.UTC()
		rebuilt.Entities[subject] = stamp
		if stamp.After(latest) {
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
			cloned.Vessels[key] = value
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
