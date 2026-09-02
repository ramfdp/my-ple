package controller

import "testing"

func TestEmploymentStatuses(t *testing.T) {
	for _, status := range []string{"Permanent Employee", "Contract Employee", "Intern"} {
		if !employmentStatus[status] {
			t.Fatalf("%q must be allowed", status)
		}
	}
	if employmentStatus["Probation"] {
		t.Fatal("unrequested status must be rejected")
	}
}

func TestOwnsAvatar(t *testing.T) {
	id := "fb98c49a-8ab7-474b-711d-7f3e2ae3a34f"
	if !ownsAvatar(id, "/uploads/"+id+"-new.jpg") || !ownsAvatar(id, "/uploads/"+id+".jpg") || ownsAvatar(id, "/uploads/other.jpg") {
		t.Fatal("avatar ownership check is incorrect")
	}
}
