package to

// BoolPtr returns a pointer to the passed bool.
func BoolPtr(b bool) *bool {
	return &b
}

// Int32Ptr returns a pointer to the passed int32.
func Int32Ptr(i int32) *int32 {
	return &i
}
