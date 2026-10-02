package core

import "testing"

func TestEvaluateUnknownAndBoundaries(t *testing.T) {
	yes, no := true, false
	rules := Rules{FreeOnly: true, ExcludeHR: true, MinBytes: 100, MaxBytes: 200, Include: "movie", Exclude: "cam"}
	rows := []Candidate{
		{ID: "unknown", Title: "Movie", SizeBytes: 100, Free: nil, HR: &no},
		{ID: "hr-unknown", Title: "Movie", SizeBytes: 100, Free: &yes, HR: nil},
		{ID: "hr", Title: "Movie", SizeBytes: 100, Free: &yes, HR: &yes},
		{ID: "lower", Title: "MOVIE", SizeBytes: 100, Free: &yes, HR: &no},
		{ID: "upper", Title: "Movie", SizeBytes: 200, Free: &yes, HR: &no},
		{ID: "large", Title: "Movie", SizeBytes: 201, Free: &yes, HR: &no},
		{ID: "blocked", Title: "Movie.CAM", SizeBytes: 150, Free: &yes, HR: &no},
		{ID: "lower", Title: "Movie", SizeBytes: 100, Free: &yes, HR: &no},
	}
	got, err := Evaluate(rules, rows)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"promotion_unknown", "hr_unknown", "hit_and_run", "matched", "matched", "size_out_of_range", "excluded", "duplicate"}
	for i, d := range got {
		if d.Reason != want[i] || d.Accepted != (want[i] == "matched") {
			t.Errorf("row %d: %+v", i, d)
		}
	}
}
func TestInvalidRulesAndOversizePreview(t *testing.T) {
	for _, r := range []Rules{{Include: "["}, {Include: "(?<=movie)"}, {MinBytes: 2, MaxBytes: 1}, {MinBytes: -1}} {
		if _, err := Evaluate(r, nil); err == nil {
			t.Fatalf("accepted invalid rule: %+v", r)
		}
	}
	if _, err := Evaluate(Rules{}, make([]Candidate, 101)); err == nil {
		t.Fatal("unbounded preview")
	}
}
func TestDuplicateTaskRejected(t *testing.T) {
	task := Task{ID: "a", Name: "A", BrushMinutes: 10, CheckMinutes: 5}
	if err := (Config{SchemaVersion: 1, Tasks: []Task{task, task}}).Validate(); err == nil {
		t.Fatal("duplicate task accepted")
	}
}
