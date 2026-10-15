package lookup

func Label(k string) string {
	m := map[string]string{"a": "alpha"}
	return m[k]
}
