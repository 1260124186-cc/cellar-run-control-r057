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

func (t *EventTimeline) Next(subject string, candidate time.Time) time.Time {
	if t.Entities == nil {
		t.Entities = map[string]time.Time{}
	}
	candidate = candidate.UTC()
	if !t.LastCommitted.IsZero() && candidate.Before(t.LastCommitted) {
		candidate = t.LastCommitted
	}
	if existing, found := t.Entities[subject]; found && !existing.IsZero() {
		return existing
	}
	t.record(subject, candidate)
	return candidate
}

func (t *EventTimeline) CommitPair(
	first string,
	second string,
	candidate time.Time,
) (time.Time, time.Time) {
	firstTime := t.Next(first, candidate)
	secondTime := candidate.UTC()
	if !t.LastCommitted.IsZero() && secondTime.Before(t.LastCommitted) {
		secondTime = t.LastCommitted
	}
	t.record(second, secondTime)
	return firstTime, secondTime
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

func RebuildEventTimeline(values map[string]time.Time) EventTimeline {
	rebuilt := EmptyEventTimeline()
	var earliest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		if earliest.IsZero() || value.Before(earliest) {
			earliest = value
		}
	}
	if earliest.IsZero() {
		return rebuilt
	}
	for subject, value := range values {
		if !value.IsZero() {
			rebuilt.Entities[subject] = earliest
		}
	}
	rebuilt.LastCommitted = earliest
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
