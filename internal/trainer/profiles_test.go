package trainer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuiltInProfilesRoundTripDeclarativeFormat(t *testing.T) {
	for id, profile := range Profiles() {
		data, err := MarshalProfileJSON(profile)
		if err != nil {
			t.Fatal(id, err)
		}
		loaded, err := LoadProfileJSON(data)
		if err != nil {
			t.Fatal(id, err)
		}
		if loaded.ID != id || string(loaded.UnlockOrder) != string(profile.UnlockOrder) || len(loaded.FrequentPairs) != len(profile.FrequentPairs) || len(CandidateSkills(loaded)) != len(CandidateSkills(profile)) {
			t.Fatalf("%s profile changed on round trip", id)
		}
	}
}

func TestInvalidProfileIdentifiesField(t *testing.T) {
	data, err := MarshalProfileJSON(Profiles()["en"])
	if err != nil {
		t.Fatal(err)
	}
	var doc ProfileDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc.FrequentPairs[0] = "z7"
	data, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProfileJSON(data); err == nil || !strings.Contains(err.Error(), "frequent_pairs[0]") {
		t.Fatalf("invalid field was not identified: %v", err)
	}
}
