package reggol

type Fields map[string]any

func (f *Fields) Add(key string, value any) *Fields {
	(*f)[key] = value

	return f
}
