package item

import (
	"testing"
)

// Test ID.Parse with various combinations of input data
func TestIDParse(t *testing.T) {
	id := ID{}
	err := id.Parse("1,2,3")
	if err == nil {
		t.Fatalf("id.Parse() with invalid string returned values, expecting error")
	}

	id = ID{}
	err = id.Parse(":svc:name")
	if err != nil {
		t.Fatalf("id.Parse(:svc:name) retuned error, expected correct value")
	}
	if id.Owner != "" {
		t.Fatalf("id.Parse(:svc:name) retuned value in owner, expected empty string")
	}
	if id.Service != "svc" {
		t.Fatalf("id.Parse(:svc:name) retuned incorrect value for service")
	}
	if id.Name != "name" {
		t.Fatalf("id.Parse(:svc:name) retuned incorrect value for name")
	}
}

// Test ParseAssignment with various combinations of input data
func TestParseAssignment(t *testing.T) {
	_, err := ParseAssignment("1,2,3=someval,10")
	if err == nil {
		t.Fatalf("ParseAssignment() with invalid string returned values, expecting error 1")
	}

	_, err = ParseAssignment(":::=someval,10")
	if err == nil {
		t.Fatalf("ParseAssignment() with invalid string returned values, expecting error 2")
	}

	assn, err := ParseAssignment(":s:n=someval,10")
	if err != nil {
		t.Fatalf("ParseAssignment() with valid string returned error")
	}
	if assn.Id.Owner != "" || assn.Id.Service != "s" || assn.Id.Name != "n" {
		t.Fatalf("ParseAssignment() returned incorrect Id field")
	}
	if assn.Value != "someval" {
		t.Fatalf("ParseAssignment() returned incorrect Value field")
	}
	if assn.Expiry == nil {
		t.Fatalf("ParseAssignment() returned missing Expiry field")
	}

}
