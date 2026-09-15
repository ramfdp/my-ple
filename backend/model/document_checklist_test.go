package model

import "testing"

func TestChecklistValidation(t *testing.T) {
	for _, role := range []string{"Admin", "Management", "Project Engineer", "Finance"} {
		if !CanUseDocumentChecklist(role) {
			t.Fatal(role)
		}
	}
	for _, role := range []string{"HR", "IT", "PMO", "Business Development", "", "Keuangan"} {
		if CanUseDocumentChecklist(role) {
			t.Fatal(role)
		}
	}
	for _, in := range []ChecklistConfigInput{
		{InternalCount: -1}, {ExternalCount: -1}, {InternalCount: 1},
		{InternalCount: 1, Signers: []ChecklistSigner{{Type: "internal", Name: "Budi"}}},
		{InternalCount: 1, Signers: []ChecklistSigner{{Type: "wrong", Name: "Budi", Position: "PM"}}},
		{InternalCount: 2, Signers: []ChecklistSigner{{ID: 1, Type: "internal", Name: "Budi", Position: "PM"}, {ID: 1, Type: "internal", Name: "Andi", Position: "PE"}}},
	} {
		if in.Validate() == nil {
			t.Fatalf("accepted invalid config: %+v", in)
		}
	}
	good := ChecklistConfigInput{InternalCount: 1, Signers: []ChecklistSigner{{Type: "internal", Name: " Budi ", Position: " PM "}}}
	if err := good.Validate(); err != nil || good.Signers[0].Name != "Budi" {
		t.Fatal(err)
	}
	for _, p := range []ChecklistItemPatch{{Field: "notes", Value: "ok"}, {Field: "digitalSigned", Checked: false}, {Field: "sentDate", Value: "2026-09-14"}, {Field: "sentDate", Value: ""}} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []ChecklistItemPatch{{Field: "unknown"}, {Field: "documentName", Value: " "}, {Field: "sentDate", Value: "2026-02-30"}, {Field: "signature", SignerID: -1}} {
		if p.Validate() == nil {
			t.Fatal(p)
		}
	}
}
