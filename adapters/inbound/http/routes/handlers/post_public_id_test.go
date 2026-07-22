package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"core/constants"
	domainpost "core/domain/post"
	"core/models"
	postmodel "core/models/post"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
)

type pollCreationErrorRepository struct {
	ports.PostRepository
	err error
}

func (r *pollCreationErrorRepository) CreateContentablePost(context.Context, ports.FormData, *models.User, string, *uuid.UUID) (*postmodel.Post, error) {
	return nil, r.err
}

func TestParsePollChoiceIDAcceptsOpaqueAndLegacyIdentifiers(t *testing.T) {
	choiceID := uuid.New()
	opaque := types.EncodeOpaqueID("pc", [16]byte(choiceID))

	parsed, err := parsePollChoiceID(opaque)
	if err != nil || parsed != choiceID {
		t.Fatalf("parsePollChoiceID(opaque) = %s, %v", parsed, err)
	}
	parsed, err = parsePollChoiceID(choiceID.String())
	if err != nil || parsed != choiceID {
		t.Fatalf("parsePollChoiceID(legacy) = %s, %v", parsed, err)
	}
	if _, err := parsePollChoiceID("pc_invalid"); err == nil {
		t.Fatal("expected malformed choice token to fail")
	}
}

func TestHandleCreateMapsPollDefinitionErrorsToSafeBadRequest(t *testing.T) {
	tests := []error{
		domainpost.ErrPollQuestionRequired,
		domainpost.ErrPollOptionsRequired,
		domainpost.ErrDuplicatePollOption,
		domainpost.ErrInvalidPollMaximum,
		domainpost.ErrInvalidPollKind,
		domainpost.ErrInvalidPollChoiceData,
	}
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	for _, validationErr := range tests {
		t.Run(validationErr.Error(), func(t *testing.T) {
			repository := &pollCreationErrorRepository{err: validationErr}
			service := usecases.NewPostService(nil, repository, nil)
			response := performMultipartHandlerRequest(t, HandleCreate(service), authUser, map[string]string{"content": "poll"}, nil)
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if err := response.Body.Close(); err != nil {
				t.Fatalf("close body: %v", err)
			}
			if response.StatusCode != 400 {
				t.Fatalf("status = %d, body=%s", response.StatusCode, body)
			}
			var payload struct {
				Code    constants.ErrorCode `json:"code"`
				Message string              `json:"message"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload.Code != constants.ErrInvalidInput || payload.Message != validationErr.Error() {
				t.Fatalf("response = %#v", payload)
			}
		})
	}
}

func TestHandleCreateDoesNotExposeUnexpectedRepositoryErrors(t *testing.T) {
	repository := &pollCreationErrorRepository{err: errors.New("database secret: password=do-not-leak")}
	service := usecases.NewPostService(nil, repository, nil)
	response := performMultipartHandlerRequest(t, HandleCreate(service), &models.User{ID: uuid.New(), PublicID: 10}, map[string]string{"content": "post"}, nil)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}
	if response.StatusCode != 500 {
		t.Fatalf("status = %d, body=%s", response.StatusCode, body)
	}
	if len(body) == 0 || !json.Valid(body) {
		t.Fatalf("invalid response: %s", body)
	}
	if bytes.Contains(body, []byte("do-not-leak")) {
		t.Fatalf("unexpected repository error leaked: %s", body)
	}
}
