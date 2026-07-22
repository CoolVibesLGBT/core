package moderation

import (
	"errors"
	"strings"
	"testing"
)

func TestReportValueObjectsNormalizeAndValidate(t *testing.T) {
	kind, err := NewKind("  custom_kind  ")
	if err != nil || kind.String() != "custom_kind" {
		t.Fatalf("NewKind() = %q, %v", kind, err)
	}
	if _, err := NewKind("   "); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("empty kind error = %v", err)
	}
	if _, err := NewKind(strings.Repeat("ğ", MaxKindLength+1)); !errors.Is(err, ErrKindTooLong) || !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("long kind error = %v", err)
	}

	description, err := NewDescription("  details  ")
	if err != nil || description.String() != "details" {
		t.Fatalf("NewDescription() = %q, %v", description, err)
	}
	if _, err := NewDescription(strings.Repeat("ş", MaxDescriptionLength+1)); !errors.Is(err, ErrDescriptionTooLong) {
		t.Fatalf("long description error = %v", err)
	}

	if !IsStandardKind(string(KindSpam)) || IsStandardKind("custom_kind") {
		t.Fatal("standard-kind classification is incorrect")
	}
}

func TestReportAggregateValidatesTargetReporterAndStartsPending(t *testing.T) {
	target, err := NewTarget(TargetUser, 42)
	if err != nil {
		t.Fatalf("NewTarget() error = %v", err)
	}
	kind, _ := NewKind(" fake_profile ")
	description, _ := NewDescription(" copied profile ")
	report, err := NewReport(target, kind, description)
	if err != nil {
		t.Fatalf("NewReport() error = %v", err)
	}
	if report.Status() != StatusPending || report.Target().PublicID() != 42 || report.Kind() != KindFakeProfile {
		t.Fatalf("report = %#v", report)
	}
	if err := report.ValidateReporter(42); !errors.Is(err, ErrCannotReportSelf) {
		t.Fatalf("self-report error = %v", err)
	}
	if err := report.ValidateReporter(7); err != nil {
		t.Fatalf("different reporter error = %v", err)
	}
	if _, err := NewTarget(TargetType("message"), 42); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("invalid target type error = %v", err)
	}
	if _, err := NewTarget(TargetPost, 0); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("invalid target id error = %v", err)
	}
}

func TestReportStatusTransitions(t *testing.T) {
	target, _ := NewTarget(TargetPost, 5)
	kind, _ := NewKind("spam")
	description, _ := NewDescription("")
	report, err := RestoreReport(target, kind, description, StatusPending)
	if err != nil {
		t.Fatalf("RestoreReport() error = %v", err)
	}
	if err := report.TransitionTo(StatusReviewed); err != nil {
		t.Fatalf("pending -> reviewed: %v", err)
	}
	if err := report.TransitionTo(StatusActioned); err != nil {
		t.Fatalf("reviewed -> actioned: %v", err)
	}
	if err := report.TransitionTo(StatusActioned); err != nil {
		t.Fatalf("idempotent actioned -> actioned: %v", err)
	}
	if err := report.TransitionTo(StatusPending); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("actioned -> pending error = %v", err)
	}
	if Status("unknown").CanTransitionTo(Status("unknown")) {
		t.Fatal("an invalid status must never transition")
	}
}

func TestValidateResolution(t *testing.T) {
	hide, show := false, true
	tests := []struct {
		name    string
		status  Status
		publish *bool
		wantErr error
	}{
		{name: "review only", status: StatusReviewed},
		{name: "hide and action", status: StatusActioned, publish: &hide},
		{name: "restore and reject", status: StatusRejected, publish: &show},
		{name: "pending is not a resolution", status: StatusPending, wantErr: ErrInvalidStatus},
		{name: "hide without action", status: StatusRejected, publish: &hide, wantErr: ErrInvalidResolutionAction},
		{name: "show while actioning", status: StatusActioned, publish: &show, wantErr: ErrInvalidResolutionAction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResolution(tt.status, tt.publish)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateResolution() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
