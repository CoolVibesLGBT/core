package models

import "testing"

func TestReportStatusTransitions(t *testing.T) {
	allowed := [][2]ReportStatus{
		{ReportStatusPending, ReportStatusReviewed},
		{ReportStatusPending, ReportStatusRejected},
		{ReportStatusPending, ReportStatusActioned},
		{ReportStatusReviewed, ReportStatusActioned},
		{ReportStatusActioned, ReportStatusActioned},
	}
	for _, transition := range allowed {
		if !transition[0].CanTransitionTo(transition[1]) {
			t.Fatalf("transition %s -> %s should be allowed", transition[0], transition[1])
		}
	}
	denied := [][2]ReportStatus{
		{ReportStatusRejected, ReportStatusPending},
		{ReportStatusActioned, ReportStatusReviewed},
		{ReportStatusRejected, ReportStatusActioned},
	}
	for _, transition := range denied {
		if transition[0].CanTransitionTo(transition[1]) {
			t.Fatalf("transition %s -> %s should be denied", transition[0], transition[1])
		}
	}
}

func TestIsStandardReportKind(t *testing.T) {
	for _, key := range []string{ReportKindSpam, ReportKindHarassment, ReportKindFakeProfile, ReportKindOther} {
		if !IsStandardReportKind(key) {
			t.Fatalf("expected %q to be a standard report kind", key)
		}
	}
	if IsStandardReportKind("custom-or-free-text") {
		t.Fatal("free text must not be treated as a standard report kind")
	}
}
