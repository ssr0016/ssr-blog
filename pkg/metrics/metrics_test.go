package metrics

import (
	"sort"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func gather(t *testing.T, name string) (labelValues []string, labelNames []string, found bool) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != name {
			continue
		}
		seenNames := map[string]bool{}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				seenNames[lp.GetName()] = true
				labelValues = append(labelValues, lp.GetValue())
			}
		}
		for n := range seenNames {
			labelNames = append(labelNames, n)
		}
		sort.Strings(labelValues)
		return labelValues, labelNames, true
	}
	return nil, nil, false
}

func TestPostWritesTotal_LabelsAreBoundedEnum(t *testing.T) {
	ops := []PostOperation{
		PostOpCreate, PostOpUpdate, PostOpDelete, PostOpPublish, PostOpUnpublish,
	}
	for _, op := range ops {
		PostWritesTotal.WithLabelValues(string(op)).Inc()
	}

	values, names, found := gather(t, "blog_post_writes_total")
	if !found {
		t.Fatal("blog_post_writes_total not registered")
	}
	if len(names) != 1 || names[0] != "operation" {
		t.Errorf("label names = %v, want exactly [operation]", names)
	}

	want := []string{"create", "delete", "publish", "unpublish", "update"}
	if len(values) != len(want) {
		t.Fatalf("label values = %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Errorf("label values = %v, want %v", values, want)
			break
		}
	}
}

func TestPostSlugAttempts_Histogram(t *testing.T) {
	PostSlugAttempts.Observe(1)
	PostSlugAttempts.Observe(3)

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "blog_post_slug_attempts" {
			continue
		}
		m := mf.GetMetric()[0]
		if len(m.GetLabel()) != 0 {
			t.Errorf("histogram must be unlabeled, got %v", m.GetLabel())
		}
		h := m.GetHistogram()
		if h.GetSampleCount() < 2 {
			t.Errorf("sample count = %d, want >= 2", h.GetSampleCount())
		}
		// Must resolve attempts up to the service's 100-attempt cap.
		var top float64
		for _, b := range h.GetBucket() {
			top = b.GetUpperBound()
		}
		if top < 100 {
			t.Errorf("highest finite bucket = %v, want >= 100", top)
		}
		return
	}
	t.Fatal("blog_post_slug_attempts not registered")
}
