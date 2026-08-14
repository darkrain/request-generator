package actions

import "testing"

func TestAtomicRecordTypedAccessors(t *testing.T) {
	userID := int64(42)
	record := AtomicRecord{Fields: []AtomicField{
		{Name: "user_id", Value: AtomicInt(userID)},
		{Name: "name", Value: AtomicString("Ada")},
	}}

	actualUserID, ok := record.Int("user_id")
	if !ok || actualUserID != userID {
		t.Fatalf("record.Int(user_id) = (%d, %t), want (%d, true)", actualUserID, ok, userID)
	}
	if _, ok := record.Int("name"); ok {
		t.Fatal("record.Int(name) unexpectedly succeeded")
	}
	if _, ok := record.Int("missing"); ok {
		t.Fatal("record.Int(missing) unexpectedly succeeded")
	}
}
