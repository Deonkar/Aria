package ai

import "testing"

func TestValidate_ValidSelect(t *testing.T) {
	if err := Validate("SELECT id FROM leads"); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_Insert(t *testing.T) {
	if err := Validate("INSERT INTO leads VALUES (1)"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_Update(t *testing.T) {
	if err := Validate("UPDATE leads SET state='x'"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_Drop(t *testing.T) {
	if err := Validate("DROP TABLE leads"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_CommentInjection(t *testing.T) {
	if err := Validate("SELECT 1-- ; DROP TABLE leads"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_Semicolon(t *testing.T) {
	if err := Validate("SELECT 1; DELETE FROM leads"); err == nil {
		t.Fatal("expected error")
	}
}

func TestInjectFilter_HasPlaceholder(t *testing.T) {
	out, err := InjectAgentFilter("SELECT * FROM leads WHERE assigned_agent_id = :agent_id", "u1", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if want := "SELECT * FROM leads WHERE assigned_agent_id = $1"; out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestInjectFilter_MissingPlaceholder(t *testing.T) {
	out, err := InjectAgentFilter("SELECT id FROM leads", "u1", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if want := "SELECT id FROM leads WHERE assigned_agent_id = $1"; out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestInjectFilter_Admin(t *testing.T) {
	out, err := InjectAgentFilter("SELECT count(*) FROM leads", "u1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if want := "SELECT count(*) FROM leads"; out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}
