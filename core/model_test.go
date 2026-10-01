package core

import "testing"

func TestEvaluateUnknownAndBoundaries(t *testing.T) {
	yes, no := true, false
	rules := Rules{FreeOnly: true, ExcludeHR: true, MinBytes: 100, MaxBytes: 200, Include: "movie", Exclude: "cam"}
	rows := []Candidate{
		{"unknown", "Movie", 100, nil, &no},
		{"hr-unknown", "Movie", 100, &yes, nil},
		{"hr", "Movie", 100, &yes, &yes},
		{"lower", "MOVIE", 100, &yes, &no},
		{"upper", "Movie", 200, &yes, &no},
		{"large", "Movie", 201, &yes, &no},
		{"blocked", "Movie.CAM", 150, &yes, &no},
		{"lower", "Movie", 100, &yes, &no},
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
