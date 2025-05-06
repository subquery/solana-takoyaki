package sqd

import (
	"testing"
)

func TestSetsDiscriminatorCorrectly(t *testing.T) {

	ir := InstructionRequest{}

	ir.SetDiscriminators([]string{"0xe517cb977ae3ad2a"});

	if len(ir.D8) != 1 {
		t.Fatalf("Expected 1 D8 entry, got %d", len(ir.D8))
	}

	if ir.D8[0] != "0xe517cb977ae3ad2a" {
		t.Fatalf("Expected discriminator 0xe517cb977ae3ad2a, got %s", ir.D8[0])
	}
}

func TestSetsAccountsCorrectly(t *testing.T) {

	ir := InstructionRequest{}

	ir.SetAccounts(1, []string{"0xe517cb977ae3ad2a"});

	if len(ir.A1) != 1 {
		t.Fatalf("Expected 1 A1 entry, got %d", len(ir.A1))
	}

	if ir.A1[0] != "0xe517cb977ae3ad2a" {
		t.Fatalf("Expected discriminator 0xe517cb977ae3ad2a, got %s", ir.A1[0])
	}
}

