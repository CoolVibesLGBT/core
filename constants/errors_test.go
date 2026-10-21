package constants_test

import (
	"core/constants"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

func TestEveryErrorCodeHasUniqueCodeAndMessage(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "errors.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[constants.ErrorCode]string)
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		typ, ok := spec.Type.(*ast.Ident)
		if !ok || typ.Name != "ErrorCode" {
			return true
		}
		for i, value := range spec.Values {
			literal, ok := value.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Errorf("%s must have an explicit string code", spec.Names[i].Name)
				continue
			}
			text, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			code := constants.ErrorCode(text)
			if previous, ok := seen[code]; ok {
				t.Errorf("%s and %s share code %q", previous, spec.Names[i].Name, text)
			}
			seen[code] = spec.Names[i].Name
			if message, ok := constants.ErrorMessages[code]; !ok || message == "" {
				t.Errorf("%s has no registered message", spec.Names[i].Name)
			}
		}
		return true
	})
	if len(seen) == 0 {
		t.Fatal("no error codes checked")
	}
	for code := range constants.ErrorMessages {
		if _, ok := seen[code]; !ok {
			t.Errorf("message has no declared error code: %q", string(code))
		}
	}
}

func TestErrorCodeCanBeReturnedAndWrapped(t *testing.T) {
	var err error = constants.ErrPostNotFound
	if err.Error() != "Post not found." {
		t.Fatalf("message = %q", err.Error())
	}
	if !errors.Is(fmt.Errorf("%w", err), constants.ErrPostNotFound) {
		t.Fatal("wrapping lost the error code")
	}
	if got := constants.ErrorCode("UNREGISTERED").Error(); got != constants.ErrUnknown.String() {
		t.Fatalf("unknown message = %q", got)
	}
}

func TestFormattedErrorPreservesCodeCauseAndMessage(t *testing.T) {
	cause := &strconv.NumError{Func: "ParseInt", Num: "bad", Err: strconv.ErrSyntax}
	err := constants.ErrEventDateTimeInvalid.Format(cause)
	if want := "invalid event date or time: " + cause.Error(); err.Error() != want {
		t.Fatalf("message = %q, want %q", err.Error(), want)
	}
	for _, target := range []error{constants.ErrEventDateTimeInvalid, cause, strconv.ErrSyntax} {
		if !errors.Is(err, target) {
			t.Errorf("formatted error lost %v", target)
		}
	}
	var original *strconv.NumError
	if !errors.As(err, &original) || original != cause {
		t.Fatal("formatted error lost the typed cause")
	}
	var code constants.ErrorCode
	if !errors.As(err, &code) || code != constants.ErrEventDateTimeInvalid {
		t.Fatal("formatted error lost the typed code")
	}
	if errors.Is(err, constants.ErrPostNotFound) {
		t.Fatal("formatted error matched an unrelated code")
	}
}

func TestFormattedErrorPreservesWrappedSentinelAndDetails(t *testing.T) {
	err := constants.ErrPostPublicIDLookupNotFound.Format(constants.ErrNotFound, int64(42))
	if err.Error() != "resource not found: post with id 42" {
		t.Fatalf("message = %q", err.Error())
	}
	if !errors.Is(err, constants.ErrNotFound) || !errors.Is(err, constants.ErrPostPublicIDLookupNotFound) {
		t.Fatal("formatted lookup lost its code or wrapped sentinel")
	}
}
